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
	"encoding/hex"
	"fmt"
	"time"
	"unsafe"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

type MountInfo struct {
	Path string
	Fsid [8]byte
}

type MountFDCache struct {
	fd       int
	lastUsed time.Time
}

func getFsid(path string) ([8]byte, error) {
	var stat unix.Statfs_t

	err := unix.Statfs(path, &stat)
	if err != nil {
		return [8]byte{}, fmt.Errorf("failed to get fsid for path %s: %w", path, err)
	}

	var fsid [8]byte
	// The Fsid field structure varies by architecture
	// We need to copy the raw bytes
	fsidBytes := (*[8]byte)(unsafe.Pointer(&stat.Fsid))
	copy(fsid[:], fsidBytes[:])

	return fsid, nil
}

func (m *Monitor) setupMountTracking() {
	for folder := range m.watchFolders {
		fsid, err := getFsid(folder)
		if err != nil {
			log.Error().Err(err).Str("folder", folder).Msg("Failed to get fsid")

			continue
		}

		m.mountInfos = append(m.mountInfos, MountInfo{
			Path: folder,
			Fsid: fsid,
		})

		log.Info().
			Str("path", folder).
			Str("fsid", hex.EncodeToString(fsid[:])).
			Msg("Cached mount fsid")
	}
}

func (m *Monitor) startMountFDCacheCleanup() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			m.cleanupStaleMountFDs()
		}
	}()
}

func (m *Monitor) cleanupStaleMountFDs() {
	now := time.Now()

	m.mountFDCacheMutex.Lock()
	defer m.mountFDCacheMutex.Unlock()

	for fsid, cache := range m.mountFDCache {
		if now.Sub(cache.lastUsed) > m.mountTTL {
			unix.Close(cache.fd)
			delete(m.mountFDCache, fsid)
			log.Debug().Str("fsid", hex.EncodeToString(fsid[:])).Msg("Closed stale mount FD")
		}
	}
}

func (m *Monitor) getOrOpenMountFD(fsid [8]byte, mountPath string) (int, error) {
	// Check cache first (read lock)
	m.mountFDCacheMutex.RLock()

	if cache, exists := m.mountFDCache[fsid]; exists {
		cache.lastUsed = time.Now()
		fd := cache.fd

		m.mountFDCacheMutex.RUnlock()
		log.Debug().Str("fsid", hex.EncodeToString(fsid[:])).Msg("Reusing cached mount FD")

		return fd, nil
	}

	m.mountFDCacheMutex.RUnlock()
	// Not in cache, open new FD (write lock)
	m.mountFDCacheMutex.Lock()
	defer m.mountFDCacheMutex.Unlock()

	// Double-check after acquiring write lock
	if cache, exists := m.mountFDCache[fsid]; exists {
		cache.lastUsed = time.Now()

		return cache.fd, nil
	}

	// Open mount FD
	mountFd, err := unix.Open(mountPath, unix.O_RDONLY|unix.O_DIRECTORY, 0)
	if err != nil {
		return -1, fmt.Errorf("failed to open mount FD for path %s: %w", mountPath, err)
	}

	// Cache it
	m.mountFDCache[fsid] = &MountFDCache{
		fd:       mountFd,
		lastUsed: time.Now(),
	}

	log.Debug().
		Str("fsid", hex.EncodeToString(fsid[:])).
		Int("fd", mountFd).
		Msg("Opened and cached mount FD")

	return mountFd, nil
}

func (m *Monitor) getMountPath(fsid [8]byte) (string, error) {
	for _, info := range m.mountInfos {
		if info.Fsid == fsid {
			return info.Path, nil
		}
	}

	return "", fmt.Errorf("no mount found for fsid %x", fsid)
}
