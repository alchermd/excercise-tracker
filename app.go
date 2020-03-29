package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
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
	http.HandleFunc("/api/exercise/log", func(w http.ResponseWriter, r *http.Request) {
		getExerciseHandler(w, r, db)
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

type Exercise struct {
	Description string `json:"description"`
	Duration    int    `json:"duration"`
	Date        string `json"date"`
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

	var username string
	ct := r.Header.Get("Content-Type")

	if ct == "application/json" {
		p := getPayload(r)
		username = p["username"].(string)
	}
	if ct == "application/x-www-form-urlencoded" {
		r.ParseForm()
		username = r.FormValue("username")
	}

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

	var (
		id, description, date string
		duration              int
	)

	ct := r.Header.Get("Content-Type")
	if ct == "application/json" {
		p := getPayload(r)

		id = p["userId"].(string)
		description = p["description"].(string)
		duration = int(p["duration"].(float64))
		date = p["date"].(string)
	}
	if ct == "application/x-www-form-urlencoded" {
		r.ParseForm()

		id = r.FormValue("userId")
		description = r.FormValue("description")
		durationFloat, _ := strconv.ParseFloat(r.FormValue("duration"), 64)
		duration = int(durationFloat)
		date = r.FormValue("date")
	}

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

	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		log.Fatal(err)
	}
	date = d.Format("Mon Jan 02 2006")
	fmt.Fprintf(w, `{"username": "%s", "description": "%s", "duration": %d, "_id": %s, "date": "%s"}`, username, description, duration, id, date)
}

func getExerciseHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	log.Print("Serving " + r.URL.Path)

	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Content-Type", "application/json")

	userId := r.URL.Query().Get("userId")

	var username string
	rows, err := db.Query("SELECT username FROM users WHERE id = ?", userId)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	rows.Next()
	rows.Scan(&username)

	rows, err = db.Query("SELECT description, duration, date FROM exercises WHERE user_id = ?", userId)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	exercises := []Exercise{}
	for rows.Next() {
		var e Exercise
		rows.Scan(&e.Description, &e.Duration, &e.Date)

		exercises = append(exercises, e)
	}

	j, err := json.Marshal(exercises)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(w, `{"_id": "%s", "username": "%s", "log": %s, "count": %d}`, userId, username, string(j), len(exercises))
}

func getPayload(r *http.Request) map[string]interface{} {
	body := make(map[string]interface{})
	b, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(b, &body)

	return body
}
