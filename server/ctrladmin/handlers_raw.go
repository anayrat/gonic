package ctrladmin

import (
	"net/http"

	"github.com/gorilla/sessions"

	"senan.xyz/g/gonic/server/key"
)

func (c *Controller) ServeLoginDo(w http.ResponseWriter, r *http.Request) {
	session := r.Context().Value(key.Session).(*sessions.Session)
	username := r.FormValue("username")
	password := r.FormValue("password")
	if username == "" || password == "" {
		sessAddFlashW(session, "please provide both a username and password")
		sessLogSave(session, w, r)
		http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
		return
	}
	user := c.DB.GetUserFromName(username)
	if user == nil || password != user.Password {
		sessAddFlashW(session, "invalid username / password")
		sessLogSave(session, w, r)
		http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
		return
	}
	// put the user name into the session. future endpoints after this one
	// are wrapped with WithUserSession() which will get the name from the
	// session and put the row into the request context
	session.Values["user"] = user.Name
	sessLogSave(session, w, r)
	http.Redirect(w, r, "/admin/home", http.StatusSeeOther)
}

func (c *Controller) ServeLogout(w http.ResponseWriter, r *http.Request) {
	session := r.Context().Value(key.Session).(*sessions.Session)
	session.Options.MaxAge = -1
	sessLogSave(session, w, r)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}
