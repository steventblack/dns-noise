//
// Copyright 2020 Steven T Black
//

package main

import (
	crypto_rand "crypto/rand"
	"database/sql"
	"encoding/binary"
	"flag"
	"log"
	math_rand "math/rand"
	"time"
)

var dbRefreshInterval float64 = 24.0
var dbPath = "/tmp/dns-noise.db"
var domainsURL = "http://s3-us-west-1.amazonaws.com/umbrella-static/top-1m.csv.zip"
var piholeRefreshInterval = 60
var piholeQueryDuration = 300
var noisePercentage = 10
var numDomains int
var numQueries int

//
// Initializer for rand
// Generates a better seed value than simply relying on a time value
//
func init() {
	var b [8]byte
	_, err := crypto_rand.Read(b[:])
	if err != nil {
	}

	math_rand.Seed(int64(binary.LittleEndian.Uint64(b[:])))
}

func main() {
	// Read in all (any) of the command line flags
	flag.Parse()

	// If reusing existing DB, skip the fetch and data import
	// Note that this flag only impacts the *initial* fetch & data import cycle
	// The database will still be refreshed every dbRefreshInterval unless that is also disabled
	var domainsDb *sql.DB
	if NoiseFlags.ReuseDatabase {
		log.Println("Reusing existing domains database")
		domainsDb = dbOpen(dbPath)
	} else {
		noiseFile := fetchDomains(domainsURL)
		domainsDb = dbLoadDomains(dbPath, noiseFile)
	}
	numDomains = dbNumDomains(domainsDb)
	dbRefreshTime := time.Now()

	// main loop
	// referesh the domains database every 24h (dbRefreshInterval) unless refresh disabled
	for {
		if (dbRefreshInterval > 0) && (time.Since(dbRefreshTime).Hours() > dbRefreshInterval) {
			noiseFile := fetchDomains(domainsURL)
			domainsDb = dbLoadDomains(dbPath, noiseFile)
			numDomains = dbNumDomains(domainsDb)
			dbRefreshTime = time.Now()
			log.Printf("Refreshed domains database; %d domains available", numDomains)
		}

		var timeUntil = time.Now().Unix()
		var timeFrom = timeUntil - int64(piholeQueryDuration)
		numQueries = piholeFetchQueries(timeFrom, timeUntil)
		sleepDuration := calcSleepDuration(numQueries, piholeQueryDuration, noisePercentage)

		// inner loop; lookup random domains and sleep a bit in between calls
		// break to main loop when time exceeds pihole refresh interval
		for {
			if time.Since(time.Unix(timeUntil, 0)).Seconds() > float64(piholeRefreshInterval) {
				break
			}

			piholeLookupDomain(dbGetRandomDomain(domainsDb))

			// add a bit of randomness between calls to increase the "noisiness"
			// this should generate a random number of ms from 0 to 10% of the sleepDuration value
			sleepDelta := time.Duration(math_rand.Int63n(sleepDuration.Milliseconds()/10)) * time.Millisecond
			time.Sleep(sleepDuration + sleepDelta)
		}
	}
}

func calcSleepDuration(traffic, period, noiseLevel int) time.Duration {
	// keep the math in the defined world
	if traffic <= 0 {
		traffic = 1
	}

	// period (in ms) divided by the amount of traffic times the noise level
	sleepDuration := time.Duration(period*1000/traffic*noiseLevel) * time.Millisecond

	log.Printf("Base sleep time %vms based on %d queries over %ds", sleepDuration.Milliseconds(), traffic, period)
	return sleepDuration
}
