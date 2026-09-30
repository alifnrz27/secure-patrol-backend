package log

import "io"

// Trace call the default logger's Trace method
func Trace(v ...any) {
	logger.Trace(v...)
}

// Debug call the default logger's Debug method
func Debug(v ...any) {
	logger.Debug(v...)
}

// Info call the default logger's Info method
func Info(v ...any) {
	logger.Info(v...)
}

// Warn call the default logger's Warn method
func Warn(v ...any) {
	logger.Warn(v...)
}

// Error call the default logger's Error method
func Error(v ...any) {
	logger.Error(v...)
}

// Fatal call the default logger's Fatal method and then os.Exit(1)
func Fatal(v ...any) {
	logger.Fatal(v...)
}

// Panic call the default logger's Panic method
func Panic(v ...any) {
	logger.Panic(v...)
}

// Trace call the default logger's Trace method
func Tracef(format string, v ...any) {
	logger.Tracef(format, v...)
}

// Debug call the default logger's Debug method
func Debugf(format string, v ...any) {
	logger.Debugf(format, v...)
}

// Info call the default logger's Info method
func Infof(format string, v ...any) {
	logger.Infof(format, v...)
}

// Warn call the default logger's Warn method
func Warnf(format string, v ...any) {
	logger.Warnf(format, v...)
}

// Error call the default logger's Error method
func Errorf(format string, v ...any) {
	logger.Errorf(format, v...)
}

// Fatal call the default logger's Fatal method and then os.Exit(1)
func Fatalf(format string, v ...any) {
	logger.Fatalf(format, v...)
}

// Panic call the default logger's Panic method
func Panicf(format string, v ...any) {
	logger.Panicf(format, v...)
}

// SetLogger is set the default logger instance.
// Note that this method is not concurrent-safe and must not be called
// after the use of DefaultLogger and global functions privateLog this package.
func SetLooger(v AllLogger) {
	logger = v
}

// SetOutput sets the output of default logger and system logger. By default, it is stderr.
func SetOutput(w io.Writer) {
	logger.SetOutput(w)
}

// SetLevel sets the level of logs below which logs will not be output.
// The default logger is LevelTrace.
// Note that this method is not concurrent-safe.
func SetLevel(lv Level) {
	logger.SetLevel(lv)
}
