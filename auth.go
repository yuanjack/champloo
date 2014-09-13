package main

import (
	"net/http"
	"strings"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
	webSessions "github.com/martini-contrib/sessions"
)

func AuthFunc(req *http.Request, session webSessions.Session, r render.Render, c martini.Context) bool {
	if strings.HasPrefix(req.RequestURI, "/api") || strings.HasPrefix(req.RequestURI, "/login") {
		return true
	}

	user := session.Get("auth_user")
	if user == nil {
		r.Redirect("/login", 302)
		return true
	}

	session.Set("auth_user", user)
	c.Map(user.(string))
	return true

}
func Login(params martini.Params, r render.Render) {
	data := map[string]interface{}{"username": "", "msg": ""}
	r.HTML(200, "login", data)
}
func Signin(req *http.Request, session webSessions.Session, r render.Render) {
	req.ParseForm()
	username := req.PostForm.Get("name")
	password := req.PostForm.Get("password")

	var user User
	db.First(&user, User{Name: username})
	if user.Id <= 0 {
		data := map[string]interface{}{"username": "", "msg": "User not found!"}
		r.HTML(200, "login", data)
		return
	}

	if password == user.Password {
		session.Set("auth_user", username)
		r.Redirect("/", 302)
	} else {
		data := map[string]interface{}{"username": "", "msg": "Incorrent password."}
		r.HTML(200, "login", data)
	}
}
func Signout(session webSessions.Session, r render.Render) {
	session.Delete("auth_user")

	r.Redirect("/login", 302)
}
