package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"io/fs"
	"path/filepath"

	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	log "github.com/sirupsen/logrus"
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

var logPath = "/var/log/fileactivity-watcher.log"
var activityPath = "/var/log/file.activity/data.log"
var maxRecords = 20000

var includeCache bool
var includeUD bool
var includeSSD bool

var addPath = false

func init() {
	logWriter, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(logWriter)
	} else {
		log.Info("Failed to log to file, using default stderr")
	}

	cfg, err := ini.Load("/boot/config/plugins/file.activity/file.activity.cfg")

	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error loading config file")
		os.Exit(1)
	}

	includeCache = cfg.Section("").Key("INCLUDE_CACHE").MustBool(false)
	includeUD = cfg.Section("").Key("INCLUDE_UD").MustBool(false)
	includeSSD = cfg.Section("").Key("INCLUDE_SSD").MustBool(false)

	log.WithFields(log.Fields{"INCLUDE_CACHE": includeCache, "INCLUDE_UD": includeUD, "INCLUDE_SSD": includeSSD}).Info("Configuration loaded")
}

func main() {
	log.Info("Starting file activity watcher...")

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

	log.Info("Watcher ready")
	<-make(chan struct{})
}

func loadUnassignedDisks() {
	if !includeUD {
		log.Info("Unassigned devices monitoring is disabled")
		return
	}

	// Read the current state from /var/state/unassigned.devices/unassigned.devices.json
	unassignedDevicesFile := "/var/state/unassigned.devices/unassigned.devices.json"
	data, err := os.ReadFile(unassignedDevicesFile)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error reading unassigned devices file")
		return
	}
	// Parse the JSON data
	var unassignedDevices map[string]UDInfo
	if err := json.Unmarshal(data, &unassignedDevices); err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error parsing unassigned devices JSON")
		return
	}
	// Iterate through the devices and filter based on type
	for name, device := range unassignedDevices {
		log.WithFields(log.Fields{"name": name, "mounted": device.Mounted, "mountpoint": device.Mountpoint}).Info("Found unassigned device")
		if device.Mounted && device.Mountpoint != "" {
			newDisk := Disk{Name: name, Type: "unassigned", Filesystem: device.Fstype, Rotational: true, Mountpoint: device.Mountpoint}
			unassignedDisks = append(unassignedDisks, newDisk)
			log.WithFields(log.Fields{"disk": newDisk.Name}).Info("Added unassigned disk")
		} else {
			log.WithFields(log.Fields{"disk": name}).Info("Skipping unassigned disk as it is not mounted or has no mountpoint")
		}
	}
}

func initExclusionFilters() {
	exclusions := []string{`(?i)appdata`, `(?i)docker`, `(?i)system`, `(?i)syslogs`}
	exclusionFilters = make([]*regexp.Regexp, 0, len(exclusions))
	for _, filter := range exclusions {
		filter = strings.TrimSpace(filter)
		log.WithFields(log.Fields{"filter": filter}).Info("Compiling exclusion filter")
		exclusionFilters = append(exclusionFilters, regexp.MustCompile(filter))
	}
}

func loadDisks() {
	disks, err := ini.Load("/var/local/emhttp/disks.ini")
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Fatal("Error loading disks file")
	}
	for _, section := range disks.Sections() {
		mountpoint := "/mnt/" + section.Key("name").MustString("")
		newDisk := Disk{Name: section.Key("name").MustString(""), Type: strings.ToLower(section.Key("type").MustString("")), Filesystem: section.Key("fsType").MustString(""), Rotational: section.Key("rotational").MustBool(false), Mountpoint: mountpoint}
		log.WithFields(log.Fields{"disk": newDisk.Name, "type": newDisk.Type, "filesystem": newDisk.Filesystem, "rotational": newDisk.Rotational}).Debug("Found disk")
		if !IsValidDiskType(newDisk.Type) {
			log.WithFields(log.Fields{"disk": newDisk.Name, "type": newDisk.Type}).Debug("Skipping invalid disk type")
			continue
		}
		if !newDisk.Rotational && !includeSSD {
			log.WithFields(log.Fields{"disk": newDisk.Name}).Debug("Skipping SSD")
			continue
		}
		if newDisk.Type == "data" {
			arrayDisks = append(arrayDisks, newDisk)
			log.WithFields(log.Fields{"disk": newDisk.Name}).Debug("Added to array disks")
		}
		if newDisk.Type == "cache" && includeCache && newDisk.Filesystem != "" {
			poolDisks = append(poolDisks, newDisk)
			log.WithFields(log.Fields{"disk": newDisk.Name}).Debug("Added to pool disks")
		}
	}
	log.WithFields(log.Fields{"array_disks": len(arrayDisks), "pool_disks": len(poolDisks)}).Info("Disk count")
}

func getWatchFolders() {
	watchDisks = append(arrayDisks, poolDisks...)
	watchDisks = append(watchDisks, unassignedDisks...)

	for _, disk := range watchDisks {
		log.WithFields(log.Fields{"disk": disk.Name, "mountpoint": disk.Mountpoint, "type": disk.Type, "filesystem": disk.Filesystem, "rotational": disk.Rotational}).Info("Watching disk")
		addPath = false // Reset addPath for each disk
		err := filepath.WalkDir(disk.Mountpoint, walk)
		if err != nil {
			log.WithFields(log.Fields{"disk": disk.Name, "error": err}).Error("Error walking directory for disk")
		}
	}
	log.WithFields(log.Fields{"count": len(watchFolders)}).Info("Watch folders")
}

func setInotifyLimit() {
	currentNotifyLimit, err := sysctl.Sysctl("fs/inotify/max_user_watches")
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Fatal("Error getting current inotify watch limit")
	}
	log.WithFields(log.Fields{"current_limit": currentNotifyLimit}).Info("Current inotify watch limit")
	currentNotifyWatches := 0
	procs, err := os.ReadDir("/proc/")
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Error reading /proc/")
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
					log.WithFields(log.Fields{"link_path": entry.Name(), "error": err}).Debug("Error reading link")
					continue
				}
				if strings.Contains(target, "anon_inode:inotify") {
					log.WithFields(log.Fields{"source": fdPath + entry.Name(), "target": target}).Debug("Found inotify watch")
					fdinfoPath := "/proc/" + proc.Name() + "/fdinfo/" + entry.Name()
					fdinfo, err := os.ReadFile(fdinfoPath)
					if err != nil {
						log.WithFields(log.Fields{"fdinfo_path": fdinfoPath, "error": err}).Debug("Error reading fdinfo")
						continue
					}
					lines := strings.Split(string(fdinfo), "\n")
					inotifyLines := 0
					for _, line := range lines {
						if strings.HasPrefix(line, "inotify") {
							inotifyLines++
						}
					}
					log.WithFields(log.Fields{"path": fdinfoPath, "lines": inotifyLines}).Debug("Inotify lines")
					currentNotifyWatches += inotifyLines
				}
			}
		}
	}
	log.WithFields(log.Fields{"current_watches": currentNotifyWatches}).Info("Active inotify watches")
	wantedNotifyLimit := int(float64(len(watchFolders)+currentNotifyWatches) * 1.1)
	log.WithFields(log.Fields{"required_limit": wantedNotifyLimit}).Info("Required inotify watch limit")
	currentNotifyLimitInt, err := strconv.Atoi(currentNotifyLimit)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Fatal("Error converting current inotify watch limit to int")
	}
	if wantedNotifyLimit > currentNotifyLimitInt {
		_, err = sysctl.Sysctl("fs/inotify/max_user_watches", strconv.Itoa(wantedNotifyLimit))
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Fatal("Error setting inotify watch limit")
		} else {
			log.WithFields(log.Fields{"new_limit": wantedNotifyLimit}).Info("Inotify watch limit increased")
		}
	}
}

func buildFsnotifyWatcher() *fsnotify.Watcher {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	return watcher
}

func startEventListener(watcher *fsnotify.Watcher) {
	go func() {
		log.Info("Starting event listener...")
		currentLines := 0
		activityFile, err := os.OpenFile(activityPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Fatal("Error opening activity file")
		}
		defer activityFile.Close()
		lines, err := csv.NewReader(activityFile).ReadAll()
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Fatal("Error reading activity file")
		}
		currentLines = len(lines)
		log.WithFields(log.Fields{"current_lines": currentLines}).Info("Current activity records")
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
				if currentLines >= maxRecords {
					// Close the file, then roll it over to .1 (deleting any existing .1 file) and truncate the original
					activityFile.Close()
					rolloverPath := activityPath + ".1"
					if _, err := os.Stat(rolloverPath); err == nil {
						log.WithFields(log.Fields{"rollover_path": rolloverPath}).Info("Removing existing rollover file")
						if err := os.Remove(rolloverPath); err != nil {
							log.WithFields(log.Fields{"error": err}).Error("Error removing existing rollover file")
						}
					}
					if err := os.Rename(activityPath, rolloverPath); err != nil {
						log.WithFields(log.Fields{"error": err}).Error("Error renaming activity file")
					}
					// Reopen the activity file for writing
					activityFile, err = os.OpenFile(activityPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
					if err != nil {
						log.WithFields(log.Fields{"error": err}).Fatal("Error reopening activity file")
					}
					activityWriter = csv.NewWriter(activityFile)
					// Reset the current lines count
					currentLines = 0
					log.Info("Activity file rolled over")
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
						log.WithFields(log.Fields{"error": err}).Error("Error getting current inotify event limit")
						continue
					}
					currentEventLimitInt, err := strconv.Atoi(currentEventLimit)
					if err != nil {
						log.WithFields(log.Fields{"error": err}).Error("Error converting current inotify event limit to int")
						continue
					}

					if currentEventLimitInt > 300000 {
						log.WithFields(log.Fields{"currentLimit": currentEventLimitInt}).Info("monitor", "Current inotify event limit is high, not increasing, ignoring overflow")
						ignoreOverflow = true
						continue
					}
					wantedEventLimit := currentEventLimitInt * 2
					_, err = sysctl.Sysctl("fs/inotify/max_queued_events", strconv.Itoa(wantedEventLimit))
					if err != nil {
						log.WithFields(log.Fields{"error": err}).Error("Error setting inotify event limit")
					} else {
						log.WithFields(log.Fields{"new_limit": wantedEventLimit}).Info("inotify event limit increased")
					}
				} else {
					log.WithFields(log.Fields{"error": err}).Error("Watcher error")
				}
			}
		}
	}()
}

func addFoldersToWatcher(watcher *fsnotify.Watcher) {
	for folder := range watchFolders {
		err := watcher.AddWith(folder, fsnotify.WithOps(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename|fsnotify.Chmod|fsnotify.UnportableOpen))
		if err != nil {
			log.WithFields(log.Fields{"folder": folder, "error": err}).Error("Error adding folder to watcher")
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
		log.WithFields(log.Fields{"link": s}).Debug("Skipping symlink")
		return nil
	}
	// addPath is false on the first run of the loop so we don't monitor that
	// /mnt/diskX gets constant OPEN activity which floods the log
	if d.IsDir() && addPath {
		// Skip directories that match exclusion filters
		for _, filter := range exclusionFilters {
			if filter.MatchString(s) {
				log.WithFields(log.Fields{"directory": s}).Debug("Skipping excluded directory")
				return nil
			}
		}
		watchFolders[s] = 1
	}
	addPath = true
	return nil
}
