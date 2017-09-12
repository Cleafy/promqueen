package model

import (
	"bytes"
	"encoding/binary"
	"io"
	"unsafe"

	"github.com/sirupsen/logrus"
)

// NewFrameReader returns a channel of Frames. The channel is closed whenever
// there are no other frames or the FrameReader encounter an error reading a frame
func NewMultiReader(r []io.Reader) <-chan Frame {
	chframe := make(chan Frame)
	if len(r) == 0 {
		close(chframe)
		return chframe
	}

	header, _ := ReadHeader(r[0])

	if !CheckVersion(header) {
		close(chframe)
		return chframe
	}

	go func() {
		defer close(chframe)
		windex := 0
		for windex < len(r) {
			frame, err := ReadFrame(r[windex])
			if err != nil {
				windex++
				if windex >= len(r) {
					break
				}
				ReadHeader(r[windex])
				continue
			}
			chframe <- *frame
		}
		logrus.Infof("Frames ended")
	}()

	return chframe
}

// ReadAll reads all the Collection (Header, Frame*) and returns in a compound
// structure.
// NOTE: the NewFrameReader streaming implementation should be preferred
func ReadAll(r io.Reader) *Collection {
	header, _ := ReadHeader(r)
	frames := make([]Frame, 0)

	for {
		frame, err := ReadFrame(r)
		if err != nil {
			break
		}
		frames = append(frames, *frame)
	}

	return &Collection{
		Header: header,
		Data:   frames,
	}
}

func readNextBytes(reader io.Reader, number int64) ([]byte, error) {
	bytes := make([]byte, number)

	_, err := reader.Read(bytes)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// ReadFrame reads the next frame from the Reader or returns an error in
// case it cannot interpret the Frame
func ReadFrame(r io.Reader) (*Frame, error) {
	frame := NewEmptyFrame()

	// read the frame Size
	data, err := readNextBytes(r, int64(unsafe.Sizeof(frame.Size)))
	if err != nil {
		return nil, err
	}
	buffer := bytes.NewBuffer(data)

	err = binary.Read(buffer, binary.BigEndian, &frame.Size)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("ReadFrame: frame.Size %d", frame.Size)

	// read the frame Timestamp
	data, err = readNextBytes(r, int64(unsafe.Sizeof(frame.Timestamp)))
	if err != nil {
		return nil, err
	}
	buffer = bytes.NewBuffer(data)

	err = binary.Read(buffer, binary.BigEndian, &frame.Timestamp)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("ReadFrame: frame.Timestamp %d", frame.Timestamp)

	// generate the correct framesize for .Data
	frame.Data = make([]byte, frame.Size)

	// read the frame Data
	data, err = readNextBytes(r, int64(len(frame.Data)))
	if err != nil {
		return nil, err
	}
	buffer = bytes.NewBuffer(data)

	err = binary.Read(buffer, binary.BigEndian, &frame.Data)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("ReadFrame: frame.Data %d", frame.Data)

	return frame, nil
}

// ReadHeader reads the header from the r (io.Reader) and returns the header
// or an error associated with the Read
func ReadHeader(r io.Reader) (*Header, error) {
	header := NewEmptyHeader()

	data, err := readNextBytes(r, int64(unsafe.Sizeof(*header)))
	if err != nil {
		return nil, err
	}
	buffer := bytes.NewBuffer(data)

	err = binary.Read(buffer, binary.BigEndian, header)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("Header %+v", header)

	return header, nil
}
