package opentracing

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

// PopEnv takes the list of the environment values and injects them into the
// process's environment variable data. Clears any existing environment values
// that may already exist.
func PopEnv(env []string) {
	// Rip https://github.com/aws/aws-sdk-go/awstesting/util.go
	os.Clearenv()
	for _, e := range env {
		p := strings.SplitN(e, "=", 2)
		k, v := p[0], ""
		if len(p) > 1 {
			v = p[1]
		}
		os.Setenv(k, v)
	}
}

// StashEnv stashes the current environment variables and returns an array of
// all environment values as key=val strings.
func StashEnv() []string {
	// Rip https://github.com/aws/aws-sdk-go/awstesting/util.go
	env := os.Environ()
	os.Clearenv()
	return env
}

func StartLogCapturing() (chan string, *os.File) {
	r, w, _ := os.Pipe()
	outC := make(chan string)

	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		_, e := io.Copy(&buf, r)
		if e != nil {
			fmt.Printf("write stream panic: %v \n", e)
			panic(e)
		}
		outC <- buf.String()
	}()
	return outC, w
}

func StopLogCapturing(outChannel chan string, writeStream *os.File) []string {
	// back to normal state
	if e := writeStream.Close(); e != nil {
		fmt.Printf("closing write stream panic: %v \n", e)
		panic(e)
	}

	logOutput := <-outChannel

	// Verify call stack contains information we care about
	s := strings.Split(logOutput, "\n")
	return s
}
