//
// Copyright 2020 Steven T Black
//

package main

import (
	"database/sql"
	"encoding/csv"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"log"
	"math/rand"
	"os"
)

// dbOpen will open the database specified in path or create the database at the path if it doesn't exist.
// If successful, it will return a database connection pointer.
func dbOpen(path string) *sql.DB {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

// dbCreateSchema will create the schema required for service operation.
// It will drop the schema (if it exists) before creating the schema in order to minimize impact of future changes.
func dbCreateSchema(db *sql.DB) {
	// validate connection to database is still valid
	err := db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	// drop existing table (and its data) if it already exists
	// don't want to have any complications if the schema changes over time
	drop := `DROP TABLE IF EXISTS Domains`
	_, err = db.Exec(drop)
	if err != nil {
		log.Fatal(err)
	}

	// create the schema
	schema := `CREATE TABLE Domains ("DomainId" INTEGER PRIMARY KEY AUTOINCREMENT, "Domain" TEXT NOT NULL, "Label" TEXT NOT NULL);`
	_, err = db.Exec(schema)
	if err != nil {
		log.Fatal(err)
	}
}

// dbLoadCSV reads the specified file into the database.
// The data is associated with the given label to provide a means for independently refreshing if multiple sources are loaded.
// If data with the label already exist in the database, it will be dropped prior to loading the new set.
// The column indicates which column in the data file has the list of domains (0-based index).
func dbLoadCSV(db *sql.DB, path, label string, column int) {
	// validate connection to database is still valid
	err := db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	// remove any data previously associated with the label first
	dbPurgeData(db, label)

	csvFile, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer csvFile.Close()

	// if there's an error loading the data, rollback to a clean state
	// if the transaction was committed successfully, the rollback will be a noop
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()

	// be sure the statement is released when done to avoid leaking resources
	statement, err := tx.Prepare("INSERT INTO Domains(Domain, Label) VALUES(?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer statement.Close()

	reader := csv.NewReader(csvFile)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		_, err = statement.Exec(record[column], label)
		if err != nil {
			log.Print(err)
			continue
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
}

// dbPurgeData deletes the data associated with the provided label from the database.
// It is not an error if no rows match the label.
func dbPurgeData(db *sql.DB, label string) {
	// validate connection to database is still valid
	err := db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	statement, err := db.Prepare("DELETE FROM Domains WHERE Label=?")
	if err != nil {
		log.Fatal(err)
		return
	}

	response, err := statement.Exec(label)
	if err != nil {
		log.Fatal(err)
	}

	numRows, err := response.RowsAffected()
	log.Printf("Deleted %d rows for label '%s'", numRows, label)
}

// dbCountRows returns the number of rows found in the Domains table.
// It ignores the source label and simply returns the number available for use.
// It is a fatal error if it is unable to access the database or query the Domains table.
func dbCountRows(db *sql.DB) int {
	// validate connection to database is still valid
	err := db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	statement := `SELECT COUNT(*) FROM Domains`
	var numRows int
	err = db.QueryRow(statement).Scan(&numRows)
	if err != nil {
		log.Fatal(err)
	}

	metricsDnsNoiseDomains(float64(numRows))

	return numRows
}

// dbGetRandomDomain fetches a random domain from the database.
// If it is unable to fetch a domain, it will return an error and the domain will be empty
func dbGetRandomDomain(db *sql.DB) (string, error) {
	// validate connection to database is still valid
	err := db.Ping()
	if err != nil {
		log.Print(err)
		return "", err
	}

	// There may be a large number of rows in the database which don't perform well
	// with the simpler queries using the ORDER BY RANDOM() as that results in table scans.
	// Selecting a random OFFSET within the table performs faster for large tables.
	numRows := dbCountRows(db)
	offset := rand.Intn(numRows)

	var domain string
	err = db.QueryRow("SELECT Domain FROM Domains LIMIT 1 OFFSET $1", offset).Scan(&domain)
	if err != nil {
		log.Print(err)
		return "", err
	}

	return domain, nil
}
