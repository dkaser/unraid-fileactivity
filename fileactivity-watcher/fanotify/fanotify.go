// Package fanotify package provides a simple fanotify API.
package fanotify

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/rs/zerolog/log"
	"golang.org/x/sys/unix"
)

// Procfs constants.
const (
	ProcFsFd     = "/proc/self/fd"
	ProcFsFdInfo = "/proc/self/fdinfo"
)

// FdInfo describes '/proc/PID/fdinfo/%d'.
type FdInfo struct {
	Position int
	Flags    int // octal
	MountID  int
}

// EventMetadata is a struct returned from 'NotifyFD.GetEvent'.
type EventMetadata struct {
	unix.FanotifyEventMetadata

	// Additional fields for FAN_REPORT_DFID_NAME
	filename   string
	fileHandle []byte  // Raw file handle data for open_by_handle_at
	handleType int32   // Handle type
	fsid       [8]byte // Filesystem ID
}

// GetPID return PID from event metadata.
func (metadata *EventMetadata) GetPID() int {
	return int(metadata.Pid)
}

// Fsid returns the filesystem ID.
func (metadata *EventMetadata) Fsid() [8]byte {
	return metadata.fsid
}

// Close is used to Close event Fd, use it to prevent Fd leak.
// With FAN_REPORT_DFID_NAME, Fd may be -1 (FAN_NOFD) which doesn't need closing.
func (metadata *EventMetadata) Close() error {
	// FAN_NOFD = -1, skip closing invalid descriptors
	if metadata.Fd < 0 {
		return nil
	}

	err := unix.Close(int(metadata.Fd))
	if err != nil {
		return fmt.Errorf("fanotify: failed to close Fd: %w, fd=%d", err, metadata.Fd)
	}

	return nil
}

func (metadata *EventMetadata) GetPathWithMountFD(mountFd int) (string, error) {
	// FAN_REPORT_DFID_NAME mode: Need to use open_by_handle_at
	if len(metadata.fileHandle) == 0 {
		// No file handle available
		if metadata.filename != "" {
			return metadata.filename, nil
		}

		return "", errors.New("fanotify: no file descriptor and no file handle available")
	}

	// If we have a mount FD, try to resolve the directory path
	if mountFd >= 0 {
		dirPath, err := metadata.openByHandle(mountFd)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to resolve directory path from file handle")
			// Fallback to just the filename
			if metadata.filename != "" {
				return metadata.filename, nil
			}

			return "", err
		}

		if metadata.filename != "" {
			return filepath.Join(dirPath, metadata.filename), nil
		}

		return dirPath, nil
	}

	// No mount FD available, return just the filename
	if metadata.filename != "" {
		return metadata.filename, nil
	}

	return "", errors.New(
		"fanotify: cannot resolve path without mount FD in FAN_REPORT_DFID_NAME mode",
	)
}

// MatchMask returns 'true' when event metadata matches specified mask.
func (metadata *EventMetadata) MatchMask(mask uint64) bool {
	return (metadata.Mask & mask) == mask
}

// NotifyFD is a notify file handle, used by all fanotify functions.
type NotifyFD struct {
	Fd   int
	File *os.File
	Rd   io.Reader
}

// Initialize initializes the fanotify support.
func Initialize(fanotifyFlags uint, openFlags uint) (*NotifyFD, error) {
	notifyFd, err := unix.FanotifyInit(fanotifyFlags, openFlags)
	if err != nil {
		return nil, fmt.Errorf("fanotify: init error, %w", err)
	}

	file := os.NewFile(uintptr(notifyFd), "")
	rd := bufio.NewReader(file)

	return &NotifyFD{
		Fd:   notifyFd,
		File: file,
		Rd:   rd,
	}, nil
}

// Mark implements Add/Delete/Modify for a fanotify mark.
func (handle *NotifyFD) Mark(flags uint, mask uint64, dirFd int, path string) error {
	err := unix.FanotifyMark(handle.Fd, flags, mask, dirFd, path)
	if err != nil {
		return fmt.Errorf("fanotify: mark error, %w", err)
	}

	return nil
}

// GetEvent returns an event from the fanotify handle.
func (handle *NotifyFD) GetEvent(skipPIDs ...int) (*EventMetadata, error) {
	event := new(EventMetadata)

	err := binary.Read(handle.Rd, binary.LittleEndian, &event.FanotifyEventMetadata)
	if err != nil {
		return nil, fmt.Errorf("fanotify: event error, %w", err)
	}

	if event.Vers != unix.FANOTIFY_METADATA_VERSION {
		err := event.Close()
		if err != nil {
			return nil, err
		}

		return nil, errors.New("fanotify: wrong metadata version")
	}

	// Read additional info for FAN_REPORT_DFID_NAME
	// The event structure includes variable-length info records after the metadata
	if int(event.Event_len) > binary.Size(event.FanotifyEventMetadata) {
		// Calculate how many bytes of additional data to read
		additionalBytes := int(event.Event_len) - binary.Size(event.FanotifyEventMetadata)
		extraData := make([]byte, additionalBytes)

		_, err := io.ReadFull(handle.Rd, extraData)
		if err != nil {
			return nil, fmt.Errorf("fanotify: error reading additional event data, %w", err)
		}

		// Parse the info records to extract filename
		err = event.parseInfoRecords(extraData)
		if err != nil {
			return nil, fmt.Errorf("fanotify: error parsing info records, %w", err)
		}
	}

	if slices.Contains(skipPIDs, int(event.Pid)) {
		return nil, event.Close()
	}

	return event, nil
}

// openByHandle uses open_by_handle_at to resolve a file handle to a path.
func (metadata *EventMetadata) openByHandle(mountFd int) (string, error) {
	if len(metadata.fileHandle) < 8 {
		return "", fmt.Errorf("invalid file handle length: %d", len(metadata.fileHandle))
	}

	// Extract handle_bytes from the stored handle
	handleBytes := binary.LittleEndian.Uint32(metadata.fileHandle[0:4])

	// Prepare the file_handle structure for the syscall
	// We need to pass: handle_bytes, handle_type, and the handle data
	handle := unix.NewFileHandle(
		metadata.handleType,
		metadata.fileHandle[8:8+handleBytes], // Skip the 8-byte header
	)

	// Open the file handle to get a file descriptor
	fileFd, err := unix.OpenByHandleAt(mountFd, handle, unix.O_RDONLY)
	if err != nil {
		return "", fmt.Errorf("open_by_handle_at failed: %w", err)
	}
	defer unix.Close(fileFd)

	// Read the path from /proc/self/fd
	path, err := os.Readlink(filepath.Join(ProcFsFd, strconv.Itoa(fileFd)))
	if err != nil {
		return "", fmt.Errorf("failed to readlink fd %d: %w", fileFd, err)
	}

	return path, nil
}

func (metadata *EventMetadata) parseInfoRecords(data []byte) error {
	offset := 0

	for offset < len(data) {
		record, nextOffset, err := parseInfoRecord(data, offset)
		if err != nil {
			return err
		}

		if record != nil {
			err := metadata.processInfoRecord(record)
			if err != nil {
				return err
			}
		}

		if nextOffset <= offset {
			break
		}

		offset = nextOffset
	}

	return nil
}

type infoRecord struct {
	infoType   byte
	recordData []byte
}

func parseInfoRecord(data []byte, offset int) (*infoRecord, int, error) {
	if offset+4 > len(data) {
		// Not enough data for a complete info header
		return nil, len(data), nil
	}

	infoType := data[offset]
	infoLen := binary.LittleEndian.Uint16(data[offset+2 : offset+4])

	if offset+int(infoLen) > len(data) {
		return nil, offset, fmt.Errorf(
			"invalid info record length: %d at offset %d (total data: %d)",
			infoLen,
			offset,
			len(data),
		)
	}

	// Data starts after the 4-byte header
	recordData := data[offset+4 : offset+int(infoLen)]

	// Calculate next offset with alignment
	alignedLen := int(infoLen)
	if alignedLen%4 != 0 {
		alignedLen += 4 - (alignedLen % 4)
	}

	return &infoRecord{
		infoType:   infoType,
		recordData: recordData,
	}, offset + alignedLen, nil
}

func (metadata *EventMetadata) processInfoRecord(record *infoRecord) error {
	switch record.infoType {
	case unix.FAN_EVENT_INFO_TYPE_DFID_NAME, unix.FAN_EVENT_INFO_TYPE_NEW_DFID_NAME:
		return metadata.processDfidNameRecord(record.recordData)
	}

	return nil
}

func (metadata *EventMetadata) processDfidNameRecord(recordData []byte) error {
	if len(recordData) < 8 {
		return errors.New("insufficient data for fsid in DFID_NAME record")
	}

	// Store fsid
	copy(metadata.fsid[:], recordData[0:8])

	// Parse file handle (after fsid)
	fileHandleData := recordData[8:]

	err := metadata.parseFileHandle(fileHandleData)
	if err != nil {
		return err
	}

	return nil
}

func (metadata *EventMetadata) parseFileHandle(fileHandleData []byte) error {
	if len(fileHandleData) < 8 {
		return errors.New("insufficient data for file handle in DFID_NAME record")
	}

	handleBytes := binary.LittleEndian.Uint32(fileHandleData[0:4])
	handleType := binary.LittleEndian.Uint32(fileHandleData[4:8])

	// Validate and store handle type
	if handleType > 0x7FFFFFFF {
		return fmt.Errorf("handle type %d exceeds int32 range", handleType)
	}

	metadata.handleType = int32(handleType)

	// Store complete file handle data
	handleSize := 8 + int(handleBytes)
	if handleSize > len(fileHandleData) {
		return fmt.Errorf(
			"handle size %d exceeds file handle data length %d",
			handleSize,
			len(fileHandleData),
		)
	}

	metadata.fileHandle = make([]byte, handleSize)
	copy(metadata.fileHandle, fileHandleData[0:handleSize])

	// Extract filename if present
	if handleSize < len(fileHandleData) {
		metadata.extractFilename(fileHandleData[handleSize:])
	}

	return nil
}

func (metadata *EventMetadata) extractFilename(filenameData []byte) {
	nullPos := bytes.IndexByte(filenameData, 0)
	if nullPos >= 0 {
		metadata.filename = string(filenameData[:nullPos])
	} else {
		log.Warn().Msg("No null terminator found in filename data")
	}
}
