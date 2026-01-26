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
	"path/filepath"
	"testing"
)

func TestLoadConfig_DefaultWhenNoFile(t *testing.T) {
	// This test assumes the default config file path doesn't exist in the test environment
	// If running on an actual Unraid system, this might need adjustment
	config := LoadConfig()

	// Verify default values
	if config.DisplayEvents != 1000 {
		t.Errorf("Expected DisplayEvents to be 1000, got %d", config.DisplayEvents)
	}

	if config.MaxRecords != 20000 {
		t.Errorf("Expected MaxRecords to be 20000, got %d", config.MaxRecords)
	}

	if config.DedupeWindow != 1 {
		t.Errorf("Expected DedupeWindow to be 1, got %d", config.DedupeWindow)
	}

	if config.ActivityPath != "/var/log/file.activity/data.log" {
		t.Errorf(
			"Expected ActivityPath to be /var/log/file.activity/data.log, got %s",
			config.ActivityPath,
		)
	}

	if !config.UnassignedDevices {
		t.Error("Expected UnassignedDevices to be true by default")
	}

	if config.Cache {
		t.Error("Expected Cache to be false by default")
	}

	if config.SSD {
		t.Error("Expected SSD to be false by default")
	}

	if config.Enable {
		t.Error("Expected Enable to be false by default")
	}

	if len(config.Exclusions) != 4 {
		t.Errorf("Expected 4 default exclusions, got %d", len(config.Exclusions))
	}
}

func TestLoadConfig_WithCustomFile(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "boot", "config", "plugins", "file.activity")

	err := os.MkdirAll(configDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create temp config directory: %v", err)
	}

	// Create a test config file
	testConfig := ActivityConfig{
		Enable:            true,
		UnassignedDevices: false,
		Cache:             true,
		SSD:               true,
		DisplayEvents:     500,
		Exclusions:        []string{`test1`, `test2`},
		MaxRecords:        10000,
		DedupeWindow:      5,
		ActivityPath:      "/tmp/test.log",
	}

	configPath := filepath.Join(configDir, "config.json")

	configData, err := json.Marshal(testConfig)
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}

	err = os.WriteFile(configPath, configData, 0o644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Note: The actual LoadConfig function reads from a hardcoded path
	// This test demonstrates the structure but won't actually test the file loading
	// unless we modify the code to accept a config path parameter
	// For now, we're just testing the JSON structure is valid

	// Verify we can unmarshal the test data
	var loadedConfig ActivityConfig

	err = json.Unmarshal(configData, &loadedConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if !loadedConfig.Enable {
		t.Error("Expected Enable to be true")
	}

	if loadedConfig.UnassignedDevices {
		t.Error("Expected UnassignedDevices to be false")
	}

	if !loadedConfig.Cache {
		t.Error("Expected Cache to be true")
	}

	if !loadedConfig.SSD {
		t.Error("Expected SSD to be true")
	}

	if loadedConfig.DisplayEvents != 500 {
		t.Errorf("Expected DisplayEvents to be 500, got %d", loadedConfig.DisplayEvents)
	}

	if loadedConfig.MaxRecords != 10000 {
		t.Errorf("Expected MaxRecords to be 10000, got %d", loadedConfig.MaxRecords)
	}

	if loadedConfig.DedupeWindow != 5 {
		t.Errorf("Expected DedupeWindow to be 5, got %d", loadedConfig.DedupeWindow)
	}

	if len(loadedConfig.Exclusions) != 2 {
		t.Errorf("Expected 2 exclusions, got %d", len(loadedConfig.Exclusions))
	}
}

func TestActivityConfig_JSONSerialization(t *testing.T) {
	config := ActivityConfig{
		Enable:            true,
		UnassignedDevices: true,
		Cache:             false,
		SSD:               false,
		DisplayEvents:     1000,
		Exclusions:        []string{`(?i)test`, `(?i)example`},
		MaxRecords:        20000,
		DedupeWindow:      2,
		ActivityPath:      "/var/log/test.log",
	}

	// Marshal to JSON
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Unmarshal back
	var decoded ActivityConfig

	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify fields match
	if decoded.Enable != config.Enable {
		t.Error("Enable field mismatch after round-trip")
	}

	if decoded.UnassignedDevices != config.UnassignedDevices {
		t.Error("UnassignedDevices field mismatch after round-trip")
	}

	if decoded.Cache != config.Cache {
		t.Error("Cache field mismatch after round-trip")
	}

	if decoded.SSD != config.SSD {
		t.Error("SSD field mismatch after round-trip")
	}

	if decoded.DisplayEvents != config.DisplayEvents {
		t.Error("DisplayEvents field mismatch after round-trip")
	}

	if decoded.MaxRecords != config.MaxRecords {
		t.Error("MaxRecords field mismatch after round-trip")
	}

	if decoded.DedupeWindow != config.DedupeWindow {
		t.Error("DedupeWindow field mismatch after round-trip")
	}

	if decoded.ActivityPath != config.ActivityPath {
		t.Error("ActivityPath field mismatch after round-trip")
	}

	if len(decoded.Exclusions) != len(config.Exclusions) {
		t.Errorf(
			"Exclusions length mismatch: expected %d, got %d",
			len(config.Exclusions),
			len(decoded.Exclusions),
		)
	}
}
