package model

import (
	"time"
)

var (
	magic = [3]byte{0x83, 0xF1, 0xF1}
	// this should be bumped every time the format is not compatible anymore
	version = [3]byte{0x00, 0x00, 0x00}
)

// CheckVersion verifies that the binary format is compatible with the current release
func CheckVersion(header *FrameHeader) bool {
	return header.Magic == magic && header.Version == version
}

// NewFrame generates a new Frame from a given byte data
func NewFrame(name string, uri string, data []byte) *Frame {
	var u, n [52]byte
	copy(u[:], uri)
	copy(n[:], name)

	return &Frame{
		Header: &FrameHeader{
			Magic:     magic,
			Version:   version,
			Size:      int64(len(data)),
			Timestamp: time.Now().Unix(),
			Name:      n,
			URI:       u,
		},
		Data: data,
	}
}

// NewEmptyFrame generates a new empty frame
func NewEmptyFrame() *Frame {
	return NewFrame("", "", nil)
}
