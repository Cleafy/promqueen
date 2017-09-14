package model

import (
	"encoding/binary"
	"io"

	"github.com/sirupsen/logrus"
)

// WriteFrame writes the frame with the given uri to the WriteSeeker
func WriteFrame(w io.Writer, frame *Frame) error {
	err := binary.Write(w, binary.BigEndian, frame.Header)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, frame.Data)
	if err != nil {
		return err
	}
	logrus.Debugf("Written data to WriteSeeker: Data %s", frame.Data)

	return nil
}
