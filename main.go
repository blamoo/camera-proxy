package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

var configPath string

func main() {
	var err error

	flag.StringVar(&configPath, "c", "./config/config.json", "Caminho para o arquivo de configuração. Padrão: ./config/config.json")

	err = InitializeConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	InitializeSession()

	r := mux.NewRouter()

	r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		data := struct {
			Title string
			Error string
		}{
			Title: "Entrar",
		}

		fUser := r.FormValue("user")
		fPassword := r.FormValue("password")

		if r.Method == http.MethodPost {
			passwordHash, foundUser := config.Users[fUser]

			if !foundUser {
				data.Error = "Usuário não encontrado"
				RenderPage(w, r, "login.gohtml", data)
				return
			}

			if !CheckPasswordHash(fPassword, passwordHash) {
				data.Error = "Senha incorreta"
				RenderPage(w, r, "login.gohtml", data)
				return
			}

			session, _ := sessionStore.Get(r, config.SessionCookieName)
			session.Values["authenticated"] = true
			session.Values["user"] = fUser
			session.Save(r, w)

			returnPage := r.FormValue("return")
			if len(returnPage) > 0 && returnPage[0] == '/' {
				http.Redirect(w, r, returnPage, 302)
				return
			}

			http.Redirect(w, r, "/", 302)
			return
		}

		RenderPage(w, r, "login.gohtml", data)
	})

	r.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			session, _ := sessionStore.Get(r, config.SessionCookieName)
			session.Values["authenticated"] = false
			session.Values["user"] = nil
			session.Save(r, w)
		}

		http.Redirect(w, r, "/login", 302)
	})

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, ret := HandleAuth(w, r)
		if ret {
			return
		}

		data := struct {
			Title   string
			Cameras []Camera
		}{
			Title:   "Câmeras",
			Cameras: config.Cameras,
		}

		RenderPage(w, r, "index.gohtml", data)
	})

	r.HandleFunc("/camera/{name}", func(w http.ResponseWriter, r *http.Request) {
		_, ret := HandleAuth(w, r)
		if ret {
			return
		}

		vars := mux.Vars(r)
		vName, _ := vars["name"]

		for _, camera := range config.Cameras {
			if vName == camera.Name {
				PipeMJPEG(w, r, camera.URL)
				return
			}
		}

		RenderError(w, r, "Câmera não encontrada")
	})

	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		http.ServeFile(w, r, "static/favicon.ico")
	})

	err = http.ListenAndServeTLS(fmt.Sprintf("%s:%d", config.ServerHost, config.ServerPort), config.TLSCertFile, config.TLSKeyFile, r)

	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
}
