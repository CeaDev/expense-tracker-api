package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/CeaDev/expense-tracker/configs"
	"github.com/CeaDev/expense-tracker/internals/Handlers"
)

var Db *sql.DB

func main() {
	// Open Database Connection
	Db, db_err := configs.ConnectDB()
	if db_err != nil {
		fmt.Println("Error Connecting to Database!")
	}

	// Gets the list of users
	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			Handlers.GetUsers(w, Db)
		case "POST":
			r.Header.Set("Content-Type", "application/json")
			Handlers.PostUser(w, r, Db)
		default:
			http.Error(w, "Endpoint could not be accessed!", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		result := strings.Split(r.URL.Path, "/")
		id := result[len(result)-1]
		Handlers.GetUserById(w, id, Db)
	})

	// Start Server
	fmt.Printf("Starting Server on port 8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Printf("Error Connecting to Server!")
	}

	// TODO: Handle Closing the db connection on shutdown!
	// Closing Database Connection
	defer configs.CloseDB(Db)

}
