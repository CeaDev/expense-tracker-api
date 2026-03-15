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

	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		Handlers.GetUsers(w, Db)
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
