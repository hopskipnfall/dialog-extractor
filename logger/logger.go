package logger

import (
	"fmt"
	"io/ioutil"
	"log"
)

// Logger struct for logging in console and to a text file.
type Logger struct {
	logOutput  string
	outputPath *string
}

// New Creates a new logger.
func New(outputPath *string) *Logger {
	l := &Logger{
		logOutput:  "",
		outputPath: outputPath,
	}
	return l
}

// Print fmt.Print
func (l *Logger) Print(v ...interface{}) {
	m := fmt.Sprint(v...)
	l.logOutput = l.logOutput + m
	fmt.Print(m)
}

// Fatal fmt.Print
func (l *Logger) Fatal(v ...interface{}) {
	m := fmt.Sprint(v...)
	l.logOutput = l.logOutput + m
	l.WriteToFile()
	log.Fatal(m)
}

// Println fmt.Println
func (l *Logger) Println(v ...interface{}) {
	m := fmt.Sprint(v...) + "\n"
	l.Print(m)
}

// Printlnf Formatted Println
func (l *Logger) Printlnf(format string, a ...interface{}) {
	l.Println(fmt.Sprintf(format, a...))
}

// Printf fmt.Printf
func (l *Logger) Printf(format string, a ...interface{}) {
	l.Print(fmt.Sprintf(format, a...))
}

// WriteToFile writes the log contents to a text file.
func (l *Logger) WriteToFile() error {
	if l.outputPath != nil {
		return ioutil.WriteFile(*l.outputPath, []byte(l.logOutput), 0644)
	}
	return nil
}
