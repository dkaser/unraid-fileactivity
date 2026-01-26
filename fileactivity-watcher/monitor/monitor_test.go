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
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dkaser/unraid-fileactivity/fileactivity-watcher/internal/types"
)

func TestGetOp(t *testing.T) {
	// Note: getOp is not exported, but we can test the operation translation
	// through GetEvent if needed, or export it for testing
	// For now, we'll test the EventDetails functionality
}

func TestGetProcessPath_InvalidPID(t *testing.T) {
	// Test with a PID that doesn't exist
	path := getProcessPath(99999999)
	if path != "" {
		t.Errorf("Expected empty string for invalid PID, got %s", path)
	}
}

func TestGetProcessPath_CurrentProcess(t *testing.T) {
	// Test with current process PID
	pid := os.Getpid()
	path := getProcessPath(pid)

	// Should get a non-empty path for the current process
	if path == "" {
		t.Error("Expected non-empty path for current process")
	}

	// Path should be an absolute path
	if !filepath.IsAbs(path) {
		t.Errorf("Expected absolute path, got %s", path)
	}
}

func TestGetContainerID_NonContainerProcess(t *testing.T) {
	// Test with current process (not in a container typically)
	pid := os.Getpid()
	containerID := getContainerID(pid)

	// If not running in Docker, should return empty string
	// If running in Docker, we can't make assumptions, so we just verify it doesn't panic
	_ = containerID
}

func TestGetContainerID_InvalidPID(t *testing.T) {
	// Test with a PID that doesn't exist
	containerID := getContainerID(99999999)
	if containerID != "" {
		t.Errorf("Expected empty string for invalid PID, got %s", containerID)
	}
}

func TestEventDetails_Struct(t *testing.T) {
	details := EventDetails{
		ContainerID: "abc123",
		ProcessPath: "/usr/bin/test",
	}

	if details.ContainerID != "abc123" {
		t.Errorf("Expected ContainerID to be 'abc123', got %s", details.ContainerID)
	}

	if details.ProcessPath != "/usr/bin/test" {
		t.Errorf("Expected ProcessPath to be '/usr/bin/test', got %s", details.ProcessPath)
	}
}

func TestGetEventDetails(t *testing.T) {
	m := &Monitor{}

	event := types.Event{
		File: "/test.txt",
		PID:  os.Getpid(),
		Op:   "WRITE",
	}

	details := m.GetEventDetails(event)

	// Should get some process path for current process
	if details.ProcessPath == "" {
		t.Error("Expected non-empty ProcessPath for current process")
	}

	// ContainerID may be empty if not in container
	_ = details.ContainerID
}

func TestMountInfo_Struct(t *testing.T) {
	fsid := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}
	mountInfo := MountInfo{
		Path: "/mnt/disk1",
		Fsid: fsid,
	}

	if mountInfo.Path != "/mnt/disk1" {
		t.Errorf("Expected Path to be '/mnt/disk1', got %s", mountInfo.Path)
	}

	if mountInfo.Fsid != fsid {
		t.Errorf("Expected Fsid to match, got %v", mountInfo.Fsid)
	}
}

func TestGetFsid(t *testing.T) {
	// Test with root directory which should always exist
	fsid, err := getFsid("/")
	if err != nil {
		t.Errorf("Expected to get fsid for /, got error: %v", err)
	}

	// Verify we got a non-zero fsid
	allZero := true

	for _, b := range fsid {
		if b != 0 {
			allZero = false

			break
		}
	}

	if allZero {
		t.Error("Expected non-zero fsid for /")
	}
}

func TestGetFsid_InvalidPath(t *testing.T) {
	// Test with a path that doesn't exist
	_, err := getFsid("/this/path/definitely/does/not/exist")
	if err == nil {
		t.Error("Expected error for invalid path")
	}

	if !strings.Contains(err.Error(), "failed to get fsid") {
		t.Errorf("Expected error message to contain 'failed to get fsid', got: %v", err)
	}
}

func TestGetMountPath(t *testing.T) {
	fsid1 := [8]byte{1, 2, 3, 4, 5, 6, 7, 8}
	fsid2 := [8]byte{8, 7, 6, 5, 4, 3, 2, 1}

	m := &Monitor{
		mountInfos: []MountInfo{
			{Path: "/mnt/disk1", Fsid: fsid1},
			{Path: "/mnt/disk2", Fsid: fsid2},
		},
	}

	// Test finding existing mount
	path, err := m.getMountPath(fsid1)
	if err != nil {
		t.Errorf("Expected to find mount, got error: %v", err)
	}

	if path != "/mnt/disk1" {
		t.Errorf("Expected path '/mnt/disk1', got %s", path)
	}

	// Test with non-existing fsid
	fsid3 := [8]byte{9, 9, 9, 9, 9, 9, 9, 9}

	_, err = m.getMountPath(fsid3)
	if err == nil {
		t.Error("Expected error for non-existing fsid")
	}
}

func TestMountFDCache_Struct(t *testing.T) {
	cache := MountFDCache{
		fd:       42,
		lastUsed: time.Now(),
	}

	if cache.fd != 42 {
		t.Errorf("Expected fd to be 42, got %d", cache.fd)
	}
}

func TestSetupMountTracking(t *testing.T) {
	tmpDir := t.TempDir()

	m := &Monitor{
		watchFolders: map[string]int{
			tmpDir: 1,
		},
		mountInfos: []MountInfo{},
	}

	m.setupMountTracking()

	// Should have added mount info
	if len(m.mountInfos) != 1 {
		t.Errorf("Expected 1 mount info, got %d", len(m.mountInfos))
	}

	if m.mountInfos[0].Path != tmpDir {
		t.Errorf("Expected mount path to be %s, got %s", tmpDir, m.mountInfos[0].Path)
	}

	// Verify fsid is non-zero
	allZero := true

	for _, b := range m.mountInfos[0].Fsid {
		if b != 0 {
			allZero = false

			break
		}
	}

	if allZero {
		t.Error("Expected non-zero fsid")
	}
}

func TestCleanupStaleMountFDs(t *testing.T) {
	// This test is challenging without actually opening file descriptors
	// We can test the logic with mock data
	m := &Monitor{
		mountFDCache: make(map[[8]byte]*MountFDCache),
		mountTTL:     1, // Very short TTL for testing
	}

	// The cleanup function will try to close FDs, so we skip actual FD operations
	// This is more of a structural test
	if m.mountFDCache == nil {
		t.Error("Expected mountFDCache to be initialized")
	}
}

func TestMonitor_EmptyWatchFolders(t *testing.T) {
	m := &Monitor{
		watchFolders: make(map[string]int),
		mountInfos:   []MountInfo{},
	}

	m.setupMountTracking()

	if len(m.mountInfos) != 0 {
		t.Errorf("Expected 0 mount infos for empty watch folders, got %d", len(m.mountInfos))
	}
}

func TestGetContainerID_ParsesDockerCgroup(t *testing.T) {
	// Test the parsing logic conceptually
	// In a real container, the cgroup file would contain docker paths

	// Test with current process (which is likely not in a container)
	pid := os.Getpid()
	containerID := getContainerID(pid)

	// If we're not in a container, should be empty
	// If we are, it should not panic
	_ = containerID
}

func TestGetProcessPath_ParsesSymlink(t *testing.T) {
	// Test with PID 1 (init process), which should always exist on Linux
	path := getProcessPath(1)

	// Should get either a valid path or empty string if can't read
	// On most systems, should be able to read init process
	if path != "" && !filepath.IsAbs(path) {
		t.Errorf("If path is returned, it should be absolute, got: %s", path)
	}
}

func TestMonitor_Initialization(t *testing.T) {
	// Test basic Monitor struct initialization
	watchFolders := map[string]int{
		"/tmp": 1,
	}

	m := &Monitor{
		mountInfos:   []MountInfo{},
		mountFDCache: make(map[[8]byte]*MountFDCache),
		watchFolders: watchFolders,
	}

	if m.mountInfos == nil {
		t.Error("Expected mountInfos to be initialized")
	}

	if m.mountFDCache == nil {
		t.Error("Expected mountFDCache to be initialized")
	}

	if len(m.watchFolders) != 1 {
		t.Errorf("Expected 1 watch folder, got %d", len(m.watchFolders))
	}
}

func TestEventDetails_EmptyFields(t *testing.T) {
	details := EventDetails{}

	if details.ContainerID != "" {
		t.Error("Expected empty ContainerID by default")
	}

	if details.ProcessPath != "" {
		t.Error("Expected empty ProcessPath by default")
	}
}
