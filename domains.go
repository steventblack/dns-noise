//
// Copyright 2020 Steven T Black
//

package main

import (
	"archive/zip"
	"database/sql"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// General functions for fetching the list of DNS domains to be used as noise values.

//
// Fetch the domains, unzipping if needed
// The domains file must be either a csv or a zip-encoded csv
// Returns back a file pointer to the csv
func fetchDomains(sourceURL string) *os.File {
	domainsFile := fetchFile(sourceURL)

	// Check the extension; if .zip then unzip it
	extension := strings.ToLower(filepath.Ext(domainsFile.Name()))
	if extension == ".zip" {
		domainsFile = unzipFile(domainsFile)
	}

	// Recheck the extension (if may have changed if unzipped)
	extension = strings.ToLower(filepath.Ext(domainsFile.Name()))
	if extension != ".csv" {
		log.Fatal("Unexpected file format: '%v'", extension)
	}

	return domainsFile
}

//
// Fetch file from remote source and save it in the tmp dir
//
func fetchFile(sourceURL string) *os.File {
	response, err := http.Get(sourceURL)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Fatal("Unable to fetch domains source: %v", response.StatusCode)
	}

	// create a file in the tmp directory
	domainsFile, err := os.Create(filepath.Join(os.TempDir(), filepath.Base(sourceURL)))
	if err != nil {
		log.Fatal(err.Error())
	}
	defer domainsFile.Close()

	// write the full response body into the newly created file
	_, err = io.Copy(domainsFile, response.Body)
	if err != nil {
		log.Fatal(err.Error())
	}

	return domainsFile
}

//
// Unzip the file and save it in the tmp dir
//
func unzipFile(zipFile *os.File) *os.File {
	zipReader, err := zip.OpenReader(zipFile.Name())
	if err != nil {
		log.Fatal(err.Error())
	}

	// There should only be a single zipped file for the domains
	// Anything more is a problem
	if len(zipReader.File) > 1 {
		log.Fatal("Unexpected number of zipped files: %v", len(zipReader.File))
	}

	// Open the first (only!) zipped file for reading
	zippedFile, err := zipReader.File[0].Open()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer zippedFile.Close()

	// Extract out only the basename for the zipped file and use it
	// to create a destination file of the same name in the tmp directory
	unzippedFilename := filepath.Base(zipReader.File[0].FileHeader.Name)
	unzippedFile, err := os.Create(filepath.Join(os.TempDir(), unzippedFilename))
	if err != nil {
		log.Fatal(err.Error())
	}
	defer unzippedFile.Close()

	// Decodes the zipped file into the destination file
	_, err = io.Copy(unzippedFile, zippedFile)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = os.Remove(zipFile.Name())
	if err != nil {
		log.Printf(err.Error())
	}

	return unzippedFile
}

//
// Check the source to see if it has exceeded its refresh period
func checkSourceRefresh(s Source) bool {
	refresh := false

	if s.Refresh != 0 && time.Since(s.Timestamp) > time.Duration(s.Refresh) {
		log.Printf("Refreshing domains source '%s'", s.Label)
		refresh = true
	}

	return refresh
}

// refreshSources checks to see if any domain sources need to be refreshed and reloads them if so.
// It will fetch a new datafile from the source and reload the database for each dataset that needs refreshing.
func refreshSources(db *sql.DB, sources []Source) {
	for i, s := range sources {
		// if timestamp has not been initialized, then set it
		// fantastic subtlety in syntax here: while slices are passed in as a value, the contents of the slice are
		// effectively passed in by ref. this means you can modify an *existing* slice entry but adding/removing an
		// entry will not persist outside of scope. modifying the timestamp for an *existing* slice entry should
		// persist. however, when the slice entry is returned from the range function, it is a *value* copy of the
		// slice entry and NOT the original! this means any modification will NOT persist outside of scope if made
		// against the copy returned by range. however, if you instead use the index value to access directly into
		// the slice you can successfully modify the contents and have it persist. perfectly logical if not perfectly obvious.
		if s.Timestamp.IsZero() {
			sources[i].Timestamp = time.Now()
			log.Printf("Initialized source '%s' refresh to %v", s.Label, s.Timestamp)
		}

		if checkSourceRefresh(s) {
			sourceFile := fetchDomains(s.Url)
			dbLoadCSV(db, sourceFile.Name(), s.Label, s.Column)

			sources[i].Timestamp = time.Now()
		}
	}
}
