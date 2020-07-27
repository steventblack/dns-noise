package main

import (
	"flag"
	"log"
	"time"
)

type Flags struct {
	ConfigFile    string
	ReuseDatabase bool
	MinInterval   time.Duration
	MaxInterval   time.Duration
}

var NoiseFlags *Flags

//
// initilize the flags
//
func init() {
	f := new(Flags)

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
