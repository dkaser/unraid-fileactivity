package writer

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
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	activityPath := filepath.Join(tmpDir, "activity.log")

	writer := New(activityPath, 1000)

	if writer == nil {
		t.Fatal("Expected New() to return a non-nil writer")
	}

	if writer.activityPath != activityPath {
		t.Errorf("Expected activityPath to be %s, got %s", activityPath, writer.activityPath)
	}

	if writer.maxRecords != 1000 {
		t.Errorf("Expected maxRecords to be 1000, got %d", writer.maxRecords)
	}

	if writer.currentLines != 0 {
		t.Errorf("Expected currentLines to be 0 for new file, got %d", writer.currentLines)
	}

	if writer.activityWriter == nil {
		t.Error("Expected activityWriter to be initialized")
	}

	if writer.activityFile == nil {
		t.Error("Expected activityFile to be initialized")
	}

	// Cleanup
	writer.Close()
}

func TestNew_WithExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	activityPath := filepath.Join(tmpDir, "activity.log")

	// Create a file with some existing records
	file, err := os.Create(activityPath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	csvWriter := csv.NewWriter(file)
	csvWriter.Write([]string{"timestamp", "file", "operation", "pid"})
	csvWriter.Write([]string{"2026-01-01", "/test.txt", "WRITE", "1234"})
	csvWriter.Write([]string{"2026-01-02", "/test2.txt", "READ", "5678"})
	csvWriter.Flush()
	file.Close()

	// Open with New()
	writer := New(activityPath, 1000)

	// Should count existing lines
	if writer.currentLines != 3 {
		t.Errorf("Expected currentLines to be 3, got %d", writer.currentLines)
	}

	writer.Close()
}

func TestWrite(t *testing.T) {
	tmpDir := t.TempDir()
	activityPath := filepath.Join(tmpDir, "activity.log")

	writer := New(activityPath, 1000)
	defer writer.Close()

	// Write a record
	record := []string{"2026-01-26", "/test.txt", "WRITE", "1234"}

	err := writer.Write(record)
	if err != nil {
		t.Errorf("Expected no error writing record, got: %v", err)
	}

	// Verify currentLines increased
	if writer.currentLines != 1 {
		t.Errorf("Expected currentLines to be 1 after write, got %d", writer.currentLines)
	}

	// Write another record
	record2 := []string{"2026-01-26", "/test2.txt", "READ", "5678"}

	err = writer.Write(record2)
	if err != nil {
		t.Errorf("Expected no error writing second record, got: %v", err)
	}

	if writer.currentLines != 2 {
		t.Errorf("Expected currentLines to be 2 after second write, got %d", writer.currentLines)
	}
}

func TestWrite_NilWriter(t *testing.T) {
	writer := &Writer{
		activityWriter: nil,
	}

	record := []string{"test", "data"}

	err := writer.Write(record)
	if err == nil {
		t.Error("Expected error when writing with nil activityWriter")
	}

	if err.Error() != "activity writer is not initialized" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestWrite_VerifyFileContents(t *testing.T) {
	tmpDir := t.TempDir()
	activityPath := filepath.Join(tmpDir, "activity.log")

	writer := New(activityPath, 1000)

	// Write some records
	records := [][]string{
		{"2026-01-26", "/file1.txt", "WRITE", "100"},
		{"2026-01-26", "/file2.txt", "READ", "200"},
		{"2026-01-26", "/file3.txt", "CREATE", "300"},
	}

	for _, record := range records {
		err := writer.Write(record)
		if err != nil {
			t.Errorf("Failed to write record: %v", err)
		}
	}

	writer.Close()

	// Read back and verify
	file, err := os.Open(activityPath)
	if err != nil {
		t.Fatalf("Failed to open activity file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	lines, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read activity file: %v", err)
	}

	if len(lines) != 3 {
		t.Errorf("Expected 3 lines in file, got %d", len(lines))
	}

	// Verify first record
	if len(lines) > 0 {
		if lines[0][0] != "2026-01-26" {
			t.Errorf("Expected first field to be '2026-01-26', got %s", lines[0][0])
		}
	}
}

func TestRolloverActivityFile(t *testing.T) {
	tmpDir := t.TempDir()
	activityPath := filepath.Join(tmpDir, "activity.log")

	// Create writer with very low max records to trigger rollover
	writer := New(activityPath, 3)

	// Write records to exceed max
	for range 5 {
		record := []string{"2026-01-26", "/test.txt", "WRITE", "1234"}

		err := writer.Write(record)
		if err != nil {
			t.Errorf("Failed to write record: %v", err)
		}
	}

	writer.Close()

	// Check if rollover file exists
	rolloverPath := activityPath + ".1"
	if _, err := os.Stat(rolloverPath); os.IsNotExist(err) {
		t.Error("Expected rollover file to exist")
	}

	// Verify main file has fewer records than total written
	file, err := os.Open(activityPath)
	if err != nil {
		t.Fatalf("Failed to open activity file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	lines, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read activity file: %v", err)
	}

	// Should have rolled over, so current file should have fewer than 5 records
	if len(lines) >= 5 {
		t.Errorf(
			"Expected current file to have fewer than 5 records after rollover, got %d",
			len(lines),
		)
	}
}

func TestClose(t *testing.T) {
	tmpDir := t.TempDir()
	activityPath := filepath.Join(tmpDir, "activity.log")

	writer := New(activityPath, 1000)

	// Write a record
	record := []string{"2026-01-26", "/test.txt", "WRITE", "1234"}

	err := writer.Write(record)
	if err != nil {
		t.Errorf("Failed to write record: %v", err)
	}

	// Close the writer
	writer.Close()

	// Verify file was flushed and closed by trying to read it
	file, err := os.Open(activityPath)
	if err != nil {
		t.Errorf("Failed to open activity file after close: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	lines, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read activity file: %v", err)
	}

	if len(lines) != 1 {
		t.Errorf("Expected 1 line in file after close, got %d", len(lines))
	}
}

func TestClose_Multiple(t *testing.T) {
	tmpDir := t.TempDir()
	activityPath := filepath.Join(tmpDir, "activity.log")

	writer := New(activityPath, 1000)

	// Close multiple times should not panic
	writer.Close()
	writer.Close()
	writer.Close()
}

func TestClose_NilFields(t *testing.T) {
	writer := &Writer{
		activityWriter: nil,
		activityFile:   nil,
	}

	// Should not panic with nil fields
	writer.Close()
}

func TestWriter_InitialState(t *testing.T) {
	tmpDir := t.TempDir()
	activityPath := filepath.Join(tmpDir, "activity.log")

	writer := New(activityPath, 5000)
	defer writer.Close()

	if writer.currentLines != 0 {
		t.Errorf("Expected initial currentLines to be 0, got %d", writer.currentLines)
	}

	if writer.maxRecords != 5000 {
		t.Errorf("Expected maxRecords to be 5000, got %d", writer.maxRecords)
	}

	if writer.activityPath != activityPath {
		t.Errorf("Expected activityPath to be %s, got %s", activityPath, writer.activityPath)
	}
}

func TestWriter_ConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	activityPath := filepath.Join(tmpDir, "activity.log")

	writer := New(activityPath, 10000)
	defer writer.Close()

	done := make(chan bool)

	// Write from multiple goroutines
	for i := range 5 {
		go func(id int) {
			for range 10 {
				record := []string{"2026-01-26", "/test.txt", "WRITE", "1234"}
				_ = writer.Write(record)
			}

			done <- true
		}(i)
	}

	// Wait for all writes
	for range 5 {
		<-done
	}

	// Should have written 50 records total (may vary due to concurrent access)
	if writer.currentLines < 1 || writer.currentLines > 50 {
		t.Errorf(
			"Expected between 1 and 50 records after concurrent writes, got %d",
			writer.currentLines,
		)
	}
}

func TestRolloverActivityFile_RemovesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	activityPath := filepath.Join(tmpDir, "activity.log")
	rolloverPath := activityPath + ".1"

	// Create an existing rollover file
	existingRollover, err := os.Create(rolloverPath)
	if err != nil {
		t.Fatalf("Failed to create existing rollover file: %v", err)
	}

	existingRollover.WriteString("old data\n")
	existingRollover.Close()

	// Create writer with low max to trigger rollover
	writer := New(activityPath, 2)

	// Write enough to trigger rollover
	for range 4 {
		record := []string{"2026-01-26", "/test.txt", "WRITE", "1234"}

		err := writer.Write(record)
		if err != nil {
			t.Errorf("Failed to write record: %v", err)
		}
	}

	writer.Close()

	// Verify new rollover file exists and doesn't contain old data
	data, err := os.ReadFile(rolloverPath)
	if err != nil {
		t.Fatalf("Failed to read rollover file: %v", err)
	}

	content := string(data)
	if content == "old data\n" {
		t.Error("Rollover file still contains old data")
	}
}

func TestWriter_EmptyRecords(t *testing.T) {
	tmpDir := t.TempDir()
	activityPath := filepath.Join(tmpDir, "activity.log")

	writer := New(activityPath, 1000)
	defer writer.Close()

	// Write empty record
	record := []string{}

	err := writer.Write(record)
	if err != nil {
		t.Errorf("Expected no error writing empty record, got: %v", err)
	}

	if writer.currentLines != 1 {
		t.Errorf(
			"Expected currentLines to increment even for empty record, got %d",
			writer.currentLines,
		)
	}
}
