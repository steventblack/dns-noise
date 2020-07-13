//
// Copyright 2020 Steven T Black
//

package main

import (
	"archive/zip"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// General functions for fetching the list of DNS domains to be used as noise values.

//
// Fetch the domains, unzipping if needed
// The domains file must be either a csv or a zip-encoded csv
// Returns back a file pointer to the csv
//
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

	return unzippedFile
}
