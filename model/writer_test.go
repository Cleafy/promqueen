package model

import (
	"io"
	"os"
	"testing"

	"github.com/mattetti/filebuffer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// This test depends on a working reader
func TestWriteFrame(t *testing.T) {
	backing := make([]byte, 0)
	buffer := filebuffer.New(backing)
	uri := "http://ciao:8080/v1/metrics"
	name := "foobar"

	err := WriteFrame(buffer, NewFrame(name, uri, []byte("Foo1Bar")))
	assert.Empty(t, err, "error1 should be empty")
	err = WriteFrame(buffer, NewFrame(name, uri, []byte("Foo2Bar")))
	assert.Empty(t, err, "error2 should be empty")
	err = WriteFrame(buffer, NewFrame(name, uri, []byte("Foo3Bar")))
	assert.Empty(t, err, "error3 should be empty")

	// restart the
	buffer.Seek(0, io.SeekStart)
	collection := ReadAll(buffer)

	for _, frame := range collection.Data {
		assert.True(t, CheckVersion(frame))
	}
	assert.Equal(t, 3, len(collection.Data), "there should be exactly 3 frames")

	for _, frame := range collection.Data {
		assert.Equal(t, uri, frame.URIString(), "saved uri should be equal")
		assert.Equal(t, name, frame.NameString(), "saved uri should be equal")
	}
	logrus.Debugf("collection data %+v", collection.Data)
	assert.Equal(t, "Foo1Bar", string(collection.Data[0].Data), "data should be equal")
	assert.Equal(t, "Foo2Bar", string(collection.Data[1].Data), "data should be equal")
	assert.Equal(t, "Foo3Bar", string(collection.Data[2].Data), "data should be equal")
}

func init() {
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	logrus.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	// logrus.SetLevel(logrus.DebugLevel)
}
