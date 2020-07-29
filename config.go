package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"time"
)

type Flags struct {
	ConfigFile    string
	ReuseDatabase bool
	MinInterval   time.Duration
	MaxInterval   time.Duration
}

type Config struct {
	NoiseDb NoiseDb `json:"noisedb"`
	Pihole  Pihole  `json:"pihole"`
}

type NoiseDb struct {
	Path          string  `json:"dbPath"`
	RefreshPeriod float64 `json:"dbRefreshPeriod"`
	Source        string  `json:"dbSource"`
}

type Pihole struct {
	PiholeHost      string `json:"phHost"`
	AuthToken       string `json:"phAuthToken"`
	QueryTimespan   int    `json:"phQueryTimespan"`
	FilterHost      string `json:"phFilterHost"`
	NoisePercentage int    `json:"phNoisePercentage"`
}

var NoiseFlags *Flags
var NoiseConfig *Config

//
// initialize the flags
//
func init() {
	f := new(Flags)

	// set default interval values
	f.MinInterval, _ = time.ParseDuration("100ms")
	f.MaxInterval, _ = time.ParseDuration("10000ms")

	// Duplicate references are permitted for providing long ("--conf") and short ("-c") version of a command line arg
	flag.BoolVar(&f.ReuseDatabase, "reusedb", false, "Reuse existing noise database")
	flag.BoolVar(&f.ReuseDatabase, "r", false, "Reuse existing noise database (shorthand)")
	flag.StringVar(&f.ConfigFile, "conf", "dns-noise.conf", "Path to configuration file")
	flag.StringVar(&f.ConfigFile, "c", "dns-noise.conf", "Path to configuration file (shorthand)")
	flag.DurationVar(&f.MinInterval, "min", f.MinInterval, "Minimum interval for issuing noise queries")
	flag.DurationVar(&f.MaxInterval, "max", f.MaxInterval, "Maximum interval for issuing noise queries")

	// Set public pointer
	NoiseFlags = f
	log.Println("Flags successfully initialized")
}

func loadConfig() {
	jsonFile, err := os.Open("dns-noise.json")
	if err != nil {
		log.Fatal(err.Error())
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	c := new(Config)
	err = json.Unmarshal(byteValue, c)
	if err != nil {
		log.Fatal(err.Error())
	}

	NoiseConfig = c
}
