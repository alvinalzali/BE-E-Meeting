package database

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	_ "github.com/lib/pq" // Driver postgres
)

// ConnectDB
func ConnectDB(username, password, dbname, host string, port int) *sql.DB {
	connSt := "host=" + host + " port=" + strconv.Itoa(port) + " user=" + username + " password=" + password + " dbname=" + dbname + " sslmode=disable"
	db, err := sql.Open("postgres", connSt)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to DB " + dbname + " successfully on port " + strconv.Itoa(port))
	return db
}
