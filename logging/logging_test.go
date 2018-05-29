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

	"github.com/stretchr/testify/assert"
)

func StartLogCapturing() (chan string, *os.File) {
	r, w, _ := os.Pipe()
	outC := make(chan string)

	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, e := io.Copy(&buf, r)
		if e != nil {
			panic(e)
		}
		outC <- buf.String()
	}()
	return outC, w
}

func StopLogCapturing(outChannel chan string, writeStream *os.File) []string {
	// back to normal state
	if e := writeStream.Close(); e != nil {
		panic(e)
	}

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
	assert.Equal(t, logger.Level(), DebugLevel)
	assert.Contains(t, s[0], "A debug log statement")
	assert.Contains(t, s[1], "An info log statement")
}

func TestEnabled(t *testing.T) {
	logger := New("testlogger")
	logger.SetLevel(DebugLevel)
	assert.True(t, logger.DebugEnabled())
	assert.True(t, logger.Enabled(DebugLevel))
	logger.SetLevel(ErrorLevel)
	assert.True(t, logger.Enabled(ErrorLevel))
}

func TestDebugDisabled(t *testing.T) {
	outC, w := StartLogCapturing()
	logger := NewWithOutput("testlogger", w)
	logger.SetLevel(InfoLevel)
	logger.Debug("A debug log statement")
	logger.Info("An info log statement")
	s := StopLogCapturing(outC, w)
	assert.Equal(t, logger.Level(), InfoLevel)
	assert.NotContains(t, s[0], "A debug log statement")
	assert.Contains(t, s[0], "An info log statement")
}

func TestWith(t *testing.T) {
	outC, w := StartLogCapturing()
	logger := NewWithOutput("testlogger", w)
	logger = logger.With("arbitrary_key", "arbitrary_value")
	logger.Info("An info log statement")
	emptyFieldsLogger := logger.With()
	assert.Equal(t, logger, emptyFieldsLogger)
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
	loggerWith := logger.With("hello", "world")
	logger.Info("parent does not contain hello world")
	loggerWith.Info("child contains hello world")
	s := StopLogCapturing(outC, w)
	assert.Contains(t, s[0], "parent does not contain hello world")
	assert.NotContains(t, s[0], `"hello":"world"`)
	assert.Contains(t, s[1], "child contains hello world")
	assert.Contains(t, s[1], `"hello":"world"`)
}

func TestNoOp(t *testing.T) {
	outC, w := StartLogCapturing()
	// Since parent logger is NoOp, child is too
	logger := NewNoOp()
	childLogger := logger.With("hello", "world")
	logger.Info("parent logger")
	childLogger.Info("child logger")
	s := StopLogCapturing(outC, w)
	assert.Equal(t, []string{""}, s, "No lines should be emitted. ")
}

func TestHostname(t *testing.T) {
	outC, w := StartLogCapturing()
	hostname, e := os.Hostname()
	if e != nil {
		t.Fatal(e)
	}
	logger := NewWithOutput("testlogger", w)
	logger.Info("An info log statement")
	s := StopLogCapturing(outC, w)
	assert.Contains(t, s[0], "An info log statement")
	assert.Contains(t, s[0], fmt.Sprintf(`"hostname":"%s"`, hostname))
}

func TestFormatting(t *testing.T) {
	outC, w := StartLogCapturing()
	logger := NewWithOutput("testContextLogger", w)
	logger.Info("Time duration", "duration", time.Second*5, "durationString", (time.Second * 5).String())
	StopLogCapturing(outC, w)
}

func TestLockWriter(t *testing.T) {
	s := lockWriter(os.Stdout)
	assert.NotNil(t, s)
	s = lockWriter(os.Stderr)
	assert.NotNil(t, s)
	s = lockWriter(ioutil.Discard)
	assert.NotNil(t, s)
	var anyWriter io.Writer
	s = lockWriter(anyWriter)
	assert.NotNil(t, s)
}

func TestLockWriterToAFileStream(t *testing.T) {
	// Setup random log file and current timestamp as log string
	// for easy verification.
	name := fmt.Sprintf("%v-%v", os.Getpid(), time.Now().Second())
	f, err := ioutil.TempFile("", name)
	defer func() {
		if e := os.Remove(f.Name()); e != nil {
			t.Fatal(e)
		}
	}()
	assert.NoError(t, err)
	if _, err = ioutil.ReadFile(f.Name()); err != nil {
		t.Fatal(err)
	}
	log := NewWithOutput("testlogtempfile", f)
	log.Info(fmt.Sprintf("you log message in file %v", name))
	// Read file content for validation.
	newContentsBytes, err := ioutil.ReadFile(f.Name())
	assert.NoError(t, err)
	s := string(newContentsBytes[:])
	assert.Contains(t, s, name)
}

func TestGlobalLogger(t *testing.T) {
	logger := New("testContextLogger")
	globalLogger := Global()
	SetGlobalLogger(logger)
	assert.NotEqual(t, logger, globalLogger) // Before assigning the logger
	globalLogger = Global()
	assert.Equal(t, logger, globalLogger) // After assigning the logger
}
