package main

type User struct {
	ID int `storm:"id,increment"`

	EveCharID         int    `storm:"index"`
	EveCharName       string `storm:"index"`
	EveCharCorpTicker string

	DiscordUserID   int
	DiscordUserName string
}

// dbGetUserByEveCharName returns a User record associated with provided
// eve character name.
func dbGetUserByEveCharName(eveCharName string) (User, error) {
	var user User
	err := db.One("EveCharName", eveCharName, &user)
	return user, err
}

// dbGetUserByEveCharId returns a User record associated with provided
// eve character id.
func dbGetUserByEveCharId(eveCharId int) (User, error) {
	var user User
	err := db.One("EveCharID", eveCharId, &user)
	return user, err
}

// dbGetUsersIdsList returns a slice containing EveCharIDs of all registered
// users.
func dbGetUsersIdsList() ([]int, error) {
	var users []User
	var err error
	var ids []int

	err = db.All(&users)
	if err != nil {
		return ids, err
	}

	for _, u := range users {
		ids = append(ids, u.EveCharID)
	}

	return ids, nil
}

// dbCreateEveUser creates a User record from eve data.
func dbCreateEveUser(eveCharId int, eveCharName, eveCharCorpTicker string) error {
	user := User{
		EveCharID:         eveCharId,
		EveCharName:       eveCharName,
		EveCharCorpTicker: eveCharCorpTicker,
	}
	err := db.Save(&user)
	if err != nil {
		return err
	}
	return nil
}
