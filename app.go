package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	log.Print("Starting application")
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT is not set")
	}

	dns := os.Getenv("DATABASE_URL")

	if dns == "" {
		log.Fatal("$DATABASE_URL is not set")
	}

	db, err := sql.Open("mysql", dns)

	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/api/exercise/new-user", func(w http.ResponseWriter, r *http.Request) {
		newUserHandler(w, r, db)
	})

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

func newUserHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	log.Print("Serving " + r.URL.Path)
	p := getPayload(r)
	username := p["username"]

	var count int
	rows, err := db.Query("SELECT COUNT(id) FROM users WHERE username = ?", username)
	if err != nil {
		log.Fatal(err)
	}
	rows.Next()
	rows.Scan(&count)

	if count != 0 {
		http.Error(w, `{"message": "username already taken"}`, http.StatusConflict)
		return
	}

	stmt, err := db.Prepare("INSERT INTO users(username) VALUES(?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	res, err := stmt.Exec(username)
	if err != nil {
		log.Fatal(err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(w, `{"username": "%s", "_id": "%d"}`, username, id)
}

func getPayload(r *http.Request) map[string]string {
	body := make(map[string]string)
	b, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(b, &body)

	return body
}
