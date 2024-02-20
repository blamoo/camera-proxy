package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/sync/errgroup"
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

			// r.AddCookie(&http.Cookie{
			// 	Name:   config.SessionCookieName,
			// 	Value:  "",
			// 	MaxAge: -1,
			// })
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

	r.HandleFunc("/camera/files/{name}/{path:.*}", func(w http.ResponseWriter, r *http.Request) {
		_, ret := HandleAuth(w, r)
		if ret {
			return
		}

		vars := mux.Vars(r)
		vName, _ := vars["name"]
		vPath, _ := vars["path"]

		camera, err := config.FindCamera(vName)

		if err != nil {
			RenderError(w, r, err.Error())
			return
		}

		path := filepath.Join(camera.Files, vPath)
		path = filepath.Clean(path)

		if err != nil {
			RenderError(w, r, err.Error())
			return
		}

		file, err := os.Stat(path)

		if err != nil {
			RenderError(w, r, err.Error())
			return
		}

		if file.IsDir() {
			type FileWrap struct {
				IsDir      bool
				Name       string
				Type       string
				Embeddable bool
				Mtime      time.Time
			}

			data := struct {
				Title  string
				Camera Camera
				VPath  string
				File   fs.FileInfo
				Files  []FileWrap
			}{
				Title:  fmt.Sprintf("Arquivos de %s", camera.Name),
				VPath:  vPath,
				File:   file,
				Camera: camera,
			}

			o, err := os.ReadDir(path)

			if err != nil {
				RenderError(w, r, err.Error())
				return
			}

			data.Files = make([]FileWrap, len(o))
			for k, v := range o {
				var tmp FileWrap

				tmp.Name = v.Name()
				tmp.IsDir = v.IsDir()

				info, _ := v.Info()
				tmp.Mtime = info.ModTime()

				if !v.IsDir() {

					switch filepath.Ext(tmp.Name) {
					case ".jpg", ".jpeg", ".png":
						tmp.Type = "Image"
						tmp.Embeddable = true

					case ".mp4":
						tmp.Type = "Video"
						tmp.Embeddable = true
					}
				}
				data.Files[k] = tmp
			}

			sort.SliceStable(data.Files, func(i, j int) bool {
				return data.Files[i].Mtime.UnixMicro() > data.Files[j].Mtime.UnixMicro()
			})

			RenderPage(w, r, "files.gohtml", data)
			return
		}

		http.ServeFile(w, r, path)
	})

	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		http.ServeFile(w, r, "static/favicon.ico")
	})

	g, _ := errgroup.WithContext(context.Background())

	if config.ServerHost != "" {
		g.Go(func() error {
			addr := fmt.Sprintf("%s:%d", config.ServerHost, config.ServerPort)
			fmt.Printf("Listening: https://%s/\n", addr)
			return http.ListenAndServeTLS(addr, config.TLSCertFile, config.TLSKeyFile, r)
		})
	}

	if config.LocalHost != "" {
		g.Go(func() error {
			addr := fmt.Sprintf("%s:%d", config.LocalHost, config.LocalPort)
			fmt.Printf("Listening: http://%s/\n", addr)
			return http.ListenAndServe(addr, r)
		})
	}

	err = g.Wait()

	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
}
