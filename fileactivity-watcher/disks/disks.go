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
	"strings"

	"github.com/dkaser/unraid-fileactivity/fileactivity-watcher/config"
	"github.com/rs/zerolog/log"
	"gopkg.in/ini.v1"
)

type Disks struct {
	arrayDisks      []Disk
	poolDisks       []Disk
	unassignedDisks []Disk
	watchDisks      []Disk
	appConfig       config.ActivityConfig
}

func New(appConfig config.ActivityConfig) *Disks {
	disks := &Disks{
		appConfig: appConfig,
	}
	disks.loadDisks()
	disks.loadUnassignedDisks()

	return disks
}

func (d *Disks) GetWatchFolders() map[string]int {
	watchFolders := make(map[string]int)

	d.watchDisks = d.arrayDisks
	d.watchDisks = append(d.watchDisks, d.poolDisks...)
	d.watchDisks = append(d.watchDisks, d.unassignedDisks...)

	for _, disk := range d.watchDisks {
		log.Info().
			Str("disk", disk.Name).
			Str("mountpoint", disk.Mountpoint).
			Str("type", disk.Type).
			Str("filesystem", disk.Filesystem).
			Bool("rotational", disk.Rotational).
			Msg("Watching disk")
		watchFolders[disk.Mountpoint] = 1
	}

	log.Info().Int("count", len(watchFolders)).Msg("Watch folders")

	return watchFolders
}

func isValidDiskType(diskType string) bool {
	switch diskType {
	case "data", "cache":
		return true
	}

	return false
}

func (d *Disks) loadUnassignedDisks() {
	if !d.appConfig.UnassignedDevices {
		log.Info().Msg("Unassigned devices monitoring is disabled")

		return
	}

	// Read the current state from /var/state/unassigned.devices/unassigned.devices.json
	unassignedDevicesFile := "/var/state/unassigned.devices/unassigned.devices.json"

	data, err := os.ReadFile(unassignedDevicesFile)
	if err != nil {
		log.Warn().Err(err).Msg("Error reading unassigned devices file")

		return
	}
	// Parse the JSON data
	var unassignedDevices map[string]UDInfo

	err = json.Unmarshal(data, &unassignedDevices)
	if err != nil {
		log.Warn().Err(err).Msg("Error parsing unassigned devices JSON")

		return
	}
	// Iterate through the devices and filter based on type
	for name, device := range unassignedDevices {
		log.Info().
			Str("name", name).
			Str("mountpoint", device.Mountpoint).
			Bool("mounted", device.Mounted).
			Msg("Unassigned device details")

		if device.Mounted && device.Mountpoint != "" {
			newDisk := Disk{
				Name:       name,
				Type:       "unassigned",
				Filesystem: device.Fstype,
				Rotational: true,
				Mountpoint: device.Mountpoint,
			}
			d.unassignedDisks = append(d.unassignedDisks, newDisk)
			log.Info().Str("disk", newDisk.Name).Msg("Added unassigned disk")
		} else {
			log.Info().Str("disk", name).Msg("Skipping unassigned disk as it is not mounted or has no mountpoint")
		}
	}
}

func (d *Disks) loadDisks() {
	disks, err := ini.Load("/var/local/emhttp/disks.ini")
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading disks file")
	}

	for _, section := range disks.Sections() {
		mountpoint := "/mnt/" + section.Key("name").MustString("")
		newDisk := Disk{
			Name:       section.Key("name").MustString(""),
			Type:       strings.ToLower(section.Key("type").MustString("")),
			Filesystem: section.Key("fsType").MustString(""),
			Rotational: section.Key("rotational").MustBool(false),
			Mountpoint: mountpoint,
		}
		log.Debug().
			Str("disk", newDisk.Name).
			Str("type", newDisk.Type).
			Str("filesystem", newDisk.Filesystem).
			Bool("rotational", newDisk.Rotational).
			Msg("Found disk")

		if !isValidDiskType(newDisk.Type) {
			log.Debug().
				Str("disk", newDisk.Name).
				Str("type", newDisk.Type).
				Msg("Skipping invalid disk type")

			continue
		}

		if !newDisk.Rotational && !d.appConfig.SSD {
			log.Debug().Str("disk", newDisk.Name).Msg("Skipping SSD")

			continue
		}

		if newDisk.Type == "data" {
			d.arrayDisks = append(d.arrayDisks, newDisk)
			log.Debug().Str("disk", newDisk.Name).Msg("Added to array disks")
		}

		if newDisk.Type == "cache" && d.appConfig.Cache && newDisk.Filesystem != "" {
			d.poolDisks = append(d.poolDisks, newDisk)
			log.Debug().Str("disk", newDisk.Name).Msg("Added to pool disks")
		}
	}

	log.Info().
		Int("array_disks", len(d.arrayDisks)).
		Int("pool_disks", len(d.poolDisks)).
		Msg("Disk count")
}
