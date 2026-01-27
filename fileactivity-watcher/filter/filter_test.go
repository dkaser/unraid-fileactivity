package filter

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
	"time"

	"github.com/dkaser/unraid-fileactivity/fileactivity-watcher/config"
	"github.com/dkaser/unraid-fileactivity/fileactivity-watcher/internal/types"
)

func TestNew(t *testing.T) {
	appConfig := config.ActivityConfig{
		Exclusions:   []string{`(?i)appdata`, `(?i)docker`},
		DedupeWindow: 2,
	}

	filter := New(appConfig)

	if filter == nil {
		t.Fatal("Expected New() to return a non-nil filter")
	}

	if len(filter.Exclusions) != 2 {
		t.Errorf("Expected 2 exclusion filters, got %d", len(filter.Exclusions))
	}

	if filter.recentEvents == nil {
		t.Error("Expected recentEvents to be initialized")
	}

	if filter.dedupeWindow != 2*time.Second {
		t.Errorf("Expected dedupeWindow to be 2s, got %v", filter.dedupeWindow)
	}
}

func TestMatchesExclusionFilter(t *testing.T) {
	appConfig := config.ActivityConfig{
		Exclusions:   []string{`(?i)appdata`, `(?i)docker`, `\.log$`},
		DedupeWindow: 1,
	}

	filter := New(appConfig)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"matches appdata", "/mnt/cache/appdata/test.txt", true},
		{"matches APPDATA case insensitive", "/mnt/cache/APPDATA/test.txt", true},
		{"matches docker", "/mnt/user/docker/container.img", true},
		{"matches log extension", "/var/log/syslog.log", true},
		{"no match", "/mnt/disk1/media/video.mp4", false},
		{"no match plain text", "/mnt/disk1/documents/file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.matchesExclusionFilter(tt.path)
			if result != tt.expected {
				t.Errorf(
					"matchesExclusionFilter(%q) = %v, expected %v",
					tt.path,
					result,
					tt.expected,
				)
			}
		})
	}
}

func TestIsExcluded_ExclusionFilter(t *testing.T) {
	appConfig := config.ActivityConfig{
		Exclusions:   []string{`(?i)appdata`, `(?i)system`},
		DedupeWindow: 1,
	}

	filter := New(appConfig)

	event := types.Event{
		File: "/mnt/cache/appdata/test.txt",
		PID:  1234,
		Op:   "WRITE",
	}

	if !filter.IsExcluded(event) {
		t.Error("Expected event to be excluded by exclusion filter")
	}
}

func TestIsDuplicateEvent(t *testing.T) {
	appConfig := config.ActivityConfig{
		Exclusions:   []string{},
		DedupeWindow: 2,
	}

	filter := New(appConfig)

	event := types.Event{
		File: "/mnt/disk1/test.txt",
		PID:  1234,
		Op:   "WRITE",
	}

	// First occurrence should not be a duplicate
	if filter.isDuplicateEvent(event) {
		t.Error("First occurrence should not be a duplicate")
	}

	// Immediate second occurrence should be a duplicate
	if !filter.isDuplicateEvent(event) {
		t.Error("Second occurrence should be a duplicate")
	}

	// Different PID should not be a duplicate
	event2 := types.Event{
		File: "/mnt/disk1/test.txt",
		PID:  5678,
		Op:   "WRITE",
	}

	if filter.isDuplicateEvent(event2) {
		t.Error("Event with different PID should not be a duplicate")
	}

	// Different file should not be a duplicate
	event3 := types.Event{
		File: "/mnt/disk1/other.txt",
		PID:  1234,
		Op:   "WRITE",
	}

	if filter.isDuplicateEvent(event3) {
		t.Error("Event with different file should not be a duplicate")
	}
}

func TestIsDuplicateEvent_AfterDedupeWindow(t *testing.T) {
	appConfig := config.ActivityConfig{
		Exclusions:   []string{},
		DedupeWindow: 1, // 1 second window
	}

	filter := New(appConfig)

	event := types.Event{
		File: "/mnt/disk1/test.txt",
		PID:  1234,
		Op:   "WRITE",
	}

	// First occurrence
	if filter.isDuplicateEvent(event) {
		t.Error("First occurrence should not be a duplicate")
	}

	// Immediate second occurrence should be a duplicate
	if !filter.isDuplicateEvent(event) {
		t.Error("Second occurrence should be a duplicate within window")
	}

	// Manually manipulate the timestamp to simulate time passing
	filter.recentEventsMutex.Lock()

	if timestamp, exists := filter.recentEvents[event]; exists {
		timestamp.lastSeen = time.Now().Add(-2 * time.Second)
	}

	filter.recentEventsMutex.Unlock()

	// After dedupe window, should not be a duplicate
	if filter.isDuplicateEvent(event) {
		t.Error("Event after dedupe window should not be a duplicate")
	}
}

func TestIsExcluded_Combined(t *testing.T) {
	appConfig := config.ActivityConfig{
		Exclusions:   []string{`(?i)appdata`},
		DedupeWindow: 1,
	}

	filter := New(appConfig)

	// Event that matches exclusion filter
	event1 := types.Event{
		File: "/mnt/cache/appdata/test.txt",
		PID:  1234,
		Op:   "WRITE",
	}

	if !filter.IsExcluded(event1) {
		t.Error("Event matching exclusion filter should be excluded")
	}

	// Event that doesn't match exclusion but is a duplicate
	event2 := types.Event{
		File: "/mnt/disk1/data/test.txt",
		PID:  1234,
		Op:   "WRITE",
	}

	// First occurrence should not be excluded
	if filter.IsExcluded(event2) {
		t.Error("First occurrence of valid event should not be excluded")
	}

	// Second occurrence should be excluded as duplicate
	if !filter.IsExcluded(event2) {
		t.Error("Duplicate event should be excluded")
	}
}

func TestCleanupStaleEvents(t *testing.T) {
	appConfig := config.ActivityConfig{
		Exclusions:   []string{},
		DedupeWindow: 1,
	}

	filter := New(appConfig)

	// Add some events
	event1 := types.Event{File: "/test1.txt", PID: 1, Op: "WRITE"}
	event2 := types.Event{File: "/test2.txt", PID: 2, Op: "READ"}

	filter.isDuplicateEvent(event1)
	filter.isDuplicateEvent(event2)

	// Verify events are in cache
	filter.recentEventsMutex.RLock()
	initialCount := len(filter.recentEvents)
	filter.recentEventsMutex.RUnlock()

	if initialCount != 2 {
		t.Errorf("Expected 2 events in cache, got %d", initialCount)
	}

	// Manually set timestamps to old values
	filter.recentEventsMutex.Lock()

	for _, timestamp := range filter.recentEvents {
		timestamp.lastSeen = time.Now().Add(-10 * time.Second)
	}

	filter.recentEventsMutex.Unlock()

	// Run cleanup
	filter.cleanupStaleEvents()

	// Verify stale events were removed
	filter.recentEventsMutex.RLock()
	finalCount := len(filter.recentEvents)
	filter.recentEventsMutex.RUnlock()

	if finalCount != 0 {
		t.Errorf("Expected 0 events after cleanup, got %d", finalCount)
	}
}

func TestFilter_ConcurrentAccess(t *testing.T) {
	appConfig := config.ActivityConfig{
		Exclusions:   []string{`(?i)test`},
		DedupeWindow: 1,
	}

	filter := New(appConfig)

	done := make(chan bool)

	// Spawn multiple goroutines to test concurrent access
	for i := range 10 {
		go func(id int) {
			for j := range 100 {
				event := types.Event{
					File: "/mnt/disk1/file.txt",
					PID:  id*100 + j,
					Op:   "WRITE",
				}
				_ = filter.IsExcluded(event)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for range 10 {
		<-done
	}
}

func TestEventTimestamp(t *testing.T) {
	now := time.Now()
	timestamp := &eventTimestamp{
		lastSeen: now,
	}

	if !timestamp.lastSeen.Equal(now) {
		t.Error("EventTimestamp lastSeen not set correctly")
	}
}

func TestNew_EmptyExclusions(t *testing.T) {
	appConfig := config.ActivityConfig{
		Exclusions:   []string{},
		DedupeWindow: 1,
	}

	filter := New(appConfig)

	if len(filter.Exclusions) != 0 {
		t.Errorf("Expected 0 exclusion filters, got %d", len(filter.Exclusions))
	}

	// No paths should match exclusion filter
	testPaths := []string{
		"/mnt/disk1/test.txt",
		"/mnt/cache/appdata/test.txt",
		"/var/log/syslog.log",
	}

	for _, path := range testPaths {
		if filter.matchesExclusionFilter(path) {
			t.Errorf("Path %q should not match with empty exclusions", path)
		}
	}
}

func TestFilter_DifferentOperations(t *testing.T) {
	appConfig := config.ActivityConfig{
		Exclusions:   []string{},
		DedupeWindow: 1,
	}

	filter := New(appConfig)

	// Same file and PID but different operations
	event1 := types.Event{File: "/test.txt", PID: 1234, Op: "WRITE"}
	event2 := types.Event{File: "/test.txt", PID: 1234, Op: "READ"}

	// First write
	if filter.isDuplicateEvent(event1) {
		t.Error("First WRITE should not be a duplicate")
	}

	// First read (different operation, so different event)
	if filter.isDuplicateEvent(event2) {
		t.Error("First READ should not be a duplicate")
	}

	// Second write (should be duplicate)
	if !filter.isDuplicateEvent(event1) {
		t.Error("Second WRITE should be a duplicate")
	}
}
