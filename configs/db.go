package configs

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func ConnectDB() (*sql.DB, error) {
	fmt.Println("OPENING DATABASE")
	connection := "postgres://postgres:passwordhere@localhost:5432/expenses_db?sslmode=disable"
	db, err := sql.Open("postgres", connection)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return db, nil
}

func CloseDB(db *sql.DB) {
	db.Close()
}
