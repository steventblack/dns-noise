package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Flags struct {
	ConfigFile    string
	DbPath        string
	ReuseDatabase bool
	MinPeriod     time.Duration
	MaxPeriod     time.Duration
}

/*
Config contains the configuration information used by the application for customizing its behavior.
The configuration file defaults to a JSON-encoded file named "dns-noise.json" in the current working directory.
It may be overwritten by supplying an alternative filepath using the '-c' or '--conf' command-line option.
  e.g. dns-noise -c /usr/local/etc/dns-noise.conf
The configuration must be expressed as strict JSON, so unfortunately comments in the configuration file are not
supported. JSON has an especially unforgiving syntax structure, so careful attention to the brackets, braces, and commas
is necessary. An example configuration file is included which may be edited/revised as desired.

Here is an annotated reference for the configuration file format:

{
  The "nameservers" block is *optional* and if omitted the system defaults will be used.
  It contains a list of nameservers that will be queried with the noise DNS requests.
  The nameservers will be queried in the order written with the primary used for all initial queries
  and any additional nameservers used only on failover.
  *  Each nameserver entry *must* contain an "ip" element with an IP address in either IPv4 or IPv6 format.
  *  A nameserver entry *may* contain a "port" element with the connection port specified.
     The default port (53) will be used if no port is specified.
  *  A nameserver entry *may* contain a "zone" element *only* with an IPv6 address. The default is to leave the zone unspecified.

  "nameservers":[
    { "ip": "127.0.0.1", "port": 53 },
    { "ip": "::1", zone: "eth0", "port": 53 }
  ],

  The "sources" block is *required* and must have at least one entry defining the source and interpretation rules.
  A source provides a list of domains that will be randomly selected for querying the DNS servers in order to generate noise.
  Each source describes the URL, how to interpret the data, and the refresh policy. All data files must be in CSV form,
  although the application can independently unzip the file if necessary.
  *  Each source entry *must* contain a "url" element specifying the URL for the domains data.
  *  A source *may* contain a "column" element indicating which column in the data file contains the list of domains.
     If unspecified, the default value is 0 which will specify the first column.
  *  A source *may* contain a "label" element to uniquely identify the dataset associated with the source.
     If unspecified, the entire dataset for all sources will be purged when a refresh is triggered.
  *  A source *may* contain a "refresh" element specifying the interval for the domains data to be reloaded from the URL.
     If unspecified, the default behavior will be to never refresh. The interval must be parsable by Go's time.ParseDuration().

  "sources": [
    { "url": "http://example.com/domains/domainlist.csv.zip", "column": 1, "label": "source1", "refresh": "24h" }
  ],

  The "noise" block is *optional* and if omitted the system defaults will be used.
  It contains a set of attributes that define how the application behaves.
  * The "minPeriod" element specifies the minimum interval  permitted for queries. The default value is 100ms.
    A command-line argument specifying the minPeriod will overwrite the default or configuration value.
    The period must be parsable by Go's time.ParseDuration() and be less than that of the maxPeriod.
  * The "maxPeriod" element specifies the maximum interval permitted for queries. The default value is 15s.
    A command-line argument specifying the maxPeriod will overwrite the default or configuration value.
    The period must be parsable by Go's time.ParseDuration() and be greater than that of minPeriod.
  * The "dbPath" element specifies the path to locate the database containing the list of domains.
    The default location is in the system's tempory directory with the filename of "dns-noise.db".
    The location must have permissions for file creation and write access.
    A command-line argument specifying the path will overwrite the default or configuration value.
  * The "ipv4" element is a boolean flag indicating whether DNS request for the IPv4 address should be utilized.
    This is a request for the "A" record from the DNS zone and is not dependent on using an IPv4 or IPv6 network.
    The default value is true.
  * The "ipv6" element is a boolean flag indicating whether DNS request for the IPv6 address should be utilized.
    This is a request for the "AAAA" record from the DNS zone and is not dependent on using an IPv4 or IPv6 network.
    The default value is false.

  "noise": {
    "minPeriod": "100ms",
    "maxPeriod": "15s",
    "dbPath": "/tmp/dns-noise.db",
    "ipv4": true,
    "ipv6": true
  },

  The "pihole" block is *optional* and if omitted the application will not utilize pihole activity for determining noise thresholds.
  If the pihole block is incomplete or incorrectly configured, the pihole will not be utilized. If the pihole is not
  used to determine the rate of DNS queries, then random values between the minPeriod and maxPeriod will be used. The pihole
  authtoken value can be found in the "/etc/pihole/setupVars.conf" file as the value for the "WEBPASSWORD" option. The
  token should be treated with appropriate security precautions and restrict access.
  * The "host" element *must* specify the hostname or IP address of the pihole server. The pihole must be listening on that interface,
    so check the pihole settings especially if running the noise generator on the same host as the pihole.
    If the host is not specified, pihole activity will not be enabled.
  * The "authToken" element *must* contain the encrypted web password for accessing the pihole's admin API. Please note that the queries
    to the pihole are sent *unencrypted* and the token value is accessible to traffic sniffers as the pihole does not support https.
    Do *not* use if there is even a remote chance of untrusted actors on the network.
  * The "activityPeriod" element *may* specify the time interval used to calculate the running average for the pihole query activity.
    The default is use a 5 minute window for examining query activity. The interval must be parsable by Go's time.ParseDuration().
  * The "refresh" element *may* specify the frequency the pihole will be queried to calculate the moving average.
    The default refresh frequency is 1 minute. The frequency must be parsable by Go's time.ParseDuration().
  * The "filter" element *may* specify a hostname that is used to exclude activity from the moving average.
    This may be desired in order to exclude the queries originating from the DNS noise host in order to just report on the "live" traffic.
  * The "noisePercentage" element *may* be specified and must be in the range of 1-100 for the pihole functionality to be enabled.
    This element allows the noise generator to dynamically adjust its traffic levels to the stated percentage of "live" traffic.
    The default value is 10. Do not include a percentage sign (%) with the value.

  "pihole": {
    "host": "pihole.example.com",
    "authToken": "pihole_authtoken_goes_here",
    "activityPeriod": "5m",
    "refresh": "1m",
    "filter": "noise.example.com",
    "noisePercentage": 10
  }

	The "metrics" block is *optional* and if omitted the application will not emit any metrics for scraping.
	If the metrics block is incorrectly formatted, it may result in a panic upon service launch or difficulty in scraping.
	The metrics are exported on the designated port and path in standard prometheus text format. They can be manually
	inspected by pointing your browser to the apprporiate URL. (e.g. "http://noise.example.com:6001/metrics")
  * The "enabled" element *may* be specified with a boolean (true/false) value. The default value is false.
  * The "port" element *may* be specified. The default value is 6001. Care should be made when selecting a port
    to pick a port that is not already in use on that host or in a restricted range.
  *	The "path" element *may* be specified. The default value is "/metrics" as that is the convential path for Prometheus
   	log scraping. Access to the path should be restricted to external networks as part of good security practices.

	"metrics": {
		"enabled": false,
		"port": 6001,
		"path": "/metrics"
	}
}
*/
type Config struct {
	NameServers []NameServer `json:"nameservers"`
	Noise       Noise        `json:"noise"`
	Sources     []Source     `json:"sources"`
	Pihole      Pihole       `json:"pihole"`
	Metrics     Metrics      `json:"metrics"`
}

type NameServer struct {
	Ip   string `json:"ip"`
	Zone string `json:"zone"`
	Port int    `json:"port"`
}

// UnmarshalJSON provides an interface for customized processing of the NameServer struct.
// It performs initialization of select fields to default values prior to the actual unmarshaling.
// The default values will be overwritten if present in the JSON blob.
func (ns *NameServer) UnmarshalJSON(data []byte) error {
	ns.Port = 53

	// Need to avoid circular looping here
	type Alias NameServer
	tmp := (*Alias)(ns)

	return json.Unmarshal(data, tmp)
}

type Noise struct {
	DbPath    string   `json:"dbPath"`
	MinPeriod Duration `json:"minPeriod"`
	MaxPeriod Duration `json:"maxPeriod"`
	IPv4      bool     `json:ipv4"`
	IPv6      bool     `json:ipv6"`
}

// UnmarshalJSON provides an interface for customized processing of the Noise struct.
// It performs initialization of select fields to default values prior to the actual unmarshaling.
// The default values will be overwritten if present in the JSON blob.
func (n *Noise) UnmarshalJSON(data []byte) error {
	n.IPv4 = true
	n.DbPath = filepath.Join(os.TempDir(), "dns-noise.db")
	n.MinPeriod, _ = parseDuration("100ms")
	n.MaxPeriod, _ = parseDuration("15s")

	// Need to avoid circular looping here
	type Alias Noise
	tmp := (*Alias)(n)

	return json.Unmarshal(data, tmp)
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

// UnmarshalJSON provides an interface for customized processing of the Pihole struct.
// It performs initialization of select fields to default values prior to the actual unmarshaling.
// The default values will be overwritten if present in the JSON blob.
func (p *Pihole) UnmarshalJSON(data []byte) error {
	p.NoisePercentage = 10
	p.ActivityPeriod, _ = parseDuration("5m")
	p.Refresh, _ = parseDuration("1m")

	// Need to avoid circular looping here
	type Alias Pihole
	tmp := (*Alias)(p)

	return json.Unmarshal(data, tmp)
}

type Metrics struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path"`
	Port    int    `json:"port"`
}

// UnmarshalJSON provides an interface for customized processing of the Metrics struct.
// It performs initialization of select fields to default values prior to the actual unmarshaling.
// The default values will be overwritten if present in the JSON blob.
func (m *Metrics) UnmarshalJSON(data []byte) error {
	m.Port = 6001
	m.Enabled = false
	m.Path = "metrics"

	type Alias Metrics
	tmp := (*Alias)(m)

	return json.Unmarshal(data, tmp)
}

// loadFlags parses the CLI arguments passed into the Flags structure.
// Unrecognized flags will be ignored.
// An initialized Flags struct will be returned which contains either the passed in values or defaults.
func loadFlags() *Flags {
	f := new(Flags)

	// set default interval values
	f.MinPeriod, _ = time.ParseDuration("100ms")
	f.MaxPeriod, _ = time.ParseDuration("15000ms")

	// Duplicate references are permitted for providing long ("--conf") and short ("-c") version of a command line arg
	flag.BoolVar(&f.ReuseDatabase, "reusedb", false, "Reuse existing noise database")
	flag.BoolVar(&f.ReuseDatabase, "r", false, "Reuse existing noise database (shorthand)")
	flag.StringVar(&f.ConfigFile, "conf", "dns-noise.json", "Path to configuration file")
	flag.StringVar(&f.ConfigFile, "c", "dns-noise.json", "Path to configuration file (shorthand)")
	flag.StringVar(&f.DbPath, "database", "/tmp/dns-noise.db", "Path to noise database file")
	flag.StringVar(&f.DbPath, "d", "/tmp/dns-noise.db", "Path to noise database file (shorthand)")
	flag.DurationVar(&f.MinPeriod, "min", f.MinPeriod, "Minimum time period for issuing noise queries")
	flag.DurationVar(&f.MaxPeriod, "max", f.MaxPeriod, "Maximum time period for issuing noise queries")

	// process the flags passed in on the CLI
	flag.Parse()

	return f
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
// If successful, the processed configuration will be returned. If an error is encountered, it will be treated as a fatal error.
func loadConfig(flags *Flags) *Config {
	jsonFile, err := os.Open(flags.ConfigFile)
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
		c.Noise.MinPeriod = Duration(flags.MinPeriod)
	}
	if isFlagPassed("max") {
		c.Noise.MaxPeriod = Duration(flags.MaxPeriod)
	}
	if isFlagPassed("database") || isFlagPassed("d") {
		c.Noise.DbPath = flags.DbPath
	}

	// bad config! no soup for you!
	if c.Noise.MinPeriod > c.Noise.MaxPeriod {
		log.Fatal("Min period exceeds max period")
	}

	return c
}

// The Duration type provides enables the JSON module to process strings as time.Durations.
// While time.Duration is available as a native type for CLI flags, it is not for the JSON parser.
// Note that in Go, you cannot define new methods on a non-local type so this workaround is the
// best alternative to hacking directly in the standard Go time module.
type Duration time.Duration

// Duration returns the time.Duration native type of the time module.
// This helper function makes it slightly less tedious to continually typecast a Duration into a time.Duration
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// ParseDuration is a helper function to parse a string utilizing the underlying time.ParseDuration functionality.
func parseDuration(s string) (Duration, error) {
	td, err := time.ParseDuration(s)
	if err != nil {
		return Duration(0), err
	}

	return Duration(td), nil
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
