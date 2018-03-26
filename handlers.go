package main

import (
	"net/http"
	"html/template"
	"fmt"
	"github.com/markbates/goth/gothic"
	"github.com/gorilla/mux"
	"github.com/asdine/storm"
	"strconv"
	"github.com/spf13/viper"
	"encoding/json"
	"github.com/markbates/goth"
	"bytes"
)

var (
	templates map[string]*template.Template
)

func init() {
	if templates == nil {
		templates = make(map[string]*template.Template)
	}
	templates["index"] = template.Must(
		template.New("index.html").ParseFiles("static/index.html"))
}

// getUserFromSession tries to get an eve char name from a cookie and return a
// User record associated with that name.
func getUserFromSession(r *http.Request) (User, error) {
	// https://github.com/gorilla/sessions#sessions
	session, _ := sessionStore.Get(r, "session")
	var user User
	var err error
	if name, ok := session.Values["EveCharName"]; ok {
		//err = db.One("EveCharName", name, &user)
		user, err = dbGetUserByEveCharName(name.(string))
	} else {
		user = User{}
	}
	// if User record is not found, then neglect the error.
	if err == storm.ErrNotFound {
		err = nil
	}

	return user, err
}

// GetIndexHandler serves the index page.
func GetIndexHandler(w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromSession(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// https://golang.org/pkg/html/template/#Template.ExecuteTemplate
	err = templates["index"].ExecuteTemplate(w, "index.html", user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetCallbackHandler is a callback handler for different auth providers.
// It's intended to let goth finish auth and apply extra logic depending on
// provider.
func GetCallbackHandler(w http.ResponseWriter, r *http.Request) {
	// https://github.com/markbates/goth/blob/master/examples/main.go#L193
	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	vars := mux.Vars(r)
	switch vars["provider"] {
	case "eveonline":
		err := eveCallback(user, w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	case "discord":
		err := discordCallback(user, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	http.Error(w, "Not Found", http.StatusNotFound)
}

// GetLogoutHandler zeroes session values and expires cookie if any.
func GetLogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := sessionStore.Get(r, "session")
	session.Options.MaxAge = -1
	session.Values = make(map[interface{}]interface{})
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// getEveCharPublicData is a helper func. Provided with an eve char id, it tries
// to fetch public data from esi.
func getEveCharPublicData(charID int) (map[string]interface{}, error) {
	var publicData map[string]interface{}

	id := strconv.Itoa(charID)
	res, err := http.Get("https://esi.tech.ccp.is/latest/characters/" + id)
	if err != nil {
		return publicData, err
	}

	d := json.NewDecoder(res.Body)
	// esi returns long int. By default, it's converted to float, so UseNumber.
	// https://stackoverflow.com/questions/22343083/json-marshaling-with-long-numbers-in-golang-gives-floating-point-number
	d.UseNumber()
	d.Decode(&publicData)
	res.Body.Close()

	return publicData, err
}

// getEveCharPublicData is a helper func. Provided with eve char id and map
// containing corporation_id and alliance_id, it these values to ones in config
// file. Returns true if any match found.
func hasAccessEveChar(charID int, publicData map[string]interface{}) (bool, error) {
	var allowedCharID []int
	var allowedCorpId []int
	var allowedAlliId []int
	viper.UnmarshalKey("AllowedCharId", &allowedCharID)
	viper.UnmarshalKey("AllowedCorpId", &allowedCorpId)
	viper.UnmarshalKey("AllowedAlliId", &allowedAlliId)

	for _, chID := range allowedCharID {
		if chID == charID {
			return true, nil
		}
	}
	for _, corpID := range allowedCorpId {
		if corpID == publicData["corporation_id"] {
			return true, nil
		}
	}
	for _, alliID := range allowedAlliId {
		if alliID == publicData["alliance_id"] {
			return true, nil
		}
	}

	return false, nil
}

// getEveCharPublicData is a helper func. Provided with eve corp id, it tries
// to fetch corp data from esi and return corp ticker.
func getEveCorpTicker(corpID int) (string, error) {
	var corpData map[string]interface{}

	id := strconv.Itoa(corpID)
	res, err := http.Get("https://esi.tech.ccp.is/latest/corporations/" + id)
	if err != nil {
		return "", err
	}

	d := json.NewDecoder(res.Body)
	d.UseNumber()
	d.Decode(&corpData)
	res.Body.Close()

	return corpData["ticker"].(string), err
}

// eveCallback is a helper func. Provided with eve char data, it determines
// whether char has access. In case of access granted, it creates cookie and
// tries to create a user record if not exists already.
func eveCallback(user goth.User, w http.ResponseWriter, r *http.Request) error {
	userID, _ := strconv.Atoi(user.UserID)
	publicData, err := getEveCharPublicData(userID)
	if err != nil {
		return err
	}
	hasAccess, err := hasAccessEveChar(userID, publicData)
	if err != nil {
		return err
	}

	if hasAccess {
		session, _ := sessionStore.Get(r, "session")
		session.Values["EveCharName"] = user.NickName
		session.Save(r, w)

		_, err := dbGetUserByEveCharName(user.NickName)
		// ErrNotFound means there is no record in db and one should be created.
		if err == storm.ErrNotFound {
			corpID, _ := publicData["corporation_id"].(json.Number).Int64()
			ticker, _ := getEveCorpTicker(int(corpID))
			err := dbCreateEveUser(userID, user.NickName, ticker)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// discordCallback is a helper func. Provided with discord user data, it
// determines whether char has access. In case of access granted, it creates
// cookie and tries to create a user record if not exists already.
func discordCallback(user goth.User, r *http.Request) error {
	dbUser, err := getUserFromSession(r)
	// Empty eve char name means there is no associated eve char with this
	// request which should not happen. In case it did happen, just return.
	if dbUser.EveCharName == "" {
		return nil
	}

	discordName := fmt.Sprintf("[%s] %s", dbUser.EveCharCorpTicker,
		dbUser.EveCharName)

	// Update User record with discord data.
	userID, _ := strconv.Atoi(user.UserID)
	dbUser.DiscordUserID = userID
	dbUser.DiscordUserName = discordName
	db.Save(&dbUser)

	// Add discord user to guild specified in config.
	// https://discordapp.com/developers/docs/resources/guild#add-guild-member
	endpointGuildAddMember := "https://discordapp.com/api/v6/guilds/" +
		viper.GetString("DiscordGuildId") + "/members/" + user.UserID
	data := struct {
		AccessToken string `json:"access_token"`
		Nick        string `json:"nick,omitempty"`
	}{user.AccessToken, discordName}
	_, err = dg.Request("PUT", endpointGuildAddMember, data)
	if err != nil {
		return err
	}

	return nil
}

// EveCharAffiliation is a helper func. Perform eve chars affiliation. Compare
// each result from esi with allowed char id, corp id, alli id. In case of
// no a single match delete associated user from discord
func EveCharAffiliation() {
	charIds, err := dbGetUsersIdsList()
	if err != nil {
	}
	var affData []map[string]interface{}
	jsonCharIds, _ := json.Marshal(charIds)
	// https://esi.tech.ccp.is/ui/#/Character/post_characters_affiliation
	res, err := http.Post("https://esi.tech.ccp.is/latest/characters/affiliation/",
		"application/json",
		bytes.NewBuffer(jsonCharIds))
	if err != nil {
		return
	}

	d := json.NewDecoder(res.Body)
	d.UseNumber()
	d.Decode(&affData)
	res.Body.Close()

	for _, data := range affData {
		charID, _ := data["character_id"].(json.Number).Int64()
		hasAccess, err := hasAccessEveChar(int(charID), data)
		if err != nil {
			continue
		}
		if !hasAccess {
			user, err := dbGetUserByEveCharId(int(charID))
			if err != nil {
				continue
			}
			// https://discordapp.com/developers/docs/resources/guild#remove-guild-member
			err = dg.GuildMemberDelete(viper.GetString("DiscordGuildId"),
				strconv.Itoa(user.DiscordUserID))
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}
