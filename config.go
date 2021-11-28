package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
)

// Config: Configuration structure.
type Config struct {
	RaidTablePath string   `json:"raid_table_path"`
	Services      []string `json:"services"`

	EncryptionKey string `json:"encryption_key"`
}

// ReadConfig: Read the configuration file.
func (a *App) ReadConfig() {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	// Configuration paths.
	localConfig, _ := filepath.Abs("./config.json")
	homeDirConfig := usr.HomeDir + "/.config/raid-mount/config.json"
	etcConfig := "/etc/raid-mount/config.json"

	// Default config.
	app.config = Config{
		RaidTablePath: "/etc/raid-mount/raidtab",
	}

	// Determine which configuration to use.
	var configFile string
	if _, err := os.Stat(app.flags.ConfigPath); err == nil && app.flags.ConfigPath != "" {
		configFile = app.flags.ConfigPath
	} else if _, err := os.Stat(localConfig); err == nil {
		configFile = localConfig
	} else if _, err := os.Stat(homeDirConfig); err == nil {
		configFile = homeDirConfig
	} else if _, err := os.Stat(etcConfig); err == nil {
		configFile = etcConfig
	} else {
		log.Println("Unable to find a configuration file.")
		return
	}

	jsonFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("Error reading JSON file: %s\n", err)
		return
	}

	err = json.Unmarshal(jsonFile, &app.config)
	if err != nil {
		fmt.Printf("Error parsing JSON file: %s\n", err)
	}
}
