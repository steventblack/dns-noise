package main

import (
	"flag"
	"log"
)

type Flags struct {
	ConfigFile    string
	ReuseDatabase bool
	MinInterval   int
	MaxInterval   int
}

var F *Flags

//
// initilize the flags
//
func init() {
	F = new(Flags)

	// Duplicate references are permitted for providing long ("--conf") and short ("-c") version of a command line arg
	flag.BoolVar(&F.ReuseDatabase, "reusedb", false, "Reuse existing noise database")
	flag.BoolVar(&F.ReuseDatabase, "r", false, "Reuse existing noise database (shorthand)")
	flag.StringVar(&F.ConfigFile, "conf", "dns-noise.conf", "Path to configuration file")
	flag.StringVar(&F.ConfigFile, "c", "dns-noise.conf", "Path to configuration file (shorthand)")
	flag.IntVar(&F.MinInterval, "min", 100, "Minimum interval for issuing noise queries (ms)")
	flag.IntVar(&F.MaxInterval, "max", 5000, "Maximum interval for issuing noise queries (ms)")

	log.Println("Flags successfully initialized")
}
