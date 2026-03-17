package Handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/CeaDev/expense-tracker/internals/models"
)

// GET: gets a list of all users in DB
func GetUsers(w http.ResponseWriter, db *sql.DB) {
	rows, err := db.Query("SELECT * from users")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	users := make([]models.User, 0)
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.Id, &user.Name, &user.Email, &user.Password, &user.CreatedAt); err != nil {
			http.Error(w, "Scan error", http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}
	w.Header().Set("Content-Type", "application/json")
	// Encode() converts users to JSON encoding. NewEncoder(w) then writes that JSON value to output stream
	encoder := json.NewEncoder(w)
	err = encoder.Encode(users)
	if err != nil {
		http.Error(w, "Error Encoding JSON", http.StatusInternalServerError)
	}
}

// GET: gets the user with the specified ID
func GetUserById(w http.ResponseWriter, id string, db *sql.DB) {
	// Content type will be plain text in case of error. If we successfully execute the function, this will be
	// set to application/json at the end of the function
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	// Verify numbwer sent is an int
	_, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "The submitted ID is not an integer!", http.StatusInternalServerError)
		return
	}

	queryString := "SELECT * FROM users WHERE user_id = " + id
	rows, err := db.Query(queryString)
	if err != nil {
		http.Error(w, "Could Not Query Database!", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var user models.User
	if rows.Next() {
		err = rows.Scan(&user.Id, &user.Name, &user.Email, &user.Password, &user.CreatedAt)
		if err != nil {
			http.Error(w, "Error scanning rows from database", http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "User does not exist!", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err = encoder.Encode(user)
	if err != nil {
		http.Error(w, "Error Encoding JSON", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
}

func PostUser(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var user models.User
	// Get JSON data from request
	jsonData, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "JSON data could not be read", http.StatusInternalServerError)
	}
	// Unmarshal the json data into a user struct
	err = json.Unmarshal(jsonData, &user)
	if err != nil {
		http.Error(w, "JSON format is not correct", http.StatusInternalServerError)
	}
	// Add user to DB
	user.CreatedAt = time.Now().Format("Jan-02-2006 03:04:05 PM")
	query := fmt.Sprintf("INSERT INTO users (name, email, createdAt, password) VALUES ('%s', '%s', '%s', '%s');", user.Name, user.Email, user.CreatedAt, user.Password)
	_, err = db.Exec(query)
	if err != nil {
		http.Error(w, "User could not be added to DB", http.StatusInternalServerError)
	}
}
