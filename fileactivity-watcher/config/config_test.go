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
