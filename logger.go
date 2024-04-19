package Artifex

import (
	"encoding"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
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
	Debug(format string, a ...any)
	Info(format string, a ...any)
	Warn(format string, a ...any)
	Error(format string, a ...any)
	Fatal(format string, a ...any)

	SetLogLevel(level LogLevel)
	LogLevel() LogLevel
	WithCallDepth(externalDepth uint) Logger
	WithKeyValue(key string, v any) Logger
}

var (
	defaultLogger Logger = NewWriterLogger(os.Stdout, true, LogLevelDebug)
)

func SetDefaultLogger(l Logger) {
	defaultLogger = l
}

func DefaultLogger() Logger {
	return defaultLogger
}

func NewLogger(printPath bool, level LogLevel) Logger {
	return NewWriterLogger(os.Stdout, printPath, level)
}

func NewWriterLogger(w io.Writer, printPath bool, level LogLevel) Logger {
	flag := log.Lmsgprefix | log.LstdFlags

	if printPath {
		flag |= log.Llongfile
	}

	internalDepth := 2

	return stdLogger{
		debug:             log.New(w, "[Debug] ", flag),
		info:              log.New(w, "[Info ] ", flag),
		warn:              log.New(w, "[Warn ] ", flag),
		err:               log.New(w, "[Error] ", flag),
		fatal:             log.New(w, "[Fatal] ", flag),
		internalCallDepth: internalDepth,
		logLevel:          &level,
		content:           newContent(),
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
	content           content
}

func (l stdLogger) Debug(format string, a ...any) {
	if *l.logLevel > LogLevelDebug {
		return
	}
	l.debug.Output(l.internalCallDepth, encode(l.content, format, a))
}

func (l stdLogger) Info(format string, a ...any) {
	if *l.logLevel > LogLevelInfo {
		return
	}
	l.info.Output(l.internalCallDepth, encode(l.content, format, a))
}

func (l stdLogger) Warn(format string, a ...any) {
	if *l.logLevel > LogLevelWarn {
		return
	}
	l.warn.Output(l.internalCallDepth, encode(l.content, format, a))
}

func (l stdLogger) Error(format string, a ...any) {
	if *l.logLevel > LogLevelError {
		return
	}
	l.err.Output(l.internalCallDepth, encode(l.content, format, a))
}

func (l stdLogger) Fatal(format string, a ...any) {
	if *l.logLevel > LogLevelFatal {
		return
	}
	l.fatal.Output(l.internalCallDepth, encode(l.content, format, a))
	os.Exit(1)
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

func (l stdLogger) WithKeyValue(key string, v any) Logger {
	if key == "" {
		return l
	}
	l.content = l.content.deepCopyAndSet(key, v)
	return l
}

//

func newContent() content {
	return content{
		keys:     make([]string, 0, 4),
		contents: make(map[string]string, 4),
	}
}

type content struct {
	keys     []string
	contents map[string]string
}

func (c content) deepCopy() content {
	size := len(c.keys)

	keys := make([]string, size, cap(c.keys))
	copy(keys, c.keys)

	contents := make(map[string]string, size)
	for key, v := range c.contents {
		contents[key] = v
	}

	return content{keys: keys, contents: contents}
}

func (c content) deepCopyAndSet(key string, v any) content {
	if key == "" {
		return c
	}

	fresh := c.deepCopy()

	// set
	fresh.keys = append(fresh.keys, key)
	fresh.contents[key] = anyToString(v)

	return fresh
}

func encode(c content, format string, other ...any) string {
	builder := &strings.Builder{}
	for _, key := range c.keys {
		if c.contents[key] == "" {
			continue
		}
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(c.contents[key])
		builder.WriteString(" ")
	}
	fmt.Fprintf(builder, format, other...)
	return builder.String()
}

func anyToString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case bool:
		return strconv.FormatBool(val)
	case int:
		return strconv.Itoa(val)
	case int8:
		return strconv.FormatInt(int64(val), 10)
	case int16:
		return strconv.FormatInt(int64(val), 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint8:
		return strconv.FormatUint(uint64(val), 10)
	case uint16:
		return strconv.FormatUint(uint64(val), 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case error:
		return val.Error()
	case fmt.Stringer:
		return val.String()
	case encoding.TextMarshaler:
		text, err := val.MarshalText()
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(text)
	default:
		bData, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(bData)
	}
}

//

func SilentLogger() Logger {
	return silentLogger{}
}

type silentLogger struct{}

func (silentLogger) Debug(format string, a ...any) {
	return
}

func (silentLogger) Info(format string, a ...any) {
	return
}

func (silentLogger) Warn(format string, a ...any) {
	return
}

func (silentLogger) Error(format string, a ...any) {
	return
}

func (silentLogger) Fatal(format string, a ...any) {
	return
}

func (silentLogger) SetLogLevel(level LogLevel) {
	return
}

func (l silentLogger) LogLevel() LogLevel {
	return 0
}

func (l silentLogger) WithCallDepth(externalDepth uint) Logger {
	return l
}

func (l silentLogger) WithKeyValue(key string, v any) Logger {
	return l
}
