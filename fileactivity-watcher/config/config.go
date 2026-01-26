package config

/*
	fileactivity-watcher
	Copyright (C) 2025-2026 Derek Kaser

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

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
	DedupeWindow      int      `json:"dedupe_window,omitempty"`
	ActivityPath      string   `json:"activity_path,omitempty"`
}

func LoadConfig() ActivityConfig {
	appConfig := ActivityConfig{
		Enable:            false,
		UnassignedDevices: true,
		Cache:             false,
		SSD:               false,
		DisplayEvents:     1000,
		Exclusions:        []string{`(?i)appdata`, `(?i)docker`, `(?i)system`, `(?i)syslogs`},
		MaxRecords:        20000,
		DedupeWindow:      1,
		ActivityPath:      "/var/log/file.activity/data.log",
	}

	file, err := os.ReadFile("/boot/config/plugins/file.activity/config.json")
	if err != nil {
		if os.IsNotExist(err) {
			// File does not exist, use default config
			log.Info().Msg("Config file not found, using default configuration")

			return appConfig
		}

		log.Fatal().Err(err).Msg("Error reading config file")
	}

	err = json.Unmarshal(file, &appConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Error parsing config file")
	}

	log.Info().
		Interface("Exclusions", appConfig.Exclusions).
		Bool("Enable", appConfig.Enable).
		Bool("UnassignedDevices", appConfig.UnassignedDevices).
		Bool("Cache", appConfig.Cache).
		Bool("SSD", appConfig.SSD).
		Int("DisplayEvents", appConfig.DisplayEvents).
		Int("MaxRecords", appConfig.MaxRecords).
		Msg("File Activity Watcher Configuration")

	return appConfig
}
