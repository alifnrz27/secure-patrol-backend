package log

import (
	"fmt"
	"io"
	"log"
	"os"
)

var logger AllLogger = &defaultLogger{
	stdLog: log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile|log.Lmicroseconds),
	depth:  4,
}

// Logger is a logger interface that provides logging function with levels.
type Logger interface {
	Trace(v ...any)
	Debug(v ...any)
	Info(v ...any)
	Warn(v ...any)
	Error(v ...any)
	Fatal(v ...any)
	Panic(v ...any)
}

// FormatLogger is logger interface that provides logging with output formated.
type FormatLooger interface {
	Tracef(format string, v ...any)
	Debugf(format string, v ...any)
	Infof(format string, v ...any)
	Warnf(format string, v ...any)
	Errorf(format string, v ...any)
	Fatalf(format string, v ...any)
	Panicf(format string, v ...any)
}

type ControlLogger interface {
	SetLevel(lv Level)
	SetOutput(w io.Writer)
}

type AllLogger interface {
	Logger
	ControlLogger
	FormatLooger
}

// Level is defines the priority of a log message.
// when a logger is configured with level, then any logger level is lower that the level
// will not be printed on the output.
type Level int

// The levels of log
const (
	LevelTrace Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
	LevelPanic
)

var strs = []string{
	"[Trace] ",
	"[Debug] ",
	"[Info] ",
	"[Error] ",
	"[Fatal] ",
	"[Panic] ",
}

func (l Level) toString() string {
	if l >= LevelTrace && l <= LevelPanic {
		return strs[l]
	}
	return fmt.Sprintf("[?%d]", l)
}
