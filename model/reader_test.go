package model

import (
	"io"
	"os"
	"testing"

	"github.com/mattetti/filebuffer"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var data = []byte{
	0x42, 0x42, 0x42, 0x42,
}

var datalen byte = byte(len(data))

var frameSample = []byte{
	0x83, 0xF1, 0xF1, // Magic
	0x00, 0x00, 0x00, // Version
	0x00, 0x00, // Reserved
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, datalen, // Size
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Timestamp
} // len(frameSample) == 20

func init() {
	f := make([]byte, FrameHeaderLength)
	copy(f, frameSample)
	frameSample = f
	frameSample = append(frameSample, data...)
}

func TestReadFrame(t *testing.T) {
	buffer := filebuffer.New(frameSample)
	_, err := ReadFrame(buffer)
	assert.Empty(t, err, "should not be any error")
}

func TestRead2Frames(t *testing.T) {
	tmp := append(frameSample, frameSample...)

	assert.Equal(t,
		len(frameSample)*2,
		len(tmp),
		"tmp length should be the sum of all the lenghts")

	buffer := filebuffer.New(tmp)

	_, err := ReadFrame(buffer)
	assert.Empty(t, err, "should not be any error")

	_, err = ReadFrame(buffer)
	assert.Empty(t, err, "should not be any error")
}

func TestReadAll(t *testing.T) {
	tmp := append(frameSample, frameSample...)

	assert.Equal(t,
		len(frameSample)*2,
		len(tmp),
		"tmp length should be the sum of all the lenghts")

	buffer := filebuffer.New(tmp)

	collection := ReadAll(buffer)

	for _, frame := range collection.Data {
		assert.True(t, CheckVersion(frame), "the header version should be correct")
	}
	assert.Equal(t, 2, len(collection.Data), "there should be two frames")
}

func TestNewFrameReader(t *testing.T) {
	tmp := append(frameSample, frameSample...)

	assert.Equal(t,
		len(frameSample)*2,
		len(tmp),
		"tmp length should be the sum of all the lenghts")

	buffer := filebuffer.New(tmp)

	frameChannel := NewMultiReader([]io.Reader{buffer})

	frame1 := <-frameChannel
	frame2 := <-frameChannel
	logrus.Debugf("%+v %+v", frame1, frame2)

	assert.Equal(t, frame1, frame2, "The two frames should be equal")
	_, ok := <-frameChannel
	assert.False(t, ok, "frameChannel should be closed")
}

func init() {
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	logrus.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	// logrus.SetLevel(logrus.DebugLevel)
}
