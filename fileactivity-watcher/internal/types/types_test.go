package types

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
	"testing"
)

func TestEvent_Struct(t *testing.T) {
	event := Event{
		File: "/mnt/disk1/test.txt",
		PID:  1234,
		Op:   "WRITE",
	}

	if event.File != "/mnt/disk1/test.txt" {
		t.Errorf("Expected File to be '/mnt/disk1/test.txt', got %s", event.File)
	}

	if event.PID != 1234 {
		t.Errorf("Expected PID to be 1234, got %d", event.PID)
	}

	if event.Op != "WRITE" {
		t.Errorf("Expected Op to be 'WRITE', got %s", event.Op)
	}
}

func TestEvent_Equality(t *testing.T) {
	event1 := Event{File: "/test.txt", PID: 100, Op: "READ"}
	event2 := Event{File: "/test.txt", PID: 100, Op: "READ"}
	event3 := Event{File: "/test.txt", PID: 200, Op: "READ"}
	event4 := Event{File: "/other.txt", PID: 100, Op: "READ"}
	event5 := Event{File: "/test.txt", PID: 100, Op: "WRITE"}

	if event1 != event2 {
		t.Error("Events with identical fields should be equal")
	}

	if event1 == event3 {
		t.Error("Events with different PIDs should not be equal")
	}

	if event1 == event4 {
		t.Error("Events with different files should not be equal")
	}

	if event1 == event5 {
		t.Error("Events with different operations should not be equal")
	}
}

func TestEvent_AsMapKey(t *testing.T) {
	eventMap := make(map[Event]string)

	event1 := Event{File: "/test1.txt", PID: 100, Op: "READ"}
	event2 := Event{File: "/test2.txt", PID: 200, Op: "WRITE"}
	event3 := Event{File: "/test1.txt", PID: 100, Op: "READ"}

	eventMap[event1] = "first"
	eventMap[event2] = "second"
	eventMap[event3] = "third"

	if len(eventMap) != 2 {
		t.Errorf("Expected map to have 2 entries, got %d", len(eventMap))
	}

	if eventMap[event1] != "third" {
		t.Errorf("Expected event1 value to be 'third', got %s", eventMap[event1])
	}

	if eventMap[event2] != "second" {
		t.Errorf("Expected event2 value to be 'second', got %s", eventMap[event2])
	}
}

func TestEvent_ZeroValue(t *testing.T) {
	var event Event

	if event.File != "" {
		t.Error("Expected File to be empty string by default")
	}

	if event.PID != 0 {
		t.Error("Expected PID to be 0 by default")
	}

	if event.Op != "" {
		t.Error("Expected Op to be empty string by default")
	}
}

func TestEvent_VariousOperations(t *testing.T) {
	operations := []string{"READ", "WRITE", "CREATE", "DELETE", "MODIFY", "OPEN", "CLOSE"}

	for _, op := range operations {
		event := Event{
			File: "/test.txt",
			PID:  1000,
			Op:   op,
		}

		if event.Op != op {
			t.Errorf("Expected Op to be %s, got %s", op, event.Op)
		}
	}
}

func TestEvent_VariousPaths(t *testing.T) {
	paths := []string{
		"/mnt/disk1/media/video.mp4",
		"/mnt/cache/appdata/config.json",
		"/mnt/user/documents/file.txt",
		"/var/log/syslog.log",
		"/home/user/.config/app/settings.conf",
	}

	for _, path := range paths {
		event := Event{
			File: path,
			PID:  1234,
			Op:   "READ",
		}

		if event.File != path {
			t.Errorf("Expected File to be %s, got %s", path, event.File)
		}
	}
}

func TestEvent_PIDBoundaries(t *testing.T) {
	testPIDs := []int{0, 1, 100, 1000, 65535, 1000000}

	for _, pid := range testPIDs {
		event := Event{
			File: "/test.txt",
			PID:  pid,
			Op:   "WRITE",
		}

		if event.PID != pid {
			t.Errorf("Expected PID to be %d, got %d", pid, event.PID)
		}
	}
}

func TestEvent_EmptyFields(t *testing.T) {
	event := Event{
		File: "",
		PID:  0,
		Op:   "",
	}

	if event.File != "" {
		t.Error("Expected empty File field")
	}

	if event.PID != 0 {
		t.Error("Expected PID to be 0")
	}

	if event.Op != "" {
		t.Error("Expected empty Op field")
	}
}

func TestEvent_CopyBehavior(t *testing.T) {
	original := Event{
		File: "/original.txt",
		PID:  100,
		Op:   "READ",
	}

	copy := original

	copy.File = "/modified.txt"
	copy.PID = 200
	copy.Op = "WRITE"

	if original.File != "/original.txt" {
		t.Error("Original File was modified")
	}

	if original.PID != 100 {
		t.Error("Original PID was modified")
	}

	if original.Op != "READ" {
		t.Error("Original Op was modified")
	}

	if copy.File != "/modified.txt" {
		t.Error("Copy File was not modified")
	}

	if copy.PID != 200 {
		t.Error("Copy PID was not modified")
	}

	if copy.Op != "WRITE" {
		t.Error("Copy Op was not modified")
	}
}
