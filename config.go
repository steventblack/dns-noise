package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"time"
)

type Flags struct {
	ConfigFile    string
	ReuseDatabase bool
	MinPeriod     time.Duration
	MaxPeriod     time.Duration
}

type Config struct {
	Noise   Noise    `json:"noise"`
	Sources []Source `json:"sources"`
	Pihole  Pihole   `json:"pihole"`
}

type Noise struct {
	NoisePath string   `json:"noisePath"`
	MinPeriod Duration `json:"minPeriod"`
	MaxPeriod Duration `json:"maxPeriod"`
}

type Source struct {
	Label     string   `json:"label"`
	Url       string   `json:"url"`
	Refresh   Duration `json:"refresh"`
	Timestamp time.Time
}

type Pihole struct {
	Host            string   `json:"host"`
	AuthToken       string   `json:"authToken"`
	ActivityPeriod  Duration `json:"activityPeriod"`
	Refresh         Duration `json:"refresh"`
	Filter          string   `json:"filter"`
	NoisePercentage int      `json:"noisePercentage"`
}

var NoiseFlags *Flags
var NoiseConfig *Config

//
// initialize the flags
//
func init() {
	f := new(Flags)

	// set default interval values
	f.MinPeriod, _ = time.ParseDuration("100ms")
	f.MaxPeriod, _ = time.ParseDuration("10000ms")

	// Duplicate references are permitted for providing long ("--conf") and short ("-c") version of a command line arg
	flag.BoolVar(&f.ReuseDatabase, "reusedb", false, "Reuse existing noise database")
	flag.BoolVar(&f.ReuseDatabase, "r", false, "Reuse existing noise database (shorthand)")
	flag.StringVar(&f.ConfigFile, "conf", "dns-noise.json", "Path to configuration file")
	flag.StringVar(&f.ConfigFile, "c", "dns-noise.json", "Path to configuration file (shorthand)")
	flag.DurationVar(&f.MinPeriod, "min", f.MinPeriod, "Minimum time period for issuing noise queries")
	flag.DurationVar(&f.MaxPeriod, "max", f.MaxPeriod, "Maximum time period for issuing noise queries")

	// Set public pointer
	NoiseFlags = f
	log.Println("Flags successfully initialized")
}

//
// Check to see if a flag was explicitly passed. This can then be used to override the equivalent value in the config (if applicable)
//
func isFlagPassed(flagName string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == flagName {
			found = true
		}
	})

	return found
}

//
// load the config from the json file
//
func loadConfig(confFile string) {
	jsonFile, err := os.Open(confFile)
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

//
// Interface functions for custom JSON handling of time.Duration fields
//
type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return errors.New("invalid duration")
	}
}
