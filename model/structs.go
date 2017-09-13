package model

import "strings"

// Collection represents the file that contains the header and all the
// subsequent frames
type Collection struct {
	Header *Header
	Data   []Frame
}

// HeaderLength is the maximum header size
const HeaderLength = 1024

// Header represents the header of the Collection file. It contains:
//   - a Magic number (3 bytes)
//   - a Version number in order to identify incompatible differences in the binary format
//   - a FrameOffset that identifies the first Frame
//   ... a series of informational headers
type Header struct {
	Magic    [3]byte
	Version  [3]byte
	URI      [128]byte
	Reserved [890]byte
}

// Header.URIString converts the backing URI to a string
func (header *Header) URIString() string {
	return strings.TrimRight(string(header.URI[:]), "\x00")
}

// Frame represents one of the frame of the Collection file. It contains:
//  - a Size that represents how big is the the Data section
//  - a Timestamp that represents when the Frame is snapshotted
//  - the Data slice that contains the data of the frame
type Frame struct {
	Size      int64
	Timestamp int64
	Data      []byte
}
