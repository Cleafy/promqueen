package model

import (
	"io"
	"sync"

	"github.com/sirupsen/logrus"
)

var mutex = &sync.Mutex{}

// WriteFrame writes the frame with the given uri to the WriteSeeker
func WriteFrame(w io.Writer, frame *Frame) error {
	mutex.Lock()
	defer mutex.Unlock()

	// err := binary.Write(w, binary.BigEndian, frame.Header)
	// if err != nil {
	// 	return err
	// }
	// err = binary.Write(w, binary.BigEndian, frame.Data)
	// if err != nil {
	// 	return err
	// }
	// logrus.Debugf("Written data to WriteSeeker: Data %s", frame.Data)
	logrus.Info("PIPPO")
	return nil
}
