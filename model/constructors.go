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
func CheckVersion(header *Header) bool {
	return header.Magic == magic && header.Version == version
}

// NewHeader generates a new Header from a given port
func NewHeader(uri string) *Header {
	var u [128]byte
	copy(u[:], uri)

	header := &Header{
		Magic:   magic,
		Version: version,
		URI:     u,
	}

	return header
}

// NewEmptyHeader generates a new empty header
func NewEmptyHeader() *Header {
	return NewHeader("")
}

// NewFrame generates a new Frame from a given byte data
func NewFrame(data []byte) *Frame {
	return &Frame{
		Size:      int64(len(data)),
		Timestamp: uint64(time.Now().Unix()),
		Data:      data,
	}
}

// NewEmptyFrame generates a new empty frame
func NewEmptyFrame() *Frame {
	return NewFrame(nil)
}
