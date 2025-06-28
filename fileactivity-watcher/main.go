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
	"github.com/jar-o/limlog"
	"github.com/sirupsen/logrus"
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

var log *limlog.Limlog

func init() {
	logWriter, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		logrus.SetOutput(logWriter)
	} else {
		logrus.Info("Failed to log to file, using default stderr")
	}

	cfg, err := ini.Load("/boot/config/plugins/file.activity/file.activity.cfg")

	log = limlog.NewLimlogrus()
	log.SetLimiter("monitor", 1, 600*time.Second, 1)

	if err != nil {
		log.Error("Error loading config file", logrus.Fields{"error": err})
		os.Exit(1)
	}

	includeCache = cfg.Section("").Key("INCLUDE_CACHE").MustBool(false)
	includeUD = cfg.Section("").Key("INCLUDE_UD").MustBool(false)
	includeSSD = cfg.Section("").Key("INCLUDE_SSD").MustBool(false)

	log.Info("Configuration loaded", logrus.Fields{"INCLUDE_CACHE": includeCache, "INCLUDE_UD": includeUD, "INCLUDE_SSD": includeSSD})
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
		log.Error("Error reading unassigned devices file", logrus.Fields{"error": err})
		return
	}
	// Parse the JSON data
	var unassignedDevices map[string]UDInfo
	if err := json.Unmarshal(data, &unassignedDevices); err != nil {
		log.Error("Error parsing unassigned devices JSON", logrus.Fields{"error": err})
		return
	}
	// Iterate through the devices and filter based on type
	for name, device := range unassignedDevices {
		log.Info("Found unassigned device", logrus.Fields{"name": name, "mounted": device.Mounted, "mountpoint": device.Mountpoint})
		if device.Mounted && device.Mountpoint != "" {
			newDisk := Disk{Name: name, Type: "unassigned", Filesystem: device.Fstype, Rotational: true, Mountpoint: device.Mountpoint}
			unassignedDisks = append(unassignedDisks, newDisk)
			log.Info("Added unassigned disk", logrus.Fields{"disk": newDisk.Name})
		} else {
			log.Info("Skipping unassigned disk as it is not mounted or has no mountpoint", logrus.Fields{"disk": name})
		}
	}
}

func initExclusionFilters() {
	exclusions := []string{`(?i)appdata`, `(?i)docker`, `(?i)system`, `(?i)syslogs`}
	exclusionFilters = make([]*regexp.Regexp, 0, len(exclusions))
	for _, filter := range exclusions {
		filter = strings.TrimSpace(filter)
		log.Info("Compiling exclusion filter", logrus.Fields{"filter": filter})
		exclusionFilters = append(exclusionFilters, regexp.MustCompile(filter))
	}
}

func loadDisks() {
	disks, err := ini.Load("/var/local/emhttp/disks.ini")
	if err != nil {
		log.Fatal("Error loading disks file", logrus.Fields{"error": err})
	}
	for _, section := range disks.Sections() {
		mountpoint := "/mnt/" + section.Key("name").MustString("")
		newDisk := Disk{Name: section.Key("name").MustString(""), Type: strings.ToLower(section.Key("type").MustString("")), Filesystem: section.Key("fsType").MustString(""), Rotational: section.Key("rotational").MustBool(false), Mountpoint: mountpoint}
		log.Debug("Found disk", logrus.Fields{"disk": newDisk.Name, "type": newDisk.Type, "filesystem": newDisk.Filesystem, "rotational": newDisk.Rotational})
		if !IsValidDiskType(newDisk.Type) {
			log.Debug("Skipping invalid disk type", logrus.Fields{"disk": newDisk.Name, "type": newDisk.Type})
			continue
		}
		if !newDisk.Rotational && !includeSSD {
			log.Debug("Skipping SSD", logrus.Fields{"disk": newDisk.Name})
			continue
		}
		if newDisk.Type == "data" {
			arrayDisks = append(arrayDisks, newDisk)
			log.Debug("Added to array disks", logrus.Fields{"disk": newDisk.Name})
		}
		if newDisk.Type == "cache" && includeCache && newDisk.Filesystem != "" {
			poolDisks = append(poolDisks, newDisk)
			log.Debug("Added to pool disks", logrus.Fields{"disk": newDisk.Name})
		}
	}
	log.Info("Disk count", logrus.Fields{"array_disks": len(arrayDisks), "pool_disks": len(poolDisks)})
}

func getWatchFolders() {
	watchDisks = append(arrayDisks, poolDisks...)
	watchDisks = append(watchDisks, unassignedDisks...)

	for _, disk := range watchDisks {
		log.Info("Watching disk", logrus.Fields{"disk": disk.Name, "mountpoint": disk.Mountpoint, "type": disk.Type, "filesystem": disk.Filesystem, "rotational": disk.Rotational})
		addPath = false // Reset addPath for each disk
		err := filepath.WalkDir(disk.Mountpoint, walk)
		if err != nil {
			log.Error("Error walking directory for disk", logrus.Fields{"disk": disk.Name, "error": err})
		}
	}
	log.Info("Watch folders", logrus.Fields{"count": len(watchFolders)})
}

func setInotifyLimit() {
	currentNotifyLimit, err := sysctl.Sysctl("fs/inotify/max_user_watches")
	if err != nil {
		log.Fatal("Error getting current inotify watch limit", logrus.Fields{"error": err})
		os.Exit(1)
	}
	log.Info("Current inotify watch limit", logrus.Fields{"current_limit": currentNotifyLimit})
	currentNotifyWatches := 0
	procs, err := os.ReadDir("/proc/")
	if err != nil {
		log.Error("Error reading /proc/", logrus.Fields{"error": err})
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
					log.Debug("Error reading link", logrus.Fields{"link_path": entry.Name(), "error": err})
					continue
				}
				if strings.Contains(target, "anon_inode:inotify") {
					log.Debug("Found inotify watch", logrus.Fields{"source": fdPath + entry.Name(), "target": target})
					fdinfoPath := "/proc/" + proc.Name() + "/fdinfo/" + entry.Name()
					fdinfo, err := os.ReadFile(fdinfoPath)
					if err != nil {
						log.Debug("Error reading fdinfo", logrus.Fields{"fdinfo_path": fdinfoPath, "error": err})
						continue
					}
					lines := strings.Split(string(fdinfo), "\n")
					inotifyLines := 0
					for _, line := range lines {
						if strings.HasPrefix(line, "inotify") {
							inotifyLines++
						}
					}
					log.Debug("Inotify lines", logrus.Fields{"path": fdinfoPath, "lines": inotifyLines})
					currentNotifyWatches += inotifyLines
				}
			}
		}
	}
	log.Info("Active inotify watches", logrus.Fields{"current_watches": currentNotifyWatches})
	wantedNotifyLimit := int(float64(len(watchFolders)+currentNotifyWatches) * 1.1)
	log.Info("Required inotify watch limit", logrus.Fields{"required_limit": wantedNotifyLimit})
	currentNotifyLimitInt, err := strconv.Atoi(currentNotifyLimit)
	if err != nil {
		log.Fatal("Error converting current inotify watch limit to int", logrus.Fields{"error": err})
		os.Exit(1)
	}
	if wantedNotifyLimit > currentNotifyLimitInt {
		_, err = sysctl.Sysctl("fs/inotify/max_user_watches", strconv.Itoa(wantedNotifyLimit))
		if err != nil {
			log.Fatal("Error setting inotify watch limit", logrus.Fields{"error": err})
			os.Exit(1)
		} else {
			log.Info("Inotify watch limit increased", logrus.Fields{"new_limit": wantedNotifyLimit})
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
			log.Fatal("Error opening activity file", logrus.Fields{"error": err})
			os.Exit(1)
		}
		defer activityFile.Close()
		lines, err := csv.NewReader(activityFile).ReadAll()
		if err != nil {
			log.Fatal("Error reading activity file: %v", logrus.Fields{"error": err})
			os.Exit(1)
		}
		currentLines = len(lines)
		log.Info("Current activity records", logrus.Fields{"current_lines": currentLines})
		activityWriter := csv.NewWriter(activityFile)
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
						log.Info("Removing existing rollover file", logrus.Fields{"rollover_path": rolloverPath})
						if err := os.Remove(rolloverPath); err != nil {
							log.Error("Error removing existing rollover file", logrus.Fields{"error": err})
						}
					}
					if err := os.Rename(activityPath, rolloverPath); err != nil {
						log.Error("Error renaming activity file", logrus.Fields{"error": err})
					}
					// Reopen the activity file for writing
					activityFile, err = os.OpenFile(activityPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
					if err != nil {
						log.Fatal("Error reopening activity file", logrus.Fields{"error": err})
						os.Exit(1)
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
					// Event overflow occurred, we should increase sysctl fs.notify.max_queued_events
					currentEventLimit, err := sysctl.Sysctl("fs/inotify/max_queued_events")
					if err != nil {
						log.ErrorL("monitor", "Error getting current fsnotify event limit", logrus.Fields{"error": err})
						continue
					}
					currentEventLimitInt, err := strconv.Atoi(currentEventLimit)
					if err != nil {
						log.ErrorL("monitor", "Error converting current fsnotify event limit to int", logrus.Fields{"error": err})
						continue
					}

					if currentEventLimitInt > 300000 {
						log.InfoL("monitor", "Current fsnotify event limit is high, not increasing", logrus.Fields{"current_limit": currentEventLimitInt})
						continue
					}
					wantedEventLimit := currentEventLimitInt * 2 // Increase by 1000, adjust as needed
					_, err = sysctl.Sysctl("fs/inotify/max_queued_events", strconv.Itoa(wantedEventLimit))
					if err != nil {
						log.ErrorL("monitor", "Error setting fsnotify event limit", logrus.Fields{"error": err})
					} else {
						log.InfoL("monitor", "Inotify event limit increased", logrus.Fields{"new_limit": wantedEventLimit})
					}
				} else {
					log.ErrorL("monitor", "Watcher error", logrus.Fields{"error": err})
				}
			}
		}
	}()
}

func addFoldersToWatcher(watcher *fsnotify.Watcher) {
	for folder := range watchFolders {
		err := watcher.AddWith(folder, fsnotify.WithOps(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename|fsnotify.Chmod|fsnotify.UnportableOpen))
		if err != nil {
			log.Error("Error adding folder to watcher", logrus.Fields{"folder": folder, "error": err})
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
		log.Debug("Skipping symlink", logrus.Fields{"link": s})
		return nil
	}
	// addPath is false on the first run of the loop so we don't monitor that
	// /mnt/diskX gets constant OPEN activity which floods the log
	if d.IsDir() && addPath {
		// Skip directories that match exclusion filters
		for _, filter := range exclusionFilters {
			if filter.MatchString(s) {
				log.Debug("Skipping excluded directory", logrus.Fields{"directory": s})
				return nil
			}
		}
		watchFolders[s] = 1
	}
	addPath = true
	return nil
}
