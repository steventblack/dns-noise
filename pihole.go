//
// Copyright 2020 Steven T Black
//

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
)

// Example response from Pihole
// It's not particularly well-structured JSON, but it'll do for this purpose
// {"data":[["1593882001","AAAA","trk.pinterest.com","impala.local","1","0","4","8","N\/A","-1"],["1593882001","A","trk.pinterest.com","impala.local","1","0","4","7","N\/A","-1"]]}
type PiholeQueries struct {
	Data [][]string
}

func piholeFetchQueries(from, until int64) int {
	// TODO: insert check for auth token here; if absent return 0

	url := fmt.Sprintf("http://%s/admin/api.php?getAllQueries&from=%d&until=%d&auth=%s", NoiseConfig.Pihole.Host, from, until, NoiseConfig.Pihole.AuthToken)

	response, err := http.Get(url)
	if err != nil {
		log.Fatal("Unable to fetch query data from pihole; status: '%s'", response.Status)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		log.Fatal("Unexpected status fetching query data from pihole; status: '%s'", response.Status)
	}

	jsonBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err.Error())
	}

	var queries PiholeQueries
	err = json.Unmarshal(jsonBody, &queries)
	if err != nil {
		log.Fatal(err.Error())
	}

	numQueries := piholeFilterNoise(queries.Data)
	log.Printf("Retrieved pihole activity: %d queries", numQueries)

	return numQueries
}

//
// Filters out queries that were generated as part of noise
// Returns the number of "legitimate" DNS queries during the period
// Assumes noise generating system doesn't make significant amount of legitimate DNS queries
// Somewhat brittle implementation; the pihole API response doesn't have a lot of structure
//
func piholeFilterNoise(queries [][]string) int {
	var numQueries int
	for _, query := range queries {
		if !strings.HasPrefix(query[3], NoiseConfig.Pihole.Filter) {
			numQueries++
		}
	}

	// Safety measure in case no traffic found
	if numQueries == 0 {
		numQueries = 1
	}

	return numQueries
}

//
// Query the pihole using the provided domain and query type
//
func piholeLookupDomain(domain string) {
	if domain == "" {
		log.Println("Cannot lookup empty domain; skipping")
		return
	}

	_, err := net.LookupHost(domain)

	// Lookup failures are expected as the pihole blocks a number of ad and tracker domains
	// Log them anyway in case something unexpected is returned
	if err != nil {
		log.Println(err.Error())
	}
}
