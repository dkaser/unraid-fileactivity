package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"io/fs"
	"path/filepath"

	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/ini.v1"

	"github.com/fsnotify/fsnotify"
)

type Disk struct {
	Name       string
	Mountpoint string
	Type       string
	Filesystem string
	Rotational bool
}

type UDInfo struct {
	Mountpoint string `json:"mountpoint"`
	Mounted    bool   `json:"mounted"`
	Fstype     string `json:"fstype"`
}

var arrayDisks []Disk
var poolDisks []Disk
var unassignedDisks []Disk
var watchDisks []Disk

var watchFolders map[string]int
var exclusionFilters []*regexp.Regexp

var activityPath = "/var/log/file.activity/data.log"

var addPath = false

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	debug := flag.Bool("debug", false, "sets log level to debug")

	flag.Parse()

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	loadConfig()

	if !appConfig.Enable {
		log.Info().Msg("File activity watcher is disabled, exiting")
		os.Exit(0)
	}
}

func main() {
	log.Info().Msg("Starting file activity watcher...")

	watchFolders = make(map[string]int)

	initExclusionFilters()
	loadDisks()
	loadUnassignedDisks()
	getWatchFolders()
	setInotifyLimit()
	watcher := buildFsnotifyWatcher()
	defer watcher.Close()
	startEventListener(watcher)
	addFoldersToWatcher(watcher)

	log.Info().Msg("Watcher ready")
	<-make(chan struct{})
}

func loadUnassignedDisks() {
	if !appConfig.UnassignedDevices {
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
	if err := json.Unmarshal(data, &unassignedDevices); err != nil {
		log.Warn().Err(err).Msg("Error parsing unassigned devices JSON")
		return
	}
	// Iterate through the devices and filter based on type
	for name, device := range unassignedDevices {
		log.Info().Str("name", name).Str("mountpoint", device.Mountpoint).Bool("mounted", device.Mounted).Msg("Unassigned device details")
		if device.Mounted && device.Mountpoint != "" {
			newDisk := Disk{Name: name, Type: "unassigned", Filesystem: device.Fstype, Rotational: true, Mountpoint: device.Mountpoint}
			unassignedDisks = append(unassignedDisks, newDisk)
			log.Info().Str("disk", newDisk.Name).Msg("Added unassigned disk")
		} else {
			log.Info().Str("disk", name).Msg("Skipping unassigned disk as it is not mounted or has no mountpoint")
		}
	}
}

func initExclusionFilters() {
	exclusionFilters = make([]*regexp.Regexp, 0, len(appConfig.Exclusions))
	for _, filter := range appConfig.Exclusions {
		filter = strings.TrimSpace(filter)
		log.Info().Str("filter", filter).Msg("Adding exclusion filter")
		exclusionFilters = append(exclusionFilters, regexp.MustCompile(filter))
	}
}

func loadDisks() {
	disks, err := ini.Load("/var/local/emhttp/disks.ini")
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading disks file")
	}
	for _, section := range disks.Sections() {
		mountpoint := "/mnt/" + section.Key("name").MustString("")
		newDisk := Disk{Name: section.Key("name").MustString(""), Type: strings.ToLower(section.Key("type").MustString("")), Filesystem: section.Key("fsType").MustString(""), Rotational: section.Key("rotational").MustBool(false), Mountpoint: mountpoint}
		log.Debug().Str("disk", newDisk.Name).Str("type", newDisk.Type).Str("filesystem", newDisk.Filesystem).Bool("rotational", newDisk.Rotational).Msg("Found disk")
		if !IsValidDiskType(newDisk.Type) {
			log.Debug().Str("disk", newDisk.Name).Str("type", newDisk.Type).Msg("Skipping invalid disk type")
			continue
		}
		if !newDisk.Rotational && !appConfig.SSD {
			log.Debug().Str("disk", newDisk.Name).Msg("Skipping SSD")
			continue
		}
		if newDisk.Type == "data" {
			arrayDisks = append(arrayDisks, newDisk)
			log.Debug().Str("disk", newDisk.Name).Msg("Added to array disks")
		}
		if newDisk.Type == "cache" && appConfig.Cache && newDisk.Filesystem != "" {
			poolDisks = append(poolDisks, newDisk)
			log.Debug().Str("disk", newDisk.Name).Msg("Added to pool disks")
		}
	}
	log.Info().Int("array_disks", len(arrayDisks)).Int("pool_disks", len(poolDisks)).Msg("Disk count")
}

func getWatchFolders() {
	watchDisks = append(arrayDisks, poolDisks...)
	watchDisks = append(watchDisks, unassignedDisks...)

	for _, disk := range watchDisks {
		log.Info().Str("disk", disk.Name).Str("mountpoint", disk.Mountpoint).Str("type", disk.Type).Str("filesystem", disk.Filesystem).Bool("rotational", disk.Rotational).Msg("Watching disk")
		addPath = false // Reset addPath for each disk
		err := filepath.WalkDir(disk.Mountpoint, walk)
		if err != nil {
			log.Error().Str("disk", disk.Name).Err(err).Msg("Error walking directory for disk")
		}
	}
	log.Info().Int("count", len(watchFolders)).Msg("Watch folders")
}

func setInotifyLimit() {
	currentNotifyLimit, err := sysctl.Sysctl("fs/inotify/max_user_watches")
	if err != nil {
		log.Fatal().Err(err).Msg("Error getting current inotify watch limit")
	}
	log.Info().Str("current_limit", currentNotifyLimit).Msg("Current inotify watch limit")
	currentNotifyWatches := 0
	procs, err := os.ReadDir("/proc/")
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading /proc/")
		return
	}
	for _, proc := range procs {
		if !proc.IsDir() || !regexp.MustCompile(`^\d+$`).MatchString(proc.Name()) {
			continue
		}
		fdPath := "/proc/" + proc.Name() + "/fd/"
		entries, err := os.ReadDir(fdPath)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.Type()&fs.ModeSymlink != 0 {
				target, err := os.Readlink(fdPath + entry.Name())
				if err != nil {
					log.Debug().Str("link_path", entry.Name()).Err(err).Msg("Error reading link")
					continue
				}
				if strings.Contains(target, "anon_inode:inotify") {
					log.Debug().Str("source", fdPath+entry.Name()).Str("target", target).Msg("Found inotify watch")
					fdinfoPath := "/proc/" + proc.Name() + "/fdinfo/" + entry.Name()
					fdinfo, err := os.ReadFile(fdinfoPath)
					if err != nil {
						log.Debug().Str("fdinfo_path", fdinfoPath).Err(err).Msg("Error reading fdinfo")
						continue
					}
					lines := strings.Split(string(fdinfo), "\n")
					inotifyLines := 0
					for _, line := range lines {
						if strings.HasPrefix(line, "inotify") {
							inotifyLines++
						}
					}
					log.Debug().Str("path", fdinfoPath).Int("lines", inotifyLines).Msg("Inotify lines")
					currentNotifyWatches += inotifyLines
				}
			}
		}
	}
	log.Info().Int("current_watches", currentNotifyWatches).Msg("Active inotify watches")
	wantedNotifyLimit := int(float64(len(watchFolders)+currentNotifyWatches) * 1.1)
	log.Info().Int("required_limit", wantedNotifyLimit).Msg("Required inotify watch limit")
	currentNotifyLimitInt, err := strconv.Atoi(currentNotifyLimit)
	if err != nil {
		log.Fatal().Err(err).Msg("Error converting current inotify watch limit to int")
	}
	if wantedNotifyLimit > currentNotifyLimitInt {
		_, err = sysctl.Sysctl("fs/inotify/max_user_watches", strconv.Itoa(wantedNotifyLimit))
		if err != nil {
			log.Fatal().Err(err).Msg("Error setting inotify watch limit")
		} else {
			log.Info().Int("new_limit", wantedNotifyLimit).Msg("Inotify watch limit increased")
		}
	}
}

func buildFsnotifyWatcher() *fsnotify.Watcher {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating fsnotify watcher")
	}
	return watcher
}

func startEventListener(watcher *fsnotify.Watcher) {
	go func() {
		log.Info().Msg("Starting event listener...")
		currentLines := 0
		activityFile, err := os.OpenFile(activityPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			log.Fatal().Err(err).Msg("Error opening activity file")
		}
		defer activityFile.Close()
		lines, err := csv.NewReader(activityFile).ReadAll()
		if err != nil {
			log.Fatal().Err(err).Msg("Error reading activity file")
		}
		currentLines = len(lines)
		log.Info().Int("current_lines", currentLines).Msg("Current activity records")
		activityWriter := csv.NewWriter(activityFile)
		ignoreOverflow := false
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				activityWriter.Write([]string{time.Now().Format("2006-01-02T15:04:05.000Z07:00"), event.Op.String(), event.Name})
				activityWriter.Flush()
				currentLines++
				if currentLines >= appConfig.MaxRecords {
					// Close the file, then roll it over to .1 (deleting any existing .1 file) and truncate the original
					activityFile.Close()
					rolloverPath := activityPath + ".1"
					if _, err := os.Stat(rolloverPath); err == nil {
						log.Info().Str("rollover_path", rolloverPath).Msg("Removing existing rollover file")
						if err := os.Remove(rolloverPath); err != nil {
							log.Error().Err(err).Msg("Error removing existing rollover file")
						}
					}
					if err := os.Rename(activityPath, rolloverPath); err != nil {
						log.Error().Err(err).Msg("Error renaming activity file")
					}
					// Reopen the activity file for writing
					activityFile, err = os.OpenFile(activityPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
					if err != nil {
						log.Fatal().Err(err).Msg("Error reopening activity file")
					}
					activityWriter = csv.NewWriter(activityFile)
					// Reset the current lines count
					currentLines = 0
					log.Info().Msg("Activity file rolled over")
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				if errors.Is(err, fsnotify.ErrEventOverflow) {
					if ignoreOverflow {
						continue
					}
					// Event overflow occurred, we should increase sysctl fs.notify.max_queued_events
					currentEventLimit, err := sysctl.Sysctl("fs/inotify/max_queued_events")
					if err != nil {
						log.Error().Err(err).Msg("Error getting current inotify event limit")
						continue
					}
					currentEventLimitInt, err := strconv.Atoi(currentEventLimit)
					if err != nil {
						log.Error().Err(err).Msg("Error converting current inotify event limit to int")
						continue
					}

					if currentEventLimitInt > 300000 {
						log.Info().Int("currentLimit", currentEventLimitInt).Msg("Current inotify event limit is high, not increasing, ignoring overflow")
						ignoreOverflow = true
						continue
					}
					wantedEventLimit := currentEventLimitInt * 2
					_, err = sysctl.Sysctl("fs/inotify/max_queued_events", strconv.Itoa(wantedEventLimit))
					if err != nil {
						log.Error().Err(err).Msg("Error setting inotify event limit")
					} else {
						log.Info().Int("new_limit", wantedEventLimit).Msg("inotify event limit increased")
					}
				} else {
					log.Error().Err(err).Msg("Watcher error")
				}
			}
		}
	}()
}

func addFoldersToWatcher(watcher *fsnotify.Watcher) {
	for folder := range watchFolders {
		err := watcher.AddWith(folder, fsnotify.WithOps(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename|fsnotify.Chmod|fsnotify.UnportableOpen))
		if err != nil {
			log.Error().Str("folder", folder).Err(err).Msg("Error adding folder to watcher")
			continue
		}
	}
}

func IsValidDiskType(diskType string) bool {
	switch diskType {
	case
		"data",
		"cache":
		return true
	}
	return false
}

func walk(s string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if d.Type() == fs.ModeSymlink {
		// Skip symlinks
		log.Debug().Str("link", s).Msg("Skipping symlink")
		return nil
	}
	// addPath is false on the first run of the loop so we don't monitor that
	// /mnt/diskX gets constant OPEN activity which floods the log
	if d.IsDir() && addPath {
		// Skip directories that match exclusion filters
		for _, filter := range exclusionFilters {
			if filter.MatchString(s) {
				log.Debug().Str("directory", s).Msg("Skipping excluded directory")
				return nil
			}
		}
		watchFolders[s] = 1
	}
	addPath = true
	return nil
}
