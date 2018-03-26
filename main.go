package main

import (
	"github.com/gorilla/mux"
	"net/http"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/eveonline"
	"fmt"

	"github.com/gorilla/sessions"
	"github.com/markbates/goth/providers/discord"
	"github.com/bwmarrin/discordgo"
	"time"
	"github.com/spf13/viper"
	"github.com/asdine/storm"
)

// Global vars
var (
	// Pointer to discord session.
	dg           *discordgo.Session
	// Pointer to db connection.
	db           *storm.DB
	// Pointer to gorilla session storage.
	sessionStore *sessions.FilesystemStore
)

func main() {
	var err error

	// Open db
	// https://github.com/asdine/storm#open-a-database
	db, err = storm.Open("my.db")
	if err != nil {
		panic("error opening database: " + err.Error())
	}
	defer db.Close()

	// Read config file
	// https://github.com/spf13/viper#reading-config-files
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err = viper.ReadInConfig()
	if err != nil {
		panic("error reading config file: " + err.Error())
	}

	// Create discord session
	// https://github.com/bwmarrin/discordgo/tree/master/examples/pingpong
	dg, err = discordgo.New("Bot " + viper.GetString("DiscordToken"))
	if err != nil {
		panic("error creating Discord session: " + err.Error())
	}
	err = dg.Open()
	if err != nil {
		panic("error opening Discord websocket: " + err.Error())
	}
	defer dg.Close()

	sessionStore = sessions.NewFilesystemStore("",
		[]byte(viper.GetString("SessionStoreKey")))

	// https://github.com/markbates/goth/blob/master/examples/main.go
	goth.UseProviders(
		eveonline.New(
			viper.GetString("EveClientId"),
			viper.GetString("EveClientSecret"),
			viper.GetString("EveCallback"),
			[]string{"publicData"}...),
		discord.New(
			viper.GetString("DiscordClientKey"),
			viper.GetString("DiscordClientSecret"),
			viper.GetString("DiscordCallback"),
			[]string{"guilds.join"}...),
	)
	// https://github.com/markbates/goth#security-notes
	gothicStore := sessions.NewCookieStore(
		[]byte(viper.GetString("GothicStoreKey")))
	gothic.Store = gothicStore

	// Perform check for access rights every 10 minutes in a separate goroutine.
	// Change 10 or Minute to Second to proc it more often.
	go func() {
		EveCharAffiliation()
		for range time.NewTicker(10 * time.Minute).C {
			EveCharAffiliation()
		}
	}()

	// https://github.com/gorilla/mux#examples
	r := mux.NewRouter()
	r.HandleFunc("/", GetIndexHandler)
	r.HandleFunc("/auth/{provider}", gothic.BeginAuthHandler).Methods("GET")
	r.HandleFunc("/auth/{provider}/callback", GetCallbackHandler).Methods("GET")
	r.HandleFunc("/logout", GetLogoutHandler).Methods("GET")

	err = http.ListenAndServe(":8080", r)
	if err != nil {
		fmt.Errorf("%s", err)
	}
}
