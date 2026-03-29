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

// Struct for marshaling JSON token
type TokenResponse struct {
	TokenString string `json:"tokenString"`
	TokenType   string `json:"tokenType"`
}

// This struct is for when we want to send a successful message along with the 200 status code
// I want to include this message as a response from endpoints that don't return any json data.
// Reason: It seems a little vague to send the 200 status code with no explanation/message
type SuccessMessage struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Function for verifying the token sent in a request header
// It will return the Token claims
func VerifyJWTToken(tokenstring string) jwt.MapClaims {
	err := godotenv.Load("../../variables.env")
	if err != nil {
		return nil
	}
	env_variables, err := godotenv.Read("../../variables.env")
	if err != nil {
		return nil
	}
	secret_key := env_variables["jwt_key"]
	token, err := jwt.Parse(tokenstring, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret_key), nil
	})
	if err != nil {
		return nil
	}
	return token.Claims.(jwt.MapClaims)
}

func PostLogin(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")
	var jsonCredentials Credentials
	// Read in request data
	jsonData, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Request Body could not be read", http.StatusInternalServerError)
		return
	}
	err = json.Unmarshal(jsonData, &jsonCredentials)
	// Validate the name-password by checking database
	query := "SELECT user_id, role FROM users WHERE (email = '" + jsonCredentials.Email + "' AND password = '" + jsonCredentials.Password + "');"
	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "Error while trying to log in", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	if rows.Next() {
		// If row exists, get the ID/role of the user so we can add it to the claims of the JSON token
		var id int64
		var role string
		err = rows.Scan(&id, &role)
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
		// Header
		token := jwt.New(jwt.SigningMethodHS256)
		// Payload
		claims := token.Claims.(jwt.MapClaims)
		current_time := time.Now()
		claims["iss"] = current_time.Unix()
		claims["exp"] = current_time.Add(6 * time.Minute).Unix()
		claims["authorized"] = true
		claims["id"] = id
		claims["role"] = role
		secret_key := envVariables["jwt_key"]
		// Signature
		signed_String, err := token.SignedString([]byte(secret_key))
		if err != nil {
			http.Error(w, "Error while setting Token signed string", http.StatusInternalServerError)
			return
		}
		response := TokenResponse{
			TokenString: signed_String,
			TokenType:   "Bearer",
		}
		err = json.NewEncoder(w).Encode(response)
		if err != nil {
			http.Error(w, "Error while processing token", http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "Email/Password combination is not correct", http.StatusInternalServerError)

	}
}

// GET: gets a list of all users in DB
// NOTE: Only and ADMIN can access the data for ALL/Other users
func GetUsers(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Get Authorization value from header
	tokenString := r.Header.Get("Authorization")
	// Get Claims from JWT token, we need this to verify user role later
	validatedToken := VerifyJWTToken(tokenString)
	if validatedToken != nil {
		// If token is valid, check that the user is an ADMIN
		if validatedToken["role"] != "admin" {
			http.Error(w, "UNAUTHORIZED! Token was validated, but user was not an Admin!", http.StatusUnauthorized)
			return
		}
	} else {
		// If the token was nil (not validated)
		http.Error(w, "UNAUTHORIZED! Token could not be validated!", http.StatusUnauthorized)
		return
	}

	// If the above passes (token valid, user is admin) get all user data
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

// GET: gets user information of the user with the specified id
// Restriction: A user role can only see their own information, not the information of others
// An admin can see information for ALL users
func GetUserById(w http.ResponseWriter, r *http.Request, id string, db *sql.DB) {
	// Verify number sent is an int
	convertedID, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "The submitted ID is not an integer!", http.StatusInternalServerError)
		return
	}
	// Get the token in request header
	tokenString := r.Header.Get("Authorization")
	// Validate token string
	validatedTokenClaims := VerifyJWTToken(tokenString)
	if validatedTokenClaims == nil {
		http.Error(w, "Token could not be validated!", http.StatusUnauthorized)
		return
	}
	// Verify that a user role can only view their own data. Admins can verify ALL user data
	tokenClaimID := int64(validatedTokenClaims["id"].(float64))
	tokenClaimRole := validatedTokenClaims["role"]
	if tokenClaimID != int64(convertedID) && tokenClaimRole != "admin" {
		http.Error(w, "Unauthorized user is making a request", http.StatusUnauthorized)
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
		http.Error(w, "User with that ID does not exist!", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err = encoder.Encode(user)
	if err != nil {
		http.Error(w, "Error Encoding JSON", http.StatusInternalServerError)
		return
	}
}

// Endpoint for creating user
// Request only needs to pass name, email, and password. createdAt and role fields are automatically set
func PostUser(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")
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
		return
	} else {
		responseMessage := SuccessMessage{
			Status:  "Success",
			Message: "User account was successfully created!",
		}
		err = json.NewEncoder(w).Encode(responseMessage)
		if err != nil {
			http.Error(w, "Error while creating response body", http.StatusInternalServerError)
			return
		}
	}
}

// Endpoint for deleting user
// Only an Admin can delete user accounts
func DeleteUser(w http.ResponseWriter, r *http.Request, id string, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")
	// Verify that the ID is indeed a number
	_, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "ID is not a number", http.StatusInternalServerError)
		return
	}
	// Get token string from request header
	tokenstring := r.Header.Get("Authorization")
	// Verify the token string
	verifiedTokenClaim := VerifyJWTToken(tokenstring)
	if verifiedTokenClaim == nil {
		http.Error(w, "Token could not be verified", http.StatusUnauthorized)
		return
	}
	// Check if user is  an admin
	if verifiedTokenClaim["role"] != "admin" {
		http.Error(w, "Unauthorized user is trying to modify user accounts", http.StatusUnauthorized)
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
	// Inform the user that the account was deleted with an HTTP response
	successMesssage := SuccessMessage{
		Status:  "Success",
		Message: "User account was successfully deleted!",
	}
	err = json.NewEncoder(w).Encode(successMesssage)
	if err != nil {
		http.Error(w, "Error while trying to delete user account!", http.StatusInternalServerError)
		return
	}
}

// Endpoint for updating user data
// Users can only modify their own user info.
func UpdateUser(w http.ResponseWriter, r *http.Request, db *sql.DB, id string) {
	w.Header().Set("Content-Type", "application/json")
	// Verify the requested ID is a number
	convertedId, err := strconv.Atoi(id)
	if err != nil {
		http.Error(w, "The ID is not an Integer!", http.StatusInternalServerError)
		return
	}
	// Get token string from request header
	tokenString := r.Header.Get("Authorization")
	// Validate token string
	verifiedTokenClaims := VerifyJWTToken(tokenString)
	if verifiedTokenClaims == nil {
		http.Error(w, "Token could not be authorized!", http.StatusUnauthorized)
		return
	}
	// Verify that the user requesting the data is modifying their own, not other user data
	tokenClaimsId := int(verifiedTokenClaims["id"].(float64))
	if convertedId != tokenClaimsId {
		http.Error(w, "Unauthorized user is making changes to user data!", http.StatusUnauthorized)
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
			http.Error(w, "Error unmarshalling JSON data into struct. JSON request body format may be incorrect!", http.StatusInternalServerError)
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
		_, err = db.Exec(update_query)
		if err != nil {
			http.Error(w, "Invalid data provided", http.StatusInternalServerError)
			return
		}
		// User data was succesfully modified, inform user
		successMessage := SuccessMessage{
			Status:  "Success",
			Message: "User data was successfully modified!",
		}
		err = json.NewEncoder(w).Encode(successMessage)
		if err != nil {
			http.Error(w, "User data could not be modified!", http.StatusInternalServerError)
		}

	} else {
		http.Error(w, "The User with that ID does not exist!!", http.StatusInternalServerError)
		return
	}
}
