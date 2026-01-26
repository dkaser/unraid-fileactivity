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
	"time"

	"github.com/dkaser/unraid-fileactivity/fileactivity-watcher/internal/types"
	"github.com/rs/zerolog/log"
)

func (f *Filter) startEventDedupeCleanup() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			f.cleanupStaleEvents()
		}
	}()
}

func (f *Filter) isDuplicateEvent(event types.Event) bool {
	now := time.Now()

	// Check if we've seen this event recently (read lock)
	f.recentEventsMutex.RLock()

	if timestamp, exists := f.recentEvents[event]; exists {
		if now.Sub(timestamp.lastSeen) < f.dedupeWindow {
			f.recentEventsMutex.RUnlock()
			log.Debug().
				Str("file", event.File).
				Str("op", event.Op).
				Int("pid", event.PID).
				Msg("Filtered duplicate event")

			return true
		}
	}

	f.recentEventsMutex.RUnlock()

	// Not a duplicate, record this event (write lock)
	f.recentEventsMutex.Lock()
	defer f.recentEventsMutex.Unlock()

	// Double-check after acquiring write lock
	if timestamp, exists := f.recentEvents[event]; exists {
		if now.Sub(timestamp.lastSeen) < f.dedupeWindow {
			return true
		}

		timestamp.lastSeen = now
	} else {
		f.recentEvents[event] = &EventTimestamp{lastSeen: now}
	}

	return false
}

func (f *Filter) cleanupStaleEvents() {
	now := time.Now()

	f.recentEventsMutex.Lock()
	defer f.recentEventsMutex.Unlock()

	for key, timestamp := range f.recentEvents {
		if now.Sub(timestamp.lastSeen) > f.dedupeWindow*2 {
			delete(f.recentEvents, key)
		}
	}
}
