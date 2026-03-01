package common

import (
	"database/sql"
	"fmt"

	"github.com/bwmarrin/discordgo"

	"github.com/aarondl/sqlboiler/v4/boil"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

var (
	VERSION = "v0.0.0"
	PQ      *sql.DB

	SuccessGreen = 0x00ff7b
	ErrorRed     = 0xFF0000

	Session *discordgo.Session
	Bot     *discordgo.User
)

var CoreSchema = []string{`
CREATE TABLE IF NOT EXISTS core_config (
	guild_id TEXT PRIMARY KEY,
	guild_prefix TEXT DEFAULT '~' NOT NULL
);
`, `
CREATE TABLE IF NOT EXISTS banned_guilds (
	guild_id TEXT PRIMARY KEY
);
`,
}

// Init creates a new discord session, attempts to connect to the postgres database, and
// intialises all the databases
func Init() error {
	s, err := discordgo.New(ConfigDgoBotToken())
	if err != nil {
		log.WithError(err).Fatal()
	}
	Session = s

	db := "summit"
	if ConfigPGDB != "" {
		db = ConfigPGDB
	}
	host := "localhost"
	if ConfigPGHost != "" {
		host = ConfigPGHost
	}
	port := "5432"
	if ConfigPGPort != "" {
		port = ConfigPGPort
	}

	err = postgresConnect(db, host, port, ConfigPGUsername, ConfigPGPassword)
	if err != nil {
		log.WithError(err).Fatal("Failed to connect to database")
	}

	log.Infof("Initializing DB schema")
	InitSchema("Core", CoreSchema...)

	return nil
}

// Run connects the bot account to the Discord session
func Run(s *discordgo.Session) {
	s.Open()
	Bot = s.State.User
	log.Infoln("Bot is now running. Press CTRL-C to exit.")
}

// postgresConnect opens the database connection and sets the global variables to be accessible
func postgresConnect(database string, host string, port string, username string, password string) error {
	if host == "" {
		host = "localhost"
	}

	if password != "" {
		password = " password='" + password + "'"
	}

	// Initialise database
	db, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable%s", host, port, username, database, password))
	if err != nil {
		return err
	}

	PQ = db
	boil.SetDB(PQ)

	return nil
}

// InitSchema initialises the schemas passed to the bot
func InitSchema(schemaname string, schemas ...string) {
	for _, schema := range schemas {
		_, err := PQ.Exec(schema)
		if err != nil {
			log.WithError(err).Fatal("Failed initializing postgres db schema for " + schemaname)
		}
	}
	log.Infoln("Schema " + schemaname + " initialized")
}
