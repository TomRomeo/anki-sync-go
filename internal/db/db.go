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

	// TODO: periodic cleaning job
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
	db.Exec(`
		-- Cards are what you review. 
		-- There can be multiple cards for each note, as determined by the Template.
		CREATE TABLE IF NOT EXISTS cards (
		    username VARCHAR(255) NOT NULL,
		    id              BIGINT,
		      -- the epoch milliseconds of when the card was created
		    nid             BIGINT not null,--    
		      -- notes.id
		    did             BIGINT not null,
		      -- deck id (available in col table)
		    ord             integer not null,
		      -- ordinal : identifies which of the card templates or cloze deletions it corresponds to 
		      --   for card templates, valid values are from 0 to num templates - 1
		      --   for cloze deletions, valid values are from 0 to max cloze index - 1 (they're 0 indexed despite the first being called 'c1')
		    mod             BIGINT not null,
		      -- modificaton time as epoch seconds
		    usn             integer not null,
		      -- update sequence number : used to figure out diffs when syncing. 
		      --   value of -1 indicates changes that need to be pushed to server. 
		      --   usn < server usn indicates changes that need to be pulled from server.
		    type            integer not null,
		      -- 0=new, 1=learning, 2=review, 3=relearning
		    queue           integer not null,
		      -- -3=user buried(In scheduler 2),
		      -- -2=sched buried (In scheduler 2), 
		      -- -2=buried(In scheduler 1),
		      -- -1=suspended,
		      -- 0=new, 1=learning, 2=review (as for type)
		      -- 3=in learning, next rev in at least a day after the previous review
		      -- 4=preview
		    due             BIGINT not null,
		     -- Due is used differently for different card types: 
		     --   new: note id or random int
		     --   due: integer day, relative to the collection's creation time
		     --   learning: integer timestamp in second
		    ivl             integer not null,
		      -- interval (used in SRS algorithm). Negative = seconds, positive = days
		    factor          integer not null,
		      -- The ease factor of the card in permille (parts per thousand). If the ease factor is 2500, the card’s interval will be multiplied by 2.5 the next time you press Good.
		    reps            integer not null,
		      -- number of reviews
		    lapses          integer not null,
		      -- the number of times the card went from a "was answered correctly" 
		      --   to "was answered incorrectly" state
		    "left"            BIGINT not null,
		      -- of the form a*1000+b, with:
		      -- a the number of reps left today
		      -- b the number of reps left till graduation
		      -- for example: '2004' means 2 reps left today and 4 reps till graduation
		    odue            BIGINT not null,
		      -- original due: In filtered decks, it's the original due date that the card had before moving to filtered.
		                    -- If the card lapsed in scheduler1, then it's the value before the lapse. (This is used when switching to scheduler 2. At this time, cards in learning becomes due again, with their previous due date)
		                    -- In any other case it's 0.
		    odid            BIGINT not null,
		      -- original did: only used when the card is currently in filtered deck
		    flags           integer not null,
		      -- an integer. This integer mod 8 represents a "flag", which can be see in browser and while reviewing a note. Red 1, Orange 2, Green 3, Blue 4, no flag: 0. This integer divided by 8 represents currently nothing
		    data            text not null,
		      -- currently unused
		    PRIMARY KEY (username, id)
	);
`)
}
