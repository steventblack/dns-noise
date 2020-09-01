package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
)

type Flags struct {
	ConfigFile    string
	DbPath        string
	ReuseDatabase bool
	MinPeriod     time.Duration
	MaxPeriod     time.Duration
}

type Config struct {
	NameServers []NameServer `json:"nameservers"`
	Noise       Noise        `json:"noise"`
	Sources     []Source     `json:"sources"`
	Pihole      Pihole       `json:"pihole"`
}

type NameServer struct {
	Ip   string `json:"ip"`
	Zone string `json:"zone"`
	Port int    `json:"port"`
}

type Noise struct {
	DbPath    string   `json:"dbPath"`
	Refresh   Duration `json:"refresh"`
	MinPeriod Duration `json:"minPeriod"`
	MaxPeriod Duration `json:"maxPeriod"`
	IPv4      bool     `json:ipv4"`
	IPv6      bool     `json:ipv6"`
}

type Source struct {
	Label     string   `json:"label"`
	Url       string   `json:"url"`
	Column    int      `json:"column"`
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
	Enabled         bool
	Timestamp       time.Time
	SleepPeriod     time.Duration
}

var NoiseFlags *Flags
var NoiseConfig *Config

// init establishes the flag set and initializes the flags to their default values.
// These values will be replaced if an explicit flag is passed on the command line.
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
	flag.StringVar(&f.DbPath, "database", "/tmp/dns-noise.db", "Path to noise database file")
	flag.StringVar(&f.DbPath, "d", "/tmp/dns-noise.db", "Path to noise database file (shorthand)")
	flag.DurationVar(&f.MinPeriod, "min", f.MinPeriod, "Minimum time period for issuing noise queries")
	flag.DurationVar(&f.MaxPeriod, "max", f.MaxPeriod, "Maximum time period for issuing noise queries")

	// Set public pointer
	NoiseFlags = f
	log.Println("Flags successfully initialized")
}

// isFlagPassed checks to see if the named flag was explicitly passed on the command line or not.
// It returns a bool reflecting whether is was passed or not.
func isFlagPassed(flagName string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == flagName {
			found = true
		}
	})

	return found
}

// loadConfig reads in and parses the named file for the configuration values.
// The file is expected to be in JSON format. Command line flags will overwrite the values (if any) found in the configuration.
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

	// checks to see if necessary elements for Pihole access are present
	c.Pihole.Enabled = piholeEnabled(&c.Pihole)

	// overwrite config vars that were set explicitly with a command-line flag
	if isFlagPassed("min") {
		c.Noise.MinPeriod = Duration(NoiseFlags.MinPeriod)
	}
	if isFlagPassed("max") {
		c.Noise.MaxPeriod = Duration(NoiseFlags.MaxPeriod)
	}
	if isFlagPassed("database") || isFlagPassed("d") {
		c.Noise.DbPath = NoiseFlags.DbPath
	}

	// bad config! no soup for you!
	if c.Noise.MinPeriod > c.Noise.MaxPeriod {
		log.Fatal("Min period exceeds max period")
	}

	NoiseConfig = c
}

// piholeEnabled checks the necessary settings are present in the config for pihole utilization.
// It does not perform any validation checks on the setting values.
// It returns a bool reflecting the configuration is setup or not.
func piholeEnabled(p *Pihole) bool {
	enabled := true

	if p.Host == "" {
		enabled = false
	}
	if p.AuthToken == "" {
		enabled = false
	}
	if p.NoisePercentage <= 0 {
		enabled = false
	}

	return enabled
}

// The Duration type provides enables the JSON module to process strings as time.Durations.
// While time.Duration is available as a native type for CLI flags, it is not for the JSON parser.
type Duration time.Duration

// Duration returns the time.Duration native type of the time module.
// This helper function makes it slightly less tedious to continually typecast a Duration into a time.Duration
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// MarshalJSON supplies an interface for processing Duration values which wrap the standard time.Duration type.
// It returns a byte array and any error encountered.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON supplies an interface for processing Duration values which wrap the standard time.Duration type.
// It accepts a byte array and returns any error encountered.
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
		return fmt.Errorf("Invalid Duration specification: '%v'", value)
	}
}
