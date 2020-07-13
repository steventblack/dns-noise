//
// Copyright 2020 Steven T Black
//

package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
)

//
// Create the db, schema and load in the data. Return the db
//
func dbLoadDomains(dbPath string, noiseData *os.File) *sql.DB {
	domainsDb := dbOpen(dbPath)
	dbCreateSchema(domainsDb)
	dbLoadData(dbPath, noiseData)

	return domainsDb
}

//
// Open the database at the named path, creating it if it doesn't exist
//
func dbOpen(dbPath string) *sql.DB {
	domainsDb, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err.Error())
	}

	return domainsDb
}

//
// Create the schema, dropping the previous schema & data if exists
//
func dbCreateSchema(domainsDb *sql.DB) {
	// verify the connection to the db is still open
	err := domainsDb.Ping()
	if err != nil {
		log.Fatal(err.Error())
	}

	// Drop existing table (and its data) if it exists; we want a clean slate here
	_, err = domainsDb.Exec(`DROP TABLE IF EXISTS Domains`)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Simple table with two columns: DomainID & Domain
	_, err = domainsDb.Exec(`CREATE TABLE Domains ("DomainId" INT NOT NULL PRIMARY KEY, "Domain" TEXT);`)
	if err != nil {
		log.Fatal(err.Error())
	}
}

//
// Sqlite3 has a command-line capability for fast bulk inserts of csv data
// It's faster to use this pathway than to build a large transaction and commit
//
func dbLoadData(dbPath string, noiseData *os.File) {
	// see "https://linux.die.net/man/1/sqlite3" for details on the .import command
	importCmds := []string{".import", noiseData.Name(), "Domains"}
	err := exec.Command("sqlite3", dbPath, "-csv", "-cmd", strings.Join(importCmds, " ")).Run()
	if err != nil {
		log.Fatal(err.Error())
	}
}

//
// Return the number of rows for Domains table
//
func dbNumDomains(domainsDb *sql.DB) int {
	// verify the connection to the db is still open
	err := domainsDb.Ping()
	if err != nil {
		log.Fatal(err.Error())
	}

	var numRows int
	row := domainsDb.QueryRow(`SELECT COUNT(*) FROM Domains`)
	err = row.Scan(&numRows)
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Printf("%d domains available for noise generation", numRows)
	return numRows
}

//
// Get a random domain; return domain if successful
//
func dbGetRandomDomain(domainsDb *sql.DB) string {
	domainId := rand.Intn(numDomains) + 1

	// verify the connection to the db is still open
	err := domainsDb.Ping()
	if err != nil {
		log.Fatal(err.Error())
	}

	var domain string
	row := domainsDb.QueryRow("SELECT Domain FROM Domains WHERE DomainId=$1", domainId)
	err = row.Scan(&domain)
	if err != nil {
		log.Printf("Unable to get random domain: '%v'", err.Error())
	}

	return domain
}
