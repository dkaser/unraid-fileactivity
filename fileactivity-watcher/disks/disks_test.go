package disks

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

	"github.com/dkaser/unraid-fileactivity/fileactivity-watcher/config"
	"gopkg.in/ini.v1"
)

func TestIsValidDiskType(t *testing.T) {
	tests := []struct {
		name     string
		diskType string
		expected bool
	}{
		{"data disk", "data", true},
		{"cache disk", "cache", true},
		{"invalid type", "parity", false},
		{"empty string", "", false},
		{"random string", "random", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidDiskType(tt.diskType)
			if result != tt.expected {
				t.Errorf("isValidDiskType(%q) = %v, expected %v", tt.diskType, result, tt.expected)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()

	disksIniPath := filepath.Join(tmpDir, "disks.ini")
	disksIni := ini.Empty()

	section, _ := disksIni.NewSection("disk1")
	section.NewKey("name", "disk1")
	section.NewKey("type", "data")
	section.NewKey("fsType", "xfs")
	section.NewKey("rotational", "true")

	err := disksIni.SaveTo(disksIniPath)
	if err != nil {
		t.Fatalf("Failed to create test disks.ini: %v", err)
	}

	udDir := filepath.Join(tmpDir, "unassigned.devices")

	err = os.MkdirAll(udDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create unassigned devices directory: %v", err)
	}

	udPath := filepath.Join(udDir, "unassigned.devices.json")
	udData := map[string]UDInfo{
		"sdb1": {
			Mountpoint: "/mnt/disks/test",
			Mounted:    true,
			Fstype:     "ext4",
		},
	}

	udJSON, _ := json.Marshal(udData)

	err = os.WriteFile(udPath, udJSON, 0o644)
	if err != nil {
		t.Fatalf("Failed to create unassigned devices JSON: %v", err)
	}

	appConfig := config.ActivityConfig{
		UnassignedDevices: true,
		Cache:             true,
		SSD:               false,
	}

	_ = appConfig
}

func TestGetWatchFolders(t *testing.T) {
	appConfig := config.ActivityConfig{
		UnassignedDevices: false,
		Cache:             false,
		SSD:               false,
	}

	d := &Disks{
		appConfig: appConfig,
		arrayDisks: []Disk{
			{
				Name:       "disk1",
				Mountpoint: "/mnt/disk1",
				Type:       "data",
				Filesystem: "xfs",
				Rotational: true,
			},
			{
				Name:       "disk2",
				Mountpoint: "/mnt/disk2",
				Type:       "data",
				Filesystem: "xfs",
				Rotational: true,
			},
		},
		poolDisks: []Disk{
			{
				Name:       "cache",
				Mountpoint: "/mnt/cache",
				Type:       "cache",
				Filesystem: "btrfs",
				Rotational: false,
			},
		},
		unassignedDisks: []Disk{
			{
				Name:       "sdb1",
				Mountpoint: "/mnt/disks/test",
				Type:       "unassigned",
				Filesystem: "ext4",
				Rotational: true,
			},
		},
	}

	watchFolders := d.GetWatchFolders()

	expectedCount := 4
	if len(watchFolders) != expectedCount {
		t.Errorf("Expected %d watch folders, got %d", expectedCount, len(watchFolders))
	}

	expectedFolders := []string{"/mnt/disk1", "/mnt/disk2", "/mnt/cache", "/mnt/disks/test"}
	for _, folder := range expectedFolders {
		if _, exists := watchFolders[folder]; !exists {
			t.Errorf("Expected watch folder %s not found", folder)
		}
	}
}

func TestDisk_Struct(t *testing.T) {
	disk := Disk{
		Name:       "disk1",
		Mountpoint: "/mnt/disk1",
		Type:       "data",
		Filesystem: "xfs",
		Rotational: true,
	}

	if disk.Name != "disk1" {
		t.Errorf("Expected Name to be 'disk1', got %s", disk.Name)
	}

	if disk.Mountpoint != "/mnt/disk1" {
		t.Errorf("Expected Mountpoint to be '/mnt/disk1', got %s", disk.Mountpoint)
	}

	if disk.Type != "data" {
		t.Errorf("Expected Type to be 'data', got %s", disk.Type)
	}

	if disk.Filesystem != "xfs" {
		t.Errorf("Expected Filesystem to be 'xfs', got %s", disk.Filesystem)
	}

	if !disk.Rotational {
		t.Error("Expected Rotational to be true")
	}
}

func TestUDInfo_Struct(t *testing.T) {
	udInfo := UDInfo{
		Mountpoint: "/mnt/disks/test",
		Mounted:    true,
		Fstype:     "ext4",
	}

	if udInfo.Mountpoint != "/mnt/disks/test" {
		t.Errorf("Expected Mountpoint to be '/mnt/disks/test', got %s", udInfo.Mountpoint)
	}

	if !udInfo.Mounted {
		t.Error("Expected Mounted to be true")
	}

	if udInfo.Fstype != "ext4" {
		t.Errorf("Expected Fstype to be 'ext4', got %s", udInfo.Fstype)
	}
}

func TestUDInfo_JSONSerialization(t *testing.T) {
	udInfo := UDInfo{
		Mountpoint: "/mnt/disks/test",
		Mounted:    true,
		Fstype:     "ext4",
	}

	data, err := json.Marshal(udInfo)
	if err != nil {
		t.Fatalf("Failed to marshal UDInfo: %v", err)
	}

	var decoded UDInfo

	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal UDInfo: %v", err)
	}

	if decoded.Mountpoint != udInfo.Mountpoint {
		t.Error("Mountpoint field mismatch after round-trip")
	}

	if decoded.Mounted != udInfo.Mounted {
		t.Error("Mounted field mismatch after round-trip")
	}

	if decoded.Fstype != udInfo.Fstype {
		t.Error("Fstype field mismatch after round-trip")
	}
}
