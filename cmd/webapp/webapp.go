package main

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

//go:embed views/*
var views embed.FS

//go:embed static/*
var static embed.FS

func main() {
	r := mux.NewRouter()
	r.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reader, err := views.Open("views/index.html")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		_, err = io.Copy(w, reader)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	}))
	r.Path("/favicon.ico").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bs, err := static.ReadFile("static/favicon.ico")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.Header().Set("Content-Type", "image/x-icon")
		w.Write(bs)
	}))
	r.PathPrefix("/static/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bs, err := static.ReadFile(strings.TrimPrefix(r.URL.Path, "/"))
		if err == os.ErrNotExist {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Content-Type", "text/css")
		} else {
			w.Header().Set("Content-Type", http.DetectContentType(bs))
		}

		_, err = io.Copy(w, bytes.NewReader(bs))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	}))

	addr := "localhost:8081"
	server := &http.Server{
		Handler: handlers.LoggingHandler(os.Stdout, r),
		Addr:    addr,
	}

	fmt.Println("Listening on", addr)
	server.ListenAndServe()
}
