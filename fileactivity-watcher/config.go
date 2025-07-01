package main

import (
	"encoding/json"
	"os"

	"github.com/rs/zerolog/log"
)

type ActivityConfig struct {
	Enable            bool     `json:"enable,omitempty"`
	UnassignedDevices bool     `json:"unassigned_devices,omitempty"`
	Cache             bool     `json:"cache,omitempty"`
	SSD               bool     `json:"ssd,omitempty"`
	DisplayEvents     int      `json:"display_events,omitempty"`
	Exclusions        []string `json:"exclusions,omitempty"`
	MaxRecords        int      `json:"max_records,omitempty"`
}

var appConfig ActivityConfig = ActivityConfig{
	Enable:            false,
	UnassignedDevices: true,
	Cache:             false,
	SSD:               false,
	DisplayEvents:     1000,
	Exclusions:        []string{`(?i)appdata`, `(?i)docker`, `(?i)system`, `(?i)syslogs`},
	MaxRecords:        20000,
}

func loadConfig() {
	// Load the configuration from /boot/config/plugins/file.activity/config.json
	// If the file does not exist, use the default configuration

	// This function should handle reading the JSON file and unmarshalling it into appConfig
	// If the file is not found, it should initialize appConfig with default values

	file, err := os.ReadFile("/boot/config/plugins/file.activity/config.json")
	if err != nil {
		if os.IsNotExist(err) {
			// File does not exist, use default config
			log.Info().Msg("Config file not found, using default configuration")
			return
		}
		log.Fatal().Err(err).Msg("Error reading config file")
	}
	err = json.Unmarshal(file, &appConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing config file")
	}

	log.Info().Interface("Exclusions", appConfig.Exclusions).Bool("Enable", appConfig.Enable).Bool("UnassignedDevices", appConfig.UnassignedDevices).Bool("Cache", appConfig.Cache).Bool("SSD", appConfig.SSD).Int("DisplayEvents", appConfig.DisplayEvents).Int("MaxRecords", appConfig.MaxRecords).Msg("File Activity Watcher Configuration")
}
