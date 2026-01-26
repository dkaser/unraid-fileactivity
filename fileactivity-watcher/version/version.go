package version

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
	"runtime/debug"

	"github.com/rs/zerolog/log"
)

var Tag = "unknown" //nolint:gochecknoglobals

type BuildInfo struct {
	Tag      string
	Revision string
	GitDirty *bool
}

// GetBuildInfo returns the current build information.
func GetBuildInfo() BuildInfo {
	buildInfo := BuildInfo{
		Tag:      Tag,
		Revision: "unknown",
		GitDirty: nil,
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				buildInfo.Revision = setting.Value
			case "vcs.modified":
				val := setting.Value == "true"
				buildInfo.GitDirty = &val
			}
		}
	}

	return buildInfo
}

func BuildInfoString() string {
	info := GetBuildInfo()

	retval := fmt.Sprintf("Tag: %s\n", info.Tag)
	retval += fmt.Sprintf("Revision: %s\n", info.Revision)

	if info.GitDirty == nil {
		retval += "Git Dirty: Unknown\n"
	} else if *info.GitDirty {
		retval += "Git Dirty: Yes\n"
	}

	return retval
}

func OutputToLog() {
	info := GetBuildInfo()

	dirty := "no"

	if info.GitDirty == nil {
		dirty = "unknown"
	} else if *info.GitDirty {
		dirty = "yes"
	}

	log.Info().
		Str("tag", info.Tag).
		Str("revision", info.Revision).
		Str("git_dirty", dirty).
		Msg("Build information")
}
