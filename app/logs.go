package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
)

func logHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("").Delims("[[", "]]").
		ParseFiles("template/basic.html", "template/logs.html")
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "basic", nil)
}

func listLogHandler(w http.ResponseWriter, r *http.Request) {
	logFiles, err := mcwDriver.ListLogs()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	bs, err := json.Marshal(logFiles)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, string(bs))
}

func getLogHandler(w http.ResponseWriter, r *http.Request) {
	keys, ok := r.URL.Query()["key"]
	if !ok || len(keys) == 0 {
		log.Println("Key len error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	key := keys[0]
	logStr, err := mcwDriver.GetLog(key)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, logStr)
}

func deleteLogHandler(w http.ResponseWriter, r *http.Request) {
	keys, ok := r.URL.Query()["key"]
	if !ok || len(keys) == 0 {
		log.Println("Key len error")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	key := keys[0]
	err := mcwDriver.DeleteLog(key)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "1")
}
