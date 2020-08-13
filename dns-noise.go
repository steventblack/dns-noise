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

var numDomains int

//
// Initializer for rand
// Generates a better seed value than simply relying on a time value
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
		domainsDb = dbOpen(NoiseConfig.Noise.DbPath)
	} else {
		noiseFile := fetchDomains(NoiseConfig.Sources[0].Url)
		domainsDb = dbLoadDomains(NoiseConfig.Noise.DbPath, noiseFile)
	}
	numDomains = dbNumDomains(domainsDb)

	// main loop
	for {
		// periodically check to see if sources need to be refreshed
		// if there was a change, recompute the numDomains available
		if refreshSources(NoiseConfig.Sources) {
			numDomains = dbNumDomains(domainsDb)
		}

		// fetch a random domain and issue a DNS query
		piholeLookupDomain(dbGetRandomDomain(domainsDb))

		// sleep between calls to moderate the query rate
		time.Sleep(calcSleepPeriod(NoiseConfig))
	}
}

// calcSleepPeriod determines an appropriate sleep duration between noise queries.
// If a pihole is properly configured, it will use a percentage of the live traffic rate as the basis.
// The pihole activity rate will be adjusted to fall within the min/max period if necessary.
// If a pihole is not configured, a random value between the min and max period will be generated.
// For additional obfuscation, a random value between 0-10% of the raw sleep period for each call will be added.
func calcSleepPeriod(c *Config) time.Duration {
	var sleepPeriod time.Duration

	if c.Pihole.Enabled {
		if time.Since(c.Pihole.Timestamp) > c.Pihole.Refresh {
			if c.Pihole.Timestamp.IsZero() {
				log.Println("Initialized pihole timestamp")
				c.Pihole.Timestamp = time.Now()
			}

			// if no activity, an error will be returned
			numQueries, err := piholeFetchActivity(&c.Pihole)
			log.Printf("Refreshed pihole activity data; %d queries", numQueries)
			if err != nil {
				c.Pihole.SleepPeriod = time.Duration(0)
			} else {
				c.Pihole.SleepPeriod = time.Duration(int64(c.Pihole.ActivityPeriod) * int64(c.Pihole.NoisePercentage) / int64(numQueries))
			}

			// if the interval time calculate by pihole activity exceeds limits, then cap appropriately
			if c.Pihole.SleepPeriod > c.Noise.MaxPeriod {
				c.Pihole.SleepPeriod = c.Noise.MaxPeriod
			} else if c.Pihole.SleepPeriod < c.Noise.MinPeriod {
				c.Pihole.SleepPeriod = c.Noise.MinPeriod
			}

			c.Pihole.Timestamp = time.Now()
		}

		sleepPeriod = c.Pihole.SleepPeriod
	} else {
		sleepRange := int64(c.Noise.MaxPeriod - c.Noise.MinPeriod)
		sleepPeriod = time.Duration(math_rand.Int63n(sleepRange)) + c.Noise.MinPeriod
	}

	sleepDelta := time.Duration(math_rand.Int63n(sleepPeriod.Milliseconds()/10)) * time.Millisecond

	return sleepPeriod + sleepDelta
}
