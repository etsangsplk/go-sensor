package logging

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"gopkg.in/mgo.v2/bson"
	"github.com/stretchr/testify/assert"
)

func StartLogCapturing() (chan string, *os.File) {
	r, w, _ := os.Pipe()
	outC := make(chan string)

	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outC <- buf.String()
	}()
	return outC, w
}

func StopLogCapturing(outChannel chan string, writeStream *os.File) []string {
	// back to normal state
	writeStream.Close()
	logOutput := <-outChannel

	// Verify call stack contains information we care about
	s := strings.Split(logOutput, "\n")
	return s
}

func TestLevels(t *testing.T) {
	outC, w := StartLogCapturing()
	logger := NewWithOutput("testlogger", w)
	logger.Debug("This is a debug log entry!")
	logger.Info("This is a info log entry!")
	logger.Warn("This is a warning log entry!")
	logger.Error(fmt.Errorf("An error"), "This is a error log entry!")
	s := StopLogCapturing(outC, w)
	assert.Contains(t, s[0], `"message":"This is a info log entry!"`)
	assert.Contains(t, s[1], `"message":"This is a warning log entry!"`)
	assert.Contains(t, s[2], `"message":"This is a error log entry!"`)
	assert.Contains(t, s[0], `"level":"INFO"`)
	assert.Contains(t, s[1], `"level":"WARN"`)
	assert.Contains(t, s[2], `"level":"ERROR"`)
}

func TestDebugEnabled(t *testing.T) {
	outC, w := StartLogCapturing()
	logger := NewWithOutput("testlogger", w)
	logger.SetLevel(DebugLevel)
	logger.Debug("A debug log statement")
	logger.Info("An info log statement")
	s := StopLogCapturing(outC, w)
	assert.Contains(t, s[0], "A debug log statement")
	assert.Contains(t, s[1], "An info log statement")
}

func TestDebugDisabled(t *testing.T) {
	outC, w := StartLogCapturing()
	logger := NewWithOutput("testlogger", w)
	logger.SetLevel(InfoLevel)
	logger.Debug("A debug log statement")
	logger.Info("An info log statement")
	s := StopLogCapturing(outC, w)
	assert.NotContains(t, s[0], "A debug log statement")
	assert.Contains(t, s[0], "An info log statement")
}

func TestWith(t *testing.T) {
	outC, w := StartLogCapturing()
	logger := NewWithOutput("testlogger", w)
	logger = logger.With("arbitrary_key", "arbitrary_value")
	logger.Info("An info log statement")
	s := StopLogCapturing(outC, w)
	assert.Contains(t, s[0], "An info log statement")
	assert.Contains(t, s[0], "\"arbitrary_key\":\"arbitrary_value\"")
}

func TestRequiredFields(t *testing.T) {
	outC, w := StartLogCapturing()
	logger := NewWithOutput("testlogger", w)
	logger.SetLevel(DebugLevel)
	logger.Info("An info log statement")
	s := StopLogCapturing(outC, w)
	assert.Contains(t, s[0], `"message":"An info log statement"`)
	assert.Contains(t, s[0], `"service":"testlogger"`)
	assert.Contains(t, s[0], `"time":`)
	assert.Contains(t, s[0], `"level":"INFO"`)
}

func TestNewWithOuput(t *testing.T) {
	outC, w := StartLogCapturing()
	logger := NewWithOutput("testlogger", w)
	ctxLogger := logger.With("hello", "world")
	logger.Info("parent does not contain hello world")
	ctxLogger.Info("child contains hello world")
	s := StopLogCapturing(outC, w)
	assert.Contains(t, s[0], "parent does not contain hello world")
	assert.NotContains(t, s[0], `"hello":"world"`)
	assert.Contains(t, s[1], "child contains hello world")
	assert.Contains(t, s[1], `"hello":"world"`)
}

func TestHostname(t *testing.T) {
	outC, w := StartLogCapturing()
	os.Setenv("HOSTNAME", "testhostname")
	logger := NewWithOutput("testlogger", w)
	logger.Info("An info log statement")
	s := StopLogCapturing(outC, w)
	assert.Contains(t, s[0], "An info log statement")
	assert.Contains(t, s[0], `"hostname":"testhostname"`)
}

func TestFormatting(t *testing.T) {
	log := New("testLogger")
	log.Info("Time duration", "duration", time.Second*5, "durationString", (time.Second * 5).String())
}

func TestLock(t *testing.T) {
	s := Lock(os.Stdout)
	assert.NotNil(t, s)
	s = Lock(os.Stderr)
	assert.NotNil(t, s)
	s = Lock(ioutil.Discard)
	assert.NotNil(t, s)
	var anyWriter io.Writer
	s = Lock(anyWriter)
	assert.NotNil(t, s)
}

func TestLockToAFileStream(t *testing.T) {
	// Setup random log file and current timestamp as log string 
	// for easy verification.
	name :=  bson.NewObjectId().Hex()
	f, err := ioutil.TempFile("", name)
	defer os.Remove(f.Name())
	assert.NoError(t, err)
	_, err = ioutil.ReadFile(f.Name())
	log := NewWithOutput("testlogtempfile", f)
	// Write current time
	now := time.Now().String()
	log.Info(now)
	// Read contents for validation.
	newContentsBytes, err := ioutil.ReadFile(f.Name())
	assert.NoError(t, err)
	s := string(newContentsBytes[:])
	assert.Contains(t, s, now)
}
