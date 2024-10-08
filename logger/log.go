package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Logger struct {
	file *os.File
}

const LOGFILENAME = "wget-log"

var OUT *os.File

func (l *Logger) Write(p []byte) (n int, err error) {
	return io.WriteString(l.file, fmt.Sprintln(string(p)))
}

func newLogger() *Logger {
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println("can't get current dir")
		os.Exit(1)
	}

	filename := filepath.Join(pwd, LOGFILENAME)

	file, err := os.OpenFile(filename, os.O_TRUNC|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println("can't open the log file")
		os.Exit(1)
	}

	return &Logger{
		file: file,
	}
}

var logger *Logger

func init() {
	logger = newLogger()
	OUT = logger.file
}
