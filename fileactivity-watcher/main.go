package main

import (
	"encoding/csv"
	"encoding/json"
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
		log.Fatalf("Error loading config file: %v", err)
	}

	includeCache = cfg.Section("").Key("INCLUDE_CACHE").MustBool(false)
	includeUD = cfg.Section("").Key("INCLUDE_UD").MustBool(false)
	includeSSD = cfg.Section("").Key("INCLUDE_SSD").MustBool(false)

	log.Infof("Configuration loaded: INCLUDE_CACHE=%v, INCLUDE_UD=%v, INCLUDE_SSD=%v", includeCache, includeUD, includeSSD)
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
		log.Errorf("Error reading unassigned devices file: %v", err)
		return
	}
	// Parse the JSON data
	var unassignedDevices map[string]UDInfo
	if err := json.Unmarshal(data, &unassignedDevices); err != nil {
		log.Errorf("Error parsing unassigned devices JSON: %v", err)
		return
	}
	// Iterate through the devices and filter based on type
	for name, device := range unassignedDevices {
		log.Infof("Found unassigned device: %s mounted: %v mountpoint: %s", name, device.Mounted, device.Mountpoint)
		if device.Mounted && device.Mountpoint != "" {
			newDisk := Disk{Name: name, Type: "unassigned", Filesystem: device.Fstype, Rotational: true, Mountpoint: device.Mountpoint}
			unassignedDisks = append(unassignedDisks, newDisk)
			log.Infof("Added unassigned disk: %s", newDisk.Name)
		} else {
			log.Infof("Skipping unassigned disk %s as it is not mounted or has no mountpoint", name)
		}
	}
}

func initExclusionFilters() {
	exclusions := []string{`(?i)appdata`, `(?i)docker`, `(?i)system`, `(?i)syslogs`}
	exclusionFilters = make([]*regexp.Regexp, 0, len(exclusions))
	for _, filter := range exclusions {
		filter = strings.TrimSpace(filter)
		log.Infof("Compiling exclusion filter: %s", filter)
		exclusionFilters = append(exclusionFilters, regexp.MustCompile(filter))
	}
}

func loadDisks() {
	disks, err := ini.Load("/var/local/emhttp/disks.ini")
	if err != nil {
		log.Fatalf("Error loading disks file: %v", err)
	}
	for _, section := range disks.Sections() {
		mountpoint := "/mnt/" + section.Key("name").MustString("")
		newDisk := Disk{Name: section.Key("name").MustString(""), Type: strings.ToLower(section.Key("type").MustString("")), Filesystem: section.Key("fsType").MustString(""), Rotational: section.Key("rotational").MustBool(false), Mountpoint: mountpoint}
		log.Debugf("Found disk: %s, Type: %s, Filesystem: %s, Rotational: %v", newDisk.Name, newDisk.Type, newDisk.Filesystem, newDisk.Rotational)
		if !IsValidDiskType(newDisk.Type) {
			log.Debugf("Skipping invalid disk type: %s", newDisk.Type)
			continue
		}
		if !newDisk.Rotational && !includeSSD {
			log.Debugf("Skipping SSD: %s", newDisk.Name)
			continue
		}
		if newDisk.Type == "data" {
			arrayDisks = append(arrayDisks, newDisk)
			log.Debugf("Added to array disks: %s", newDisk.Name)
		}
		if newDisk.Type == "cache" && includeCache && newDisk.Filesystem != "" {
			poolDisks = append(poolDisks, newDisk)
			log.Debugf("Added to pool disks: %s", newDisk.Name)
		}
	}
	log.Infof("Array Disks: %d, Pool Disks: %d", len(arrayDisks), len(poolDisks))
}

func getWatchFolders() {
	watchDisks = append(arrayDisks, poolDisks...)
	watchDisks = append(watchDisks, unassignedDisks...)

	for _, disk := range watchDisks {
		log.Infof("Watching disk: %s, Type: %s, Filesystem: %s, Rotational: %v, Mountpoint: %s", disk.Name, disk.Type, disk.Filesystem, disk.Rotational, disk.Mountpoint)
		addPath = false // Reset addPath for each disk
		err := filepath.WalkDir(disk.Mountpoint, walk)
		if err != nil {
			log.Errorf("Error walking directory for disk %s: %v", disk.Name, err)
		}
	}
	log.Infof("Watch folders: %d", len(watchFolders))
}

func setInotifyLimit() {
	currentNotifyLimit, err := sysctl.Sysctl("fs/inotify/max_user_watches")
	if err != nil {
		log.Fatalf("Error getting current inotify watch limit: %v", err)
	}
	log.Infof("Current inotify watch limit: %s", currentNotifyLimit)
	currentNotifyWatches := 0
	procs, err := os.ReadDir("/proc/")
	if err != nil {
		log.Errorf("Error reading /proc/: %v", err)
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
					log.Debugf("Error reading link for %s: %v", entry.Name(), err)
					continue
				}
				if strings.Contains(target, "anon_inode:inotify") {
					log.Debugf("Found inotify watch: %s -> %s", fdPath+entry.Name(), target)
					fdinfoPath := "/proc/" + proc.Name() + "/fdinfo/" + entry.Name()
					fdinfo, err := os.ReadFile(fdinfoPath)
					if err != nil {
						log.Debugf("Error reading fdinfo for %s: %v", fdinfoPath, err)
						continue
					}
					lines := strings.Split(string(fdinfo), "\n")
					inotifyLines := 0
					for _, line := range lines {
						if strings.HasPrefix(line, "inotify") {
							inotifyLines++
						}
					}
					log.Debugf("Inotify lines for %s: %d", fdinfoPath, inotifyLines)
					currentNotifyWatches += inotifyLines
				}
			}
		}
	}
	log.Infof("Active inotify watches: %d", currentNotifyWatches)
	wantedNotifyLimit := int(float64(len(watchFolders)+currentNotifyWatches) * 1.1)
	log.Infof("Required inotify watch limit: %d", wantedNotifyLimit)
	currentNotifyLimitInt, err := strconv.Atoi(currentNotifyLimit)
	if err != nil {
		log.Fatalf("Error converting current inotify watch limit to int: %v", err)
	}
	if wantedNotifyLimit > currentNotifyLimitInt {
		sysctl.Sysctl("fs/inotify/max_user_watches", strconv.Itoa(wantedNotifyLimit))
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
			log.Fatalf("Error opening activity file: %v", err)
		}
		defer activityFile.Close()
		lines, err := csv.NewReader(activityFile).ReadAll()
		if err != nil {
			log.Fatalf("Error reading activity file: %v", err)
		}
		currentLines = len(lines)
		log.Infof("Current activity records: %d", currentLines)
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
						log.Infof("Removing existing rollover file: %s", rolloverPath)
						if err := os.Remove(rolloverPath); err != nil {
							log.Errorf("Error removing existing rollover file: %v", err)
						}
					}
					if err := os.Rename(activityPath, rolloverPath); err != nil {
						log.Errorf("Error renaming activity file: %v", err)
					}
					// Reopen the activity file for writing
					activityFile, err = os.OpenFile(activityPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
					if err != nil {
						log.Fatalf("Error reopening activity file: %v", err)
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
				log.Println("error:", err)
			}
		}
	}()
}

func addFoldersToWatcher(watcher *fsnotify.Watcher) {
	for folder := range watchFolders {
		err := watcher.AddWith(folder, fsnotify.WithOps(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename|fsnotify.Chmod|fsnotify.UnportableOpen))
		if err != nil {
			log.Errorf("Error adding folder to watcher %s: %v", folder, err)
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
		log.Debugf("Skipping symlink: %s", s)
		return nil
	}
	// addPath is false on the first run of the loop so we don't monitor that
	// /mnt/diskX gets constant OPEN activity which floods the log
	if d.IsDir() && addPath {
		// Skip directories that match exclusion filters
		for _, filter := range exclusionFilters {
			if filter.MatchString(s) {
				log.Debugf("Skipping excluded directory: %s", s)
				return nil
			}
		}
		watchFolders[s] = 1
	}
	addPath = true
	return nil
}
