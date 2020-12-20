package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
)

var templateStore *template.Template

func RenderPage(w http.ResponseWriter, r *http.Request, name string, data interface{}) {
	var err error
	r.ParseForm()

	if templateStore == nil || config.Debug {
		tmpl := template.New("")

		tmpl = tmpl.Funcs(template.FuncMap{"noescape": func(str string) template.HTML {
			return template.HTML(str)
		}})

		tmpl = tmpl.Funcs(template.FuncMap{"trd": func(str string) template.HTML {
			s := r.FormValue(str)
			return template.HTML(s)
		}})

		tmpl, err = tmpl.ParseGlob("templates/*.gohtml")
		if err != nil {
			fmt.Fprint(w, err)
			return
		}

		tmpl, err = tmpl.ParseGlob("pages/*.gohtml")
		if err != nil {
			fmt.Fprint(w, err)
			return
		}

		fmt.Println(tmpl.DefinedTemplates())

		templateStore = tmpl
	}

	err = templateStore.ExecuteTemplate(w, name, data)
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
}

func RenderError(w http.ResponseWriter, r *http.Request, message string) {
	data := struct {
		Title string
		Error string
	}{}

	data.Error = message
	data.Title = "Erro"
	w.WriteHeader(400)
	RenderPage(w, r, "error.gohtml", data)
}

func PipeMJPEG(w http.ResponseWriter, r *http.Request, url string) {
	client, err := http.Get(url)

	if err != nil {
		fmt.Fprintf(w, "Erro")
		return
	}

	defer client.Body.Close()

	w.Header().Set("Content-Type", client.Header.Get("Content-Type"))
	written, err := io.Copy(w, client.Body)

	if config.Debug {
		log.Printf("Requisição encerrada: %d %s", written, err)
	}
}
