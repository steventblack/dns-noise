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
	loadConfig(NoiseFlags.ConfigFile)

	// If reusing existing DB, skip the fetch and data import
	// Note that this flag only impacts the *initial* fetch & data import cycle
	// The database will still be refreshed every RefreshPeriod unless that is also disabled
	var domainsDb *sql.DB
	if NoiseFlags.ReuseDatabase {
		log.Println("Reusing existing domains database")
		domainsDb = dbOpen(NoiseConfig.Noise.NoisePath)
	} else {
		noiseFile := fetchDomains(NoiseConfig.Sources[0].Url)
		domainsDb = dbLoadDomains(NoiseConfig.Noise.NoisePath, noiseFile)
	}
	numDomains = dbNumDomains(domainsDb)

	// main loop
	for {
		// periodically check to see if sources need to be refreshed
		// if there was a change, recompute the numDomains available
		if refreshSources(NoiseConfig.Sources) {
			numDomains = dbNumDomains(domainsDb)
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

	log.Printf("Noise query interval %vms based on %d queries over %ds", sleepDuration.Milliseconds(), traffic, period)
	return sleepDuration
}
