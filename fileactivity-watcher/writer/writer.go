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
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/rs/zerolog/log"
)

type Writer struct {
	mu             sync.Mutex
	currentLines   int
	maxRecords     int
	activityPath   string
	activityFile   *os.File
	activityWriter *csv.Writer
}

func New(path string, maxRecords int) *Writer {
	activityFile, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		log.Fatal().Err(err).Msg("Error opening activity file")
	}

	reader := csv.NewReader(activityFile)
	reader.FieldsPerRecord = -1

	lines, err := reader.ReadAll()
	if err != nil {
		log.Fatal().Err(err).Msg("Error reading activity file")
	}

	currentLines := len(lines)
	log.Info().Int("current_lines", currentLines).Msg("Current activity records")

	activityWriter := csv.NewWriter(activityFile)

	return &Writer{
		currentLines:   currentLines,
		activityPath:   path,
		activityFile:   activityFile,
		activityWriter: activityWriter,
		maxRecords:     maxRecords,
	}
}

func (w *Writer) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.activityWriter != nil {
		w.activityWriter.Flush()

		err := w.activityWriter.Error()
		if err != nil {
			log.Warn().Err(err).Msg("Error flushing activity writer")
		}
	}

	if w.activityFile != nil {
		err := w.activityFile.Close()
		if err != nil {
			log.Warn().Err(err).Msg("Error closing activity file")
		}
	}
}

func (w *Writer) Write(record []string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.activityWriter == nil {
		return errors.New("activity writer is not initialized")
	}

	err := w.activityWriter.Write(record)
	if err != nil {
		return fmt.Errorf("error writing activity record: %w", err)
	}

	w.activityWriter.Flush()

	err = w.activityWriter.Error()
	if err != nil {
		return fmt.Errorf("error flushing activity writer: %w", err)
	}

	w.currentLines++
	if w.currentLines > w.maxRecords {
		err := w.rolloverActivityFile()
		if err != nil {
			log.Error().Err(err).Msg("Error rolling over activity file")
		}
	}

	return nil
}

func (w *Writer) rolloverActivityFile() error {
	err := w.activityFile.Close()
	if err != nil {
		return fmt.Errorf("error closing activity file: %w", err)
	}

	rolloverPath := w.activityPath + ".1"

	_, err = os.Stat(rolloverPath)
	if err == nil {
		log.Info().Str("rollover_path", rolloverPath).Msg("Removing existing rollover file")

		err = os.Remove(rolloverPath)
		if err != nil {
			return fmt.Errorf("error removing existing rollover file: %w", err)
		}
	}

	err = os.Rename(w.activityPath, rolloverPath)
	if err != nil {
		return fmt.Errorf("error renaming activity file: %w", err)
	}

	// Reopen the activity file for writing
	w.activityFile, err = os.OpenFile(
		w.activityPath,
		os.O_APPEND|os.O_CREATE|os.O_RDWR,
		0o644,
	)
	if err != nil {
		return fmt.Errorf("error reopening activity file: %w", err)
	}

	w.activityWriter = csv.NewWriter(w.activityFile)
	// Reset the current lines count
	w.currentLines = 0

	log.Info().Msg("Activity file rolled over")

	return nil
}
