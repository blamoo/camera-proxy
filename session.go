package main

import (
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

var sessionStore *sessions.FilesystemStore

func InitializeSession() {
	sessionStore = sessions.NewFilesystemStore("", config.SessionCookieKey)
	sessionStore.Options.Secure = false
	sessionStore.Options.SameSite = http.SameSiteStrictMode
	sessionStore.MaxAge(config.SessionCookieMaxAge)
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func CheckAuth(w http.ResponseWriter, r *http.Request) (string, bool) {
	session, _ := sessionStore.Get(r, config.SessionCookieName)
	session.Save(r, w)

	auth, ok1 := session.Values["authenticated"].(bool)
	id, ok2 := session.Values["user"].(string)

	if !ok1 || !ok2 || !auth {
		return "", true
	}

	return id, false
}

func HandleAuth(w http.ResponseWriter, r *http.Request) (string, bool) {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	ipParsed := net.ParseIP(ip)

	for _, cidr := range config.AuthWhitelist {
		_, parsedCidr, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}

		if parsedCidr.Contains(ipParsed) {
			return fmt.Sprintf("%s %s", ip, cidr), false
		}
	}

	id, ret := CheckAuth(w, r)

	if ret {
		returnPath := fmt.Sprintf("%s?%s", r.URL.Path, r.URL.RawQuery)
		u := new(url.URL)
		u.Path = "/login"
		q := u.Query()
		q.Set("return", returnPath)
		u.RawQuery = q.Encode()
		http.Redirect(w, r, u.String(), 302)
	}

	return id, ret
}
