package auth

import (
	"ankiSyncGo/internal/db"
	"golang.org/x/crypto/bcrypt"
	"os"
	"path"
	"strings"
)

func AddUser(username, password string) error {

	username = strings.ToLower(username)

	passwdHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.DB.Exec("INSERT INTO auth (username, pw) VALUES ($1, $2)", username, passwdHash)
	if err != nil {
		return err
	}

	if err = createUserDir(username); err != nil {
		return err
	}
	return nil
}

func ValidateUser(username, password string) bool {

	username = strings.ToLower(username)

	row := db.DB.QueryRow("SELECT pw FROM auth WHERE username=$1", username)
	var dbPasswd string
	if err := row.Scan(&dbPasswd); err != nil {
		return false
	}
	if err := bcrypt.CompareHashAndPassword([]byte(dbPasswd), []byte(password)); err != nil {
		return false
	}
	return true
}

func createUserDir(username string) error {

	userDir := path.Join(os.Getenv("ROOT_DIR"), os.Getenv("COLLECTION_DIR"), username)

	if err := os.MkdirAll(userDir, 644); err != nil {
		return err
	}
	return nil
}
