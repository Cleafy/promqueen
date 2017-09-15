package model

import "strings"

// Collection represents the file that contains all the subsequent frames
type Collection struct {
	Data []*Frame
}

// Header.URIString converts the backing URI to a string
func (frame *Frame) URIString() string {
	return strings.TrimRight(string(frame.Header.URI[:]), "\x00")
}

// Header.URIString converts the backing URI to a string
func (frame *Frame) NameString() string {
	return strings.TrimRight(string(frame.Header.Name[:]), "\x00")
}

// FrameHeaderLength total header length size for each frame
const FrameHeaderLength = 128

// FrameHeader represents the header of each Frame
//  - a Size that represents how big is the the Data section
//  - a Timestamp that represents when the Frame is snapshotted
//  - a Name that represents the service that has been snapshotted
//  - an URL that represents the service location
type FrameHeader struct {
	Magic     [3]byte
	Version   [3]byte
	Reserved  [2]byte
	Size      int64
	Timestamp int64
	Name      [52]byte
	URI       [52]byte
}

// Frame represents one of the frame of the Collection file. It contains:
//  - the Data slice that contains the data of the frame
type Frame struct {
	Header *FrameHeader
	Data   []byte
}
