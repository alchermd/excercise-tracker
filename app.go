package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
)

func main() {
	log.Print("Starting application")
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT is not set")
	}

	http.HandleFunc("/", indexHandler)

	log.Print("Serving static assets on /assets")
	fs := http.FileServer(http.Dir("assets/"))
	http.Handle("/assets/", http.StripPrefix("/assets/", fs))

	log.Print("Listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("Serving " + r.URL.Path)

	t := template.Must(template.ParseFiles("views/index.html"))
	t.Execute(w, nil)
}
