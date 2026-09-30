package log

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/valyala/bytebufferpool"
)

var _ AllLogger = (*defaultLogger)(nil)

// The default logger configuration
type defaultLogger struct {
	stdLog *log.Logger
	level  Level
	depth  int
}

// writeLog is a message of log at a given level log,
// when the level is fatal, it will exit the program
func (l *defaultLogger) writeLog(lv Level, args []interface{}) {
	if l.level > lv {
		return
	}
	level := lv.toString()
	buf := bytebufferpool.Get()
	_, _ = buf.WriteString(level)
	_, _ = buf.WriteString(fmt.Sprint(args...))

	_ = l.stdLog.Output(l.depth, buf.String())
	buf.Reset()
	bytebufferpool.Put(buf)
	if lv == LevelFatal {
		os.Exit(1)
	}
}

// writeLogf is a log message at given level log,
// when the level is fatal, it will exit the program
func (l *defaultLogger) writeLogf(lv Level, format string, args []interface{}) {
	if l.level > lv {
		return
	}

	level := lv.toString()
	buf := bytebufferpool.Get()
	_, _ = buf.WriteString(level)

	if len(args) > 0 {
		_, _ = fmt.Fprintf(buf, format, args...)
	} else {
		_, _ = fmt.Fprint(buf, args...)
	}

	_ = l.stdLog.Output(l.depth, buf.String())
	buf.Reset()
	bytebufferpool.Put(buf)
	if lv == LevelFatal {
		os.Exit(1)
	}
}

func (l *defaultLogger) Trace(v ...any) {
	l.writeLog(LevelTrace, v)
}

func (l *defaultLogger) Debug(v ...any) {
	l.writeLog(LevelDebug, v)
}

func (l *defaultLogger) Info(v ...any) {
	l.writeLog(LevelInfo, v)
}

func (l *defaultLogger) Warn(v ...any) {
	l.writeLog(LevelWarn, v)
}

func (l *defaultLogger) Error(v ...any) {
	l.writeLog(LevelError, v)
}

func (l *defaultLogger) Fatal(v ...any) {
	l.writeLog(LevelFatal, v)
}

func (l *defaultLogger) Panic(v ...any) {
	l.writeLog(LevelPanic, v)
}

func (l *defaultLogger) Tracef(format string, v ...any) {
	l.writeLogf(LevelTrace, format, v)
}

func (l *defaultLogger) Debugf(format string, v ...any) {
	l.writeLogf(LevelDebug, format, v)
}

func (l *defaultLogger) Infof(format string, v ...any) {
	l.writeLogf(LevelInfo, format, v)

}

func (l *defaultLogger) Warnf(format string, v ...any) {
	l.writeLogf(LevelWarn, format, v)

}

func (l *defaultLogger) Errorf(format string, v ...any) {
	l.writeLogf(LevelError, format, v)

}

func (l *defaultLogger) Fatalf(format string, v ...any) {
	l.writeLogf(LevelFatal, format, v)

}

func (l *defaultLogger) Panicf(format string, v ...any) {
	l.writeLogf(LevelPanic, format, v)

}

func (l *defaultLogger) SetLevel(level Level) {
	l.level = level
}

func (l *defaultLogger) SetOutput(w io.Writer) {
	l.stdLog.SetOutput(w)
}
