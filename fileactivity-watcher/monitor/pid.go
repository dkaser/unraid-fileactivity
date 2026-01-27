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
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// getProcessPath returns the full path to the executable for the given PID.
// It reads the /proc/<pid>/exe symlink to get the executable path.
func getProcessPath(pid int) string {
	exePath := fmt.Sprintf("/proc/%d/exe", pid)

	target, err := os.Readlink(exePath)
	if err != nil {
		return ""
	}

	return target
}

// getContainerName returns the name of the container given its PID.
// It checks if the process is running inside a Docker container by inspecting cgroup information.
func getContainerID(pid int) string {
	cgroupPath := fmt.Sprintf("/proc/%d/cgroup", pid)

	data, err := os.ReadFile(cgroupPath)
	if err != nil {
		return ""
	}

	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) == 3 && strings.Contains(parts[2], "docker") {
			return filepath.Base(parts[2])
		}
	}

	return ""
}
