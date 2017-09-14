package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFrameCreation(t *testing.T) {
	uri := "http://testtest:9090/net"
	name := "test"
	frame := NewFrame(uri, name, []byte("foobar"))
	assert.Equal(t, frame.Header.Size, int64(6), "length should be equal")
	assert.Equal(t, frame.Data, []byte("foobar"), "data contained should be equal")
	assert.NotZero(t, frame.Header.Timestamp, "timestamp should be setted and different from zero")
	assert.Condition(t, func() bool {
		return time.Now().Unix() >= frame.Header.Timestamp
	}, "current timestamp should be greater or equal that frame creation timestamp")
}
