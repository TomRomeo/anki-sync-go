package db

import (
	"database/sql"
	"fmt"
	"github.com/gin-contrib/sessions/postgres"
	_ "github.com/lib/pq"
	"log"
	"os"
)

var DB *sql.DB
var Store postgres.Store

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
	db.Exec(`
		CREATE TABLE IF NOT EXISTS auth (
			username VARCHAR(255) NOT NULL PRIMARY KEY,
			pw VARCHAR(64) NOT NULL);
	`)

	store, err := postgres.NewStore(db, []byte("secret"))
	if err != nil {
		log.Fatal(err)
	}
	Store = store
}
