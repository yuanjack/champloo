package main

import (
	"encoding/base64"
	"github.com/go-martini/martini"
	"net/http"
	"strings"
)

type AuthUser string

var BasicRealm = "Authorization Required"

func BasicFunc(authfn func(string, string) bool) martini.Handler {
	return func(res http.ResponseWriter, req *http.Request, c martini.Context) {
		if strings.HasPrefix(req.RequestURI, "/api") {
			return
		}

		auth := req.Header.Get("Authorization")
		if len(auth) < 6 || auth[:6] != "Basic " {
			unauthorized(res)
			return
		}
		b, err := base64.StdEncoding.DecodeString(auth[6:])
		if err != nil {
			unauthorized(res)
			return
		}
		tokens := strings.SplitN(string(b), ":", 2)
		if len(tokens) != 2 || !authfn(tokens[0], tokens[1]) {
			unauthorized(res)
			return
		}
		c.Map(AuthUser(tokens[0]))
	}
}

func unauthorized(res http.ResponseWriter) {
	res.Header().Set("WWW-Authenticate", "Basic realm=\""+BasicRealm+"\"")
	http.Error(res, "Not Authorized", http.StatusUnauthorized)
}
