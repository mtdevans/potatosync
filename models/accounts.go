package models

import (
	"context"
	u "github.com/ATechnoHazard/potatonotes-api/utils"
	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
	"log"
	"os"
	"strings"
)

type Token struct {
	UserId uint
	jwt.StandardClaims
}

// Represents a user account
type Account struct {
	gorm.Model
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `gorm:"-" json:"token"`
}

// Validate incoming user details
func (acc *Account) Validate() (map[string]interface{}, bool) {
	if !strings.Contains(acc.Email, "@") {
		return u.Message(false, "Missing/Malformed email"), false
	}

	if len(acc.Username) <= 4 || len(acc.Username) > 60 {
		return u.Message(false, "Username length not in bounds"), false
	}

	if len(acc.Password) < 8 || len(acc.Password) > 60 {
		return u.Message(false, "Password length not in bounds"), false
	}

	acc.Username = strings.ToLower(acc.Username)

	temp := &Account{}

	// check for duplicate email and username
	err := GetDB().Where("email = ?", acc.Email).First(temp).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return u.Message(false, "Connection error, try again"), false
	}
	if temp.Email != "" {
		return u.Message(false, "Email address already in use!"), false
	}

	err = GetDB().Where("username = ?", acc.Username).First(temp).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return u.Message(false, "Connection error, try again"), false
	}
	if temp.Username != "" {
		return u.Message(false, "Username already in use!"), false
	}

	return u.Message(true, "Validated successfully"), true
}

func (acc *Account) Create() map[string]interface{} {
	if res, ok := acc.Validate(); !ok {
		return res
	}

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(acc.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalln(err)
	}
	acc.Password = string(hashedPass)

	GetDB().Create(acc)

	if acc.ID <= 0 {
		return u.Message(false, "Failed to create account, connection error")
	}

	// create a jwt token for the newly registered account
	tk := &Token{UserId: acc.ID}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tk)
	tokenString, err := token.SignedString([]byte(os.Getenv("token_password")))
	if err != nil {
		log.Fatalln(err)
	}
	acc.Token = tokenString

	acc.Password = "" // delete password

	response := u.Message(true, "Account has been created")
	response["account"] = acc
	return response
}

func Login(email, pass string) map[string]interface{} {
	acc := &Account{}
	err := GetDB().Where("email = ?", email).First(acc).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return u.Message(false, "Email address not found")
		}
		return u.Message(false, "Connection error, try again")
	}

	err = bcrypt.CompareHashAndPassword([]byte(acc.Password), []byte(pass))
	if err != nil && err == bcrypt.ErrMismatchedHashAndPassword { // password doesn't match
		return u.Message(false, "Invalid login credentials")
	}

	// login successful
	acc.Password = ""

	// create jwt token
	tk := &Token{UserId: acc.ID}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tk)
	tokenString, err := token.SignedString([]byte(os.Getenv("token_password")))
	if err != nil {
		log.Fatalln(err)
	}
	acc.Token = tokenString
	res := u.Message(true, "Login successful")
	res["account"] = acc
	return res
}

func LoginUsername(username, pass string) map[string]interface{} {
	acc := &Account{}
	err := GetDB().Where("username = ?", strings.ToLower(username)).First(acc).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return u.Message(false, "Username not found")
		}
		return u.Message(false, "Connection error, try again")
	}

	err = bcrypt.CompareHashAndPassword([]byte(acc.Password), []byte(pass))
	if err != nil && err == bcrypt.ErrMismatchedHashAndPassword { // password doesn't match
		return u.Message(false, "Invalid login credentials")
	}

	// login successful
	acc.Password = ""

	// create jwt token
	tk := &Token{UserId: acc.ID}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, tk)
	tokenString, err := token.SignedString([]byte(os.Getenv("token_password")))
	if err != nil {
		log.Fatalln(err)
	}
	acc.Token = tokenString
	res := u.Message(true, "Login successful")
	res["account"] = acc
	return res
}

func Delete(ctx context.Context) map[string]interface{} {
	acc := GetUser(ctx.Value("user").(uint))
	if acc == nil {
		return u.Message(false, "Account not found")
	}

	err := GetDB().Delete(acc).Error
	if err != nil {
		return u.Message(false, err.Error())
	}

	return u.Message(true, "Account deleted")
}

func GetUser(u uint) *Account {
	acc := &Account{}
	GetDB().Where("id = ?", u).First(acc)
	if acc.Email == "" {
		return nil
	}

	acc.Password = ""
	return acc
}
