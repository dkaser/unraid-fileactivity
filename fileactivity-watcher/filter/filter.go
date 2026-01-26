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
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dkaser/unraid-fileactivity/fileactivity-watcher/config"
	"github.com/dkaser/unraid-fileactivity/fileactivity-watcher/internal/types"
	"github.com/rs/zerolog/log"
)

type Filter struct {
	Exclusions []*regexp.Regexp

	recentEvents      map[types.Event]*EventTimestamp
	recentEventsMutex sync.RWMutex
	dedupeWindow      time.Duration
}

func New(appConfig config.ActivityConfig) *Filter {
	exclusionFilters := make([]*regexp.Regexp, 0, len(appConfig.Exclusions))
	for _, filter := range appConfig.Exclusions {
		filter = strings.TrimSpace(filter)
		log.Info().Str("filter", filter).Msg("Adding exclusion filter")
		exclusionFilters = append(exclusionFilters, regexp.MustCompile(filter))
	}

	filter := &Filter{
		Exclusions:   exclusionFilters,
		recentEvents: make(map[types.Event]*EventTimestamp),
		dedupeWindow: time.Duration(appConfig.DedupeWindow) * time.Second,
	}

	filter.startEventDedupeCleanup()

	return filter
}

func (f *Filter) IsExcluded(event types.Event) bool {
	return f.matchesExclusionFilter(event.File) || f.isDuplicateEvent(event)
}

func (f *Filter) matchesExclusionFilter(path string) bool {
	for _, filter := range f.Exclusions {
		if filter.MatchString(path) {
			return true
		}
	}

	return false
}
