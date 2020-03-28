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
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

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
	http.HandleFunc("/api/exercise/users", func(w http.ResponseWriter, r *http.Request) {
		allUsersHandler(w, r, db)
	})
	http.HandleFunc("/api/exercise/add", func(w http.ResponseWriter, r *http.Request) {
		newExerciseHandler(w, r, db)
	})

	log.Print("Serving static assets on /assets")
	fs := http.FileServer(http.Dir("assets/"))
	http.Handle("/assets/", http.StripPrefix("/assets/", fs))

	log.Print("Listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

type User struct {
	Id       int64  `json:"_id"`
	Username string `json:"username"`
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("Serving " + r.URL.Path)

	t := template.Must(template.ParseFiles("views/index.html"))
	t.Execute(w, nil)
}

func newUserHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	log.Print("Serving " + r.URL.Path)

	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Content-Type", "application/json")

	username := getPayloadData(r, "username")

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

	user := &User{
		Username: username,
		Id:       id,
	}

	j, err := json.Marshal(user)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(w, string(j))
}

func allUsersHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	log.Print("Serving " + r.URL.Path)

	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Content-Type", "application/json")

	rows, err := db.Query("SELECT id, username FROM users ORDER BY id")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.Id, &user.Username)
		if err != nil {
			log.Fatal(err)
		}

		users = append(users, user)
	}

	j, err := json.Marshal(users)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(w, string(j))
}

func newExerciseHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	log.Print("Serving " + r.URL.Path)

	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Content-Type", "application/json")

	id := getPayloadData(r, "userId")
	description := getPayloadData(r, "description")
	duration := getPayloadData(r, "duration")
	date := getPayloadData(r, "date")

	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	stmt, err := db.Prepare("INSERT INTO exercises(user_id, description, duration, date) VALUES(?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(id, description, duration, date)
	if err != nil {
		log.Fatal(err)
	}

	rows, err := db.Query("SELECT username FROM  users WHERE id = ?", id)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	var username string
	rows.Next()
	rows.Scan(&username)

	fmt.Fprintf(w, `{"username": "%s", "description": "%s", "duration": %s, "_id": %s, date: "%s"}`, username, description, duration, id, date)
}

func getPayloadData(r *http.Request, key string) (value string) {
	ct := r.Header.Get("Content-Type")
	if ct == "application/json" {
		body := make(map[string]string)
		b, _ := ioutil.ReadAll(r.Body)
		json.Unmarshal(b, &body)

		value = body[key]
	}

	if ct == "application/x-www-form-urlencoded" {
		r.ParseForm()
		value = r.FormValue(key)
	}

	return
}
