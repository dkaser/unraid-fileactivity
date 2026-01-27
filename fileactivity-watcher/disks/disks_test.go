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
	"testing"
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
