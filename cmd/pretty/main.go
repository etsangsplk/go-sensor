package main

import (
	"os"
	"fmt"
	"bufio"
	"encoding/json"
	"strings"
	"time"
	"github.com/prometheus/common/log"
	"flag"
	"io"
)

const (
	nocolor = 0
	red     = 31
	green   = 32
	yellow  = 33
	blue    = 36
	gray    = 37
)

type prettyPrinter func(entry map[string]interface{})

func main(){
	var CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flag.Usage = func() {
		fmt.Fprintf(CommandLine.Output(), "Usage: %v [OPTION]... [LOGFILE]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
    args := flag.Args()

	fi, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}
	if fi.Mode() & os.ModeNamedPipe == 0 {
		// No piped input
		switch len(args) {
		case 1:
			filename := args[0]
			file, err := os.Open(filename)
			defer file.Close()
			if err != nil {
				log.Error(err)
			}
			processLines(bufio.NewReader(file), printLine)
		default:
			flag.Usage()
		}
	} else {
		// piped input
		processLines(os.Stdin, printLine)
	}
}

func processLines(r io.Reader, print prettyPrinter){
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		var entry map[string]interface{}
		json.Unmarshal([]byte(line), &entry)
		print(entry)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}
}

func extractAndRemove(m map[string]interface{}, key string) (string, bool) {
	if v, ok := m[key]; ok {
		value := v.(string)
		delete(m, key)
		return value, true
	} else {
		log.Warnf("Missing the required key '%s'.", key)
		return "", false
	}
}

func printLine(entry map[string]interface{}) {
	level,_ := extractAndRemove(entry, "level")
	timestamp, timestampExists:= extractAndRemove(entry, "time")
	file, _ := extractAndRemove(entry, "file")
	message, _ := extractAndRemove(entry, "message")

	if timestampExists {
		parsedTime, err := time.Parse(time.RFC3339, timestamp)
		if err != nil {
			log.Fatal("Can't parse timestamp: " + fmt.Sprintf("%v", timestamp))
		}
		timestamp = parsedTime.Format("0102 15:04:05.999")
	}

	var theRest []string

	for key, value := range entry {
		var keyValue string
		// quote the value if necessary
		if strings.Contains(fmt.Sprintf("%v",value), " "){
			keyValue = fmt.Sprintf("%s=\"%s\"", key, value)
		} else {
			keyValue = fmt.Sprintf("%s=%s", key, value)
		}
		theRest = append(theRest, keyValue)
	}
	theRestStr := strings.Join(theRest, " ")
	switch level = fmt.Sprintf("%-5s",level);level {
	
	case "WARN":
		level = colorize(yellow, level)
	case "ERROR":
		level = colorize(red, level)
	default:
		level = colorize(green, level)
	}

	fmt.Printf("%-17s %5s %-22s | \"%s\" %s\n", timestamp, level, file, message, theRestStr)
}

func colorize(color int, str string) string{
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color, str)
}
