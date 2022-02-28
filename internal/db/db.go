package db

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"os"
)

var DB *sql.DB

var (
	host     string
	port     string
	username string
	password string
	dbname   string
)

func Initialize() {

	host = os.Getenv("POSTGRES_HOST")
	port = os.Getenv("POSTGRES_PORT")
	username = os.Getenv("POSTGRES_USER")
	password = os.Getenv("POSTGRES_PASSWORD")
	dbname = os.Getenv("POSTGRES_DB")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		username, password, host, port, dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}
	DB = db

	// initialize the databases, if they do not exist

	db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)
	db.Exec(`
		CREATE TABLE IF NOT EXISTS auth (
			username VARCHAR(255) NOT NULL PRIMARY KEY,
			pw VARCHAR(64) NOT NULL);
	`)

	db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			skey uuid DEFAULT uuid_generate_v4 () PRIMARY KEY ,
			username VARCHAR(255) UNIQUE NOT NULL,
		    created TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
	`)

	db.Exec(`
		-- col contains a single row that holds various information about the collection
		CREATE TABLE IF NOT EXISTS col (
		    username		VARCHAR(255) UNIQUE NOT NULL,
		    id              integer primary key,
		      -- arbitrary number since there is only one row
		    crt             integer not null,
		      -- timestamp of the creation date in second. It's correct up to the day. For V1 scheduler, the hour corresponds to starting a new day. By default, new day is 4.
		    mod             integer not null,
		      -- last modified in milliseconds
		    scm             integer not null,
		      -- schema mod time: time when "schema" was modified. 
		      --   If server scm is different from the client scm a full-sync is required
		    ver             integer not null,
		      -- version
		    dty             integer not null,
		      -- dirty: unused, set to 0
		    usn             integer not null,
		      -- update sequence number: used for finding diffs when syncing. 
		      --   See usn in cards table for more details.
		    ls              integer not null,
		      -- "last sync time"
		    conf            text not null,
		      -- json object containing configuration options that are synced. Described below in "configuration JSONObjects"
		    models          text not null,
		      -- json object of json object(s) representing the models (aka Note types) 
		      -- keys of this object are strings containing integers: "creation time in epoch milliseconds" of the models
		      -- values of this object are other json objects of the form described below in "Models JSONObjects"
		    decks           text not null,
		      -- json object of json object(s) representing the deck(s)
		      -- keys of this object are strings containing integers: "deck creation time in epoch milliseconds" for most decks, "1" for the default deck
		      -- values of this object are other json objects of the form described below in "Decks JSONObjects"
		    dconf           text not null,
		      -- json object of json object(s) representing the options group(s) for decks
		      -- keys of this object are strings containing integers: "options group creation time in epoch milliseconds" for most groups, "1" for the default option group
		      -- values of this object are other json objects of the form described below in "DConf JSONObjects"
		    tags            text not null
		      -- a cache of tags used in the collection (This list is displayed in the browser. Potentially at other place)
	);
`)
}
