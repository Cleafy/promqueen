package model

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSameVersion(t *testing.T) {
	header := NewHeader("")
	assert.True(t, CheckVersion(header), "header version should be equal")
	header.Version = [3]byte{0x00, 0x11, 0x00}
	assert.False(t, CheckVersion(header), "header version should not be equal")
}

func TestHeaderCreation(t *testing.T) {
	uri := "http://testuri:9090/testuri"
	header := NewHeader(uri)
	assert.True(t, CheckVersion(header), "header version should be equal")
	assert.Equal(t, len(header.URIString()), len(uri), "uristring should be equal")
	assert.Equal(t, int64(HeaderLength), int64(dataSize(reflect.ValueOf(*header))), "header should be of size 1024")
}

func TestFrameCreation(t *testing.T) {
	frame := NewFrame([]byte("foobar"))
	assert.Equal(t, frame.Size, int64(6), "length should be equal")
	assert.Equal(t, frame.Data, []byte("foobar"), "data contained should be equal")
	assert.NotZero(t, frame.Timestamp, "timestamp should be setted and different from zero")
	assert.Condition(t, func() bool {
		return time.Now().Unix() >= frame.Timestamp
	}, "current timestamp should be greater or equal that frame creation timestamp")
}
