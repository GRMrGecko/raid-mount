package main

import (
	"flag"
	"fmt"
	"os"
)

// Flags: Configuration options for cli execution.
type Flags struct {
	ConfigPath         string
	EncryptionKey      string
	EncryptionPassword string
}

// Init: Parses configuration options.
func (f *Flags) Init() {
	flag.Usage = func() {
		fmt.Printf("raid-mount: Mounts raid drives and starts services\n\nUsage:\n")
		flag.PrintDefaults()
	}

	var printVersion bool
	flag.BoolVar(&printVersion, "v", false, "Print version")

	var usage string
	usage = "Load configuration from `FILE`"
	flag.StringVar(&f.ConfigPath, "config", "", usage)
	flag.StringVar(&f.ConfigPath, "c", "", usage+" (shorthand)")

	flag.StringVar(&f.EncryptionKey, "encryption-key", "", "Keyfile to decrypt drives")
	usage = "Password to decrypt drives"
	flag.StringVar(&f.EncryptionPassword, "encryption-password", "", usage)
	flag.StringVar(&f.EncryptionPassword, "p", "", usage+" (shorthand)")

	flag.Parse()

	if printVersion {
		fmt.Println("raid-mount: 0.1")
		os.Exit(0)
	}
}
