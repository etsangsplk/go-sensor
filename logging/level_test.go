package logging

import (
	"testing"

	"errors"
	"github.com/stretchr/testify/assert"
	"strings"
)

type levelStringPairs struct {
	Level  Level
	String string
}

var levelStrings = []levelStringPairs{
	{FatalLevel, "fatal"},
	{ErrorLevel, "error"},
	{DebugLevel, "debug"},
	{InfoLevel, "info"},
	{WarnLevel, "warn"},
}

func TestStringify(t *testing.T) {
	for _, v := range levelStrings {
		actual := v.Level.String()
		assert.Equal(t, v.String, actual)
	}
}

func TestParseLevel(t *testing.T) {
	// test with lowercase level strings
	for _, v := range levelStrings {
		actual, err := ParseLevel(strings.ToLower(v.String))
		assert.Equal(t, v.Level, actual)
		assert.NoError(t, err)
	}
	// test with uppercase level strings
	for _, v := range levelStrings {
		actual, err := ParseLevel(strings.ToUpper(v.String))
		assert.Equal(t, v.Level, actual)
		assert.NoError(t, err)
	}
	// test with invalid
	_, err := ParseLevel("something")
	assert.Equal(t, errors.New("unrecognized level: \"something\""), err)
}

func TestMarshal(t *testing.T) {
	for _, v := range levelStrings {
		data, err := v.Level.MarshalText()
		text := string(data)
		assert.Equal(t, nil, err)
		assert.Equal(t, v.String, text)
	}
}

func TestUnmarshal(t *testing.T) {
	// test with lowercase level strings
	for _, v := range levelStrings {
		var level Level
		err := level.UnmarshalText([]byte(strings.ToLower(v.String)))
		assert.Equal(t, nil, err)
		assert.Equal(t, v.Level, level)
	}
	// test with uppercase level strings
	for _, v := range levelStrings {
		var level Level
		err := level.UnmarshalText([]byte(strings.ToUpper(v.String)))
		assert.Equal(t, nil, err)
		assert.Equal(t, v.Level, level)
	}
	// test with invalid
	var level Level
	err := level.UnmarshalText([]byte("something"))
	assert.Equal(t, errors.New("unrecognized level: \"something\""), err)
}
