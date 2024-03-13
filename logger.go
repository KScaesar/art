package Artifex

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// Level

type LogLevel uint8

const (
	LogLevelDebug LogLevel = iota + 1
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

func LookupLogLevel(level LogLevel) string {
	return level_to_name[level]
}

var level_to_name = map[LogLevel]string{
	LogLevelDebug: "debug",
	LogLevelInfo:  "info",
	LogLevelWarn:  "warn",
	LogLevelError: "error",
	LogLevelFatal: "fatal",
}

// Logger

type Logger interface {
	Debug(format string, a ...interface{})
	Info(format string, a ...interface{})
	Warn(format string, a ...interface{})
	Error(format string, a ...interface{})
	Fatal(format string, a ...interface{})

	Copy() Logger
	SetLogLevel(level LogLevel)
	WithCallDepth(externalDepth uint) Logger
	WithMessageId(msgId string) Logger
}

var (
	defaultLogger Logger = NewLogger(true, LogLevelDebug)
)

func SetDefaultLogger(l Logger) {
	defaultLogger = l
}

func DefaultLogger() Logger {
	return defaultLogger
}

func NewLogger(printPath bool, level LogLevel) Logger {
	flag := log.Lmsgprefix | log.LstdFlags

	if printPath {
		flag |= log.Llongfile
	}

	internalDepth := 2

	return stdLogger{
		debug:             log.New(os.Stdout, "[Debug] ", flag),
		info:              log.New(os.Stdout, "[Info ] ", flag),
		warn:              log.New(os.Stdout, "[Warn ] ", flag),
		err:               log.New(os.Stdout, "[Error] ", flag),
		fatal:             log.New(os.Stdout, "[Fatal] ", flag),
		internalCallDepth: internalDepth,
		logLevel:          &level,

		contextKey: []string{
			"msg_id",
		},
		contextInfo: newContextInfo(),
	}
}

type stdLogger struct {
	debug             *log.Logger
	info              *log.Logger
	warn              *log.Logger
	err               *log.Logger
	fatal             *log.Logger
	internalCallDepth int
	logLevel          *LogLevel

	contextKey  []string
	contextInfo contextInfo
}

func (l stdLogger) Debug(format string, a ...interface{}) {
	if *l.logLevel > LogLevelDebug {
		return
	}
	format = l.processPreformat(format)
	l.debug.Output(l.internalCallDepth, fmt.Sprintf(format, a...))
}

func (l stdLogger) Info(format string, a ...interface{}) {
	if *l.logLevel > LogLevelInfo {
		return
	}
	format = l.processPreformat(format)
	l.info.Output(l.internalCallDepth, fmt.Sprintf(format, a...))
}

func (l stdLogger) Warn(format string, a ...interface{}) {
	if *l.logLevel > LogLevelWarn {
		return
	}
	format = l.processPreformat(format)
	l.warn.Output(l.internalCallDepth, fmt.Sprintf(format, a...))
}

func (l stdLogger) Error(format string, a ...interface{}) {
	if *l.logLevel > LogLevelError {
		return
	}
	format = l.processPreformat(format)
	l.err.Output(l.internalCallDepth, fmt.Sprintf(format, a...))
}

func (l stdLogger) Fatal(format string, a ...interface{}) {
	if *l.logLevel > LogLevelFatal {
		return
	}
	format = l.processPreformat(format)
	l.fatal.Output(l.internalCallDepth, fmt.Sprintf(format, a...))
	os.Exit(1)
}

func (l stdLogger) Copy() Logger {
	l.contextInfo = l.contextInfo.copy()
	return l
}

func (l stdLogger) SetLogLevel(level LogLevel) {
	*l.logLevel = level
}

func (l stdLogger) LogLevel() LogLevel {
	return *l.logLevel
}

func (l stdLogger) WithCallDepth(externalDepth uint) Logger {
	l.internalCallDepth += int(externalDepth)
	return l
}

func (l stdLogger) processPreformat(format string) string {
	b := strings.Builder{}

	for _, key := range l.contextKey {
		value := l.contextInfo.get(key)
		if value == "" {
			continue
		}

		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(value)
		b.WriteString(" ")
	}

	b.WriteString(format)
	return b.String()
}

func (l stdLogger) WithMessageId(msgId string) Logger {
	if msgId == "" {
		return l
	}

	l.contextInfo = l.contextInfo.copyAndSet("msg_id", msgId)
	return l
}

//

func newContextInfo() map[string]string {
	return make(map[string]string)
}

type contextInfo map[string]string

func (ctx contextInfo) copy() contextInfo {
	source := ctx
	size := len(source)
	target := make(map[string]string, size)
	for key, v := range source {
		target[key] = v
	}
	return target
}

func (ctx contextInfo) set(key, v string) {
	ctx[key] = v
}

func (ctx contextInfo) copyAndSet(key, v string) contextInfo {
	target := ctx.copy()
	target.set(key, v)
	return target
}

func (ctx contextInfo) get(key string) string {
	return ctx[key]
}

//

func SilentLogger() Logger {
	return silentLogger{}
}

type silentLogger struct{}

func (silentLogger) Debug(format string, a ...interface{}) {
	return
}

func (silentLogger) Info(format string, a ...interface{}) {
	return
}

func (silentLogger) Warn(format string, a ...interface{}) {
	return
}

func (silentLogger) Error(format string, a ...interface{}) {
	return
}

func (silentLogger) Fatal(format string, a ...interface{}) {
	return
}

func (l silentLogger) Copy() Logger {
	return l
}

func (silentLogger) SetLogLevel(level LogLevel) {
	return
}

func (l silentLogger) WithCallDepth(externalDepth uint) Logger {
	return l
}

func (l silentLogger) WithMessageId(msgId string) Logger {
	return l
}
