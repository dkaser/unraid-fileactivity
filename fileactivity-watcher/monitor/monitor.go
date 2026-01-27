package monitor

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
	"os"
	"sync"
	"time"

	"github.com/dkaser/unraid-fileactivity/fileactivity-watcher/fanotify"
	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

type Monitor struct {
	mountInfos        []MountInfo
	mountFDCache      map[[8]byte]*MountFDCache
	mountFDCacheMutex sync.RWMutex
	mountTTL          time.Duration
	watchFolders      map[string]int
	watcher           *fanotify.NotifyFD
}

func New(watchFolders map[string]int) *Monitor {
	monitor := &Monitor{
		mountInfos:        []MountInfo{},
		mountFDCache:      make(map[[8]byte]*MountFDCache),
		mountFDCacheMutex: sync.RWMutex{},
		mountTTL:          10 * time.Second,
		watchFolders:      watchFolders,
	}

	monitor.setupMountTracking()
	monitor.startMountFDCacheCleanup()
	monitor.buildFanotifyWatcher()
	monitor.addFoldersToWatcher()

	return monitor
}

func (m *Monitor) buildFanotifyWatcher() {
	watcher, err := fanotify.Initialize(
		unix.FAN_CLOEXEC|
			unix.FAN_CLASS_NOTIF|
			unix.FAN_UNLIMITED_QUEUE|
			unix.FAN_UNLIMITED_MARKS|
			unix.FAN_REPORT_DFID_NAME,
		uint(os.O_RDONLY|
			unix.O_LARGEFILE|
			unix.O_CLOEXEC),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating fanotify watcher")
	}

	m.watcher = watcher
}

func (m *Monitor) addFoldersToWatcher() {
	for folder := range m.watchFolders {
		// err := watcher.AddWith(folder, fsnotify.WithOps(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename|fsnotify.Chmod|fsnotify.UnportableOpen))
		err := m.watcher.Mark(
			unix.FAN_MARK_ADD|
				unix.FAN_MARK_FILESYSTEM,
			unix.FAN_CREATE|unix.FAN_RENAME|unix.FAN_MODIFY|unix.FAN_DELETE|unix.FAN_ACCESS|unix.FAN_ATTRIB|unix.FAN_OPEN,
			unix.AT_FDCWD,
			folder,
		)
		if err != nil {
			log.Error().Str("folder", folder).Err(err).Msg("Error adding folder to watcher")

			continue
		}
	}
}
