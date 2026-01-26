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
	"strings"

	"github.com/dkaser/unraid-fileactivity/fileactivity-watcher/fanotify"
	"github.com/dkaser/unraid-fileactivity/fileactivity-watcher/internal/types"
	"golang.org/x/sys/unix"
)

type EventDetails struct {
	ContainerID string
	ProcessPath string
}

func (m *Monitor) GetEvent() (types.Event, error) {
	data, err := m.watcher.GetEvent()
	if err != nil {
		return types.Event{}, fmt.Errorf("error getting event: %w", err)
	}

	opName := getOp(data)
	pid := data.GetPID()

	var path string

	// If we have an fsid, get or open the cached mount FD
	if data.Fsid() != [8]byte{} {
		mountPath, err := m.getMountPath(data.Fsid())
		if err != nil {
			return types.Event{}, fmt.Errorf(
				"fanotify: failed to get mount path for fsid %x: %w",
				data.Fsid(),
				err,
			)
		}

		// Get or open mount FD from cache (automatically managed)
		mountFd, err := m.getOrOpenMountFD(data.Fsid(), mountPath)
		if err != nil {
			return types.Event{}, fmt.Errorf(
				"fanotify: failed to open mount FD for fsid %x: %w",
				data.Fsid(),
				err,
			)
		}

		path, err = data.GetPathWithMountFD(mountFd)
		if err != nil {
			return types.Event{}, fmt.Errorf(
				"fanotify: failed to get path with mount FD for fsid %x: %w",
				data.Fsid(),
				err,
			)
		}

		return types.Event{
			File: path,
			Op:   opName,
			PID:  pid,
		}, nil
	}

	return types.Event{}, fmt.Errorf("fanotify: failed to get event path for fsid %x", data.Fsid())
}

func (m *Monitor) GetEventDetails(event types.Event) EventDetails {
	return EventDetails{
		ContainerID: getContainerID(event.PID),
		ProcessPath: getProcessPath(event.PID),
	}
}

func getOp(data *fanotify.EventMetadata) string {
	ops := []string{}
	if data.Mask&unix.FAN_CREATE != 0 {
		ops = append(ops, "CREATE")
	}

	if data.Mask&unix.FAN_DELETE != 0 {
		ops = append(ops, "REMOVE")
	}

	if data.Mask&unix.FAN_MODIFY != 0 {
		ops = append(ops, "WRITE")
	}

	if data.Mask&unix.FAN_OPEN != 0 {
		ops = append(ops, "OPEN")
	}

	if data.Mask&unix.FAN_ACCESS != 0 {
		ops = append(ops, "READ")
	}

	if data.Mask&unix.FAN_RENAME != 0 {
		ops = append(ops, "RENAME")
	}

	if data.Mask&unix.FAN_ATTRIB != 0 {
		ops = append(ops, "CHMOD")
	}

	return strings.Join(ops, "|")
}
