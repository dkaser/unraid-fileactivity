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
	"strings"
	"testing"
)

// Testing objectives:
// - Verify GetBuildInfo returns valid BuildInfo structure
// - Test BuildInfoString produces non-empty output
// - Test BuildInfoString contains expected fields
// - Test OutputToDebug doesn't panic
// - Test Tag variable can be set and retrieved
// - Test handling of true/false GitDirty values

func TestGetBuildInfo_ReturnsValidStruct(t *testing.T) {
	info := GetBuildInfo()

	if info.Tag == "" {
		t.Error("BuildInfo.Tag should not be empty")
	}

	if info.Revision == "" {
		t.Error("BuildInfo.Revision should not be empty")
	}

	// GitDirty can be nil, true, or false - all valid
}

func TestGetBuildInfo_UsesTagVariable(t *testing.T) {
	// Save original value
	originalTag := Tag

	defer func() { Tag = originalTag }()

	// Set custom tag
	Tag = "v1.2.3-test"

	info := GetBuildInfo()

	if info.Tag != "v1.2.3-test" {
		t.Errorf("Expected Tag to be 'v1.2.3-test', got '%s'", info.Tag)
	}
}

func TestBuildInfoString_NotEmpty(t *testing.T) {
	result := BuildInfoString()

	if result == "" {
		t.Error("BuildInfoString should return non-empty string")
	}
}

func TestBuildInfoString_ContainsExpectedFields(t *testing.T) {
	result := BuildInfoString()

	expectedFields := []string{
		"Tag:",
		"Revision:",
	}

	for _, field := range expectedFields {
		if !strings.Contains(result, field) {
			t.Errorf("BuildInfoString should contain '%s', got: %s", field, result)
		}
	}
}

func TestBuildInfoString_WithCustomTag(t *testing.T) {
	// Save original value
	originalTag := Tag

	defer func() { Tag = originalTag }()

	// Set custom tag
	Tag = "v2.0.0-custom"

	result := BuildInfoString()

	if !strings.Contains(result, "v2.0.0-custom") {
		t.Errorf("BuildInfoString should contain custom tag, got: %s", result)
	}
}

func TestOutputToLog_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("OutputToLog panicked: %v", r)
		}
	}()

	OutputToLog()
}

func TestOutputToLog_WithDifferentTags(t *testing.T) {
	// Save original value
	originalTag := Tag

	defer func() { Tag = originalTag }()

	testTags := []string{
		"v1.0.0",
		"v2.5.3-beta",
		"development",
		"",
	}

	for _, tag := range testTags {
		t.Run(tag, func(t *testing.T) {
			Tag = tag

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("OutputToDebug panicked with tag '%s': %v", tag, r)
				}
			}()

			OutputToLog()
		})
	}
}

func TestBuildInfoString_FormatsCorrectly(t *testing.T) {
	// Save original value
	originalTag := Tag

	defer func() { Tag = originalTag }()

	Tag = "test-tag"

	result := BuildInfoString()

	// Should have newlines for proper formatting
	if !strings.Contains(result, "\n") {
		t.Error("BuildInfoString should contain newlines for formatting")
	}

	// Should end with newline
	if !strings.HasSuffix(result, "\n") {
		t.Error("BuildInfoString should end with newline")
	}
}

func TestGetBuildInfo_Consistency(t *testing.T) {
	// Multiple calls should return consistent Tag values
	info1 := GetBuildInfo()
	info2 := GetBuildInfo()

	if info1.Tag != info2.Tag {
		t.Error("GetBuildInfo should return consistent Tag values")
	}

	// Note: Revision might vary if running in different contexts,
	// but Tag should be consistent within same execution
}

func TestBuildInfo_RevisionField(t *testing.T) {
	info := GetBuildInfo()

	// Revision should be set to something, even if "unknown"
	if info.Revision == "" {
		t.Error("Revision should not be empty")
	}

	// In test environment, it's likely "unknown" unless built with proper vcs info
	// Just verify it's not causing issues
}
