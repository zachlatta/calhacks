package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/zachlatta/calhacks/config"
	"github.com/zachlatta/calhacks/model"
	"github.com/dgrijalva/jwt-go"
)

func renderJSON(w http.ResponseWriter, v interface{}, status int) error {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(&v)
}

func createToken(u *model.User) (string, error) {
	token := jwt.New(jwt.GetSigningMethod("HS256"))
	token.Claims["id"] = u.ID
	token.Claims["exp"] = time.Now().Add(time.Hour * 72).Unix()
	return token.SignedString([]byte(config.JWTSecret()))
}
