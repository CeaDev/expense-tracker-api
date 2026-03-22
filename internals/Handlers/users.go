package Handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CeaDev/expense-tracker/internals/models"
	"github.com/golang-jwt/jwt"
	"github.com/joho/godotenv"
)

// Struct for unmarshaling login credentials
type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func PostLogin(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var jsonCredentials Credentials
	// Read in request data
	jsonData, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Request Body could not be read", http.StatusInternalServerError)
		return
	}
	err = json.Unmarshal(jsonData, &jsonCredentials)
	// Validate the name-password by checking database
	query := "SELECT user_id FROM users WHERE (email = '" + jsonCredentials.Email + "' AND password = '" + jsonCredentials.Password + "');"
	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "Error while Querying Database for user name", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	if rows.Next() {
		// If row exists, get the ID so we can add it to the claims of the JSON token
		var id int64
		err = rows.Scan(&id)
		if err != nil {
			http.Error(w, "Error while getting ID from found user", http.StatusInternalServerError)
		}
		// Generate JWT Token if user is found
		// Load secret key from .ENV file
		err := godotenv.Load("../../variables.env")
		if err != nil {
			http.Error(w, "Error loading .env file", http.StatusInternalServerError)
			return
		}
		envVariables, err := godotenv.Read("../../variables.env")
		if err != nil {
			http.Error(w, "Error reading variables from .env file!", http.StatusInternalServerError)
			return
		}
		token := jwt.New(jwt.SigningMethodHS256)
		claims := token.Claims.(jwt.MapClaims)
		claims["exp"] = time.Now().Add(10 * time.Minute)
		claims["authorized"] = true
		claims["id"] = id
		secret_key := envVariables["jwt_key"]
		signed_String, err := token.SignedString([]byte(secret_key))
		fmt.Println(signed_String)
		if err != nil {
			fmt.Println(err.Error())
			http.Error(w, "Error while setting Token signed string", http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Email/Password combination is not correct", http.StatusInternalServerError)

	}
}

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
		if err := rows.Scan(&user.Id, &user.Name, &user.Email, &user.CreatedAt, &user.Password, &user.Role); err != nil {
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
		err = rows.Scan(&user.Id, &user.Name, &user.Email, &user.CreatedAt, &user.Password, &user.Role)
		if err != nil {
			http.Error(w, "Error scanning rows from database", http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "User does not exist!", http.StatusInternalServerError)
		return
	}
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
		return
	}
	// Unmarshal the json data into a user struct
	err = json.Unmarshal(jsonData, &user)
	if err != nil {
		http.Error(w, "JSON format is not correct", http.StatusInternalServerError)
		return
	}
	// Add user to DB
	user.CreatedAt = time.Now().Format("Jan-02-2006 03:04:05 PM")
	query := fmt.Sprintf("INSERT INTO users (name, email, createdAt, password, role) VALUES ('%s', '%s', '%s', '%s', 'user');", user.Name, user.Email, user.CreatedAt, user.Password)
	_, err = db.Exec(query)
	if err != nil {
		http.Error(w, "User could not be added to DB", http.StatusInternalServerError)
	}
}

func DeleteUser(w http.ResponseWriter, id string, db *sql.DB) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "ID is not a number", http.StatusInternalServerError)
		return
	}
	query := "DELETE from users WHERE user_id = " + id
	result, err := db.Exec(query)
	numRows, err := result.RowsAffected()
	if err != nil {
		http.Error(w, "There was an ERROR while deleting the user!", http.StatusInternalServerError)
		return
	}
	if numRows == 0 {
		http.Error(w, "There is no user that has that ID!", http.StatusInternalServerError)
		return
	}
}

func UpdateUser(w http.ResponseWriter, r *http.Request, db *sql.DB, id string) {
	_, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "The ID is not an Integer!", http.StatusInternalServerError)
		return
	}
	query := "SELECT * from users where user_id = " + id
	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "Could not Query the Database", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	if rows.Next() {
		// unmarshal json and change the fields
		var user models.User
		jsonData, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error parsing response body!", http.StatusInternalServerError)
			return
		}
		err = json.Unmarshal(jsonData, &user)
		if err != nil {
			http.Error(w, "Error unmarshalling JSON data into struct", http.StatusInternalServerError)
			return
		}
		update_query := "UPDATE users SET "
		if user.Name != "" {
			set_query := "name = '" + user.Name + "',"
			update_query += set_query
		}
		if user.Email != "" {
			set_query := "email = '" + user.Email + "',"
			update_query += set_query
		}
		if user.Password != "" {
			set_query := "password = '" + user.Password + "'"
			update_query += set_query
		}
		// check for trailing comma
		update_query = strings.TrimSuffix(update_query, ",")
		update_query += " WHERE user_id = " + id
		fmt.Println(update_query)
		_, err = db.Exec(update_query)
		if err != nil {
			http.Error(w, "ERROR trying to execute UPDATE Query on Database!", http.StatusInternalServerError)
		}

	} else {
		http.Error(w, "The User with that ID does not exist!!", http.StatusInternalServerError)
		return
	}
}
