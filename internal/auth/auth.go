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

	auth := db.Auth{
		Username: username,
		Pw:       string(passwdHash),
	}

	db.DB.Create(auth)

	if err = createUserDir(username); err != nil {
		return err
	}
	return nil
}

func ValidateUser(username, password string) bool {

	username = strings.ToLower(username)

	var auth db.Auth
	db.DB.First(&auth, "username = ?", username)

	if err := bcrypt.CompareHashAndPassword([]byte(auth.Pw), []byte(password)); err != nil {
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
