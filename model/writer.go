package model

import (
	"encoding/binary"
	"io"

	"github.com/sirupsen/logrus"
)

// WriteFrame writes the frame with the given uri to the WriteSeeker
func WriteFrame(w io.WriteSeeker, uri string, frame *Frame) error {
	// Seek end of the file, if it is possible
	currpos, err := w.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	logrus.Debugf("WriteFrame: currpos %d", currpos)
	// in case the file is empty we need to write the header
	if currpos == 0 {
		logrus.Debugf("The file is empty, write header with uri: %s", uri)
		header := NewHeader(uri)
		logrus.Debugf("The new header is %+v", header)
		err = binary.Write(w, binary.BigEndian, header)
		if err != nil {
			return err
		}
	}

	err = binary.Write(w, binary.BigEndian, frame.Size)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, frame.Timestamp)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, frame.Data)
	if err != nil {
		return err
	}
	logrus.Debugf("Written data to WriteSeeker: Size %d and Data %s", frame.Size, frame.Data)

	return nil
}
