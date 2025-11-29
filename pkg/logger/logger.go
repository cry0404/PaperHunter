package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)


type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

var (
	levelNames = map[Level]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
	}
	levelColors = map[Level]string{
		DEBUG: "\033[36m", 
		INFO:  "\033[32m", 
		WARN:  "\033[33m", 
		ERROR: "\033[31m", 
	}
	reset = "\033[0m"
)


type Logger struct {
	mu       sync.Mutex
	level    Level
	out      io.Writer
	prefix   string
	useColor bool
}

var (
	std     *Logger
	stdOnce sync.Once
)


func Init(level string, useColor bool) {
	stdOnce.Do(func() {
		l := parseLevel(level)
		std = &Logger{
			level:    l,
			out:      os.Stderr,
			useColor: useColor,
		}
	})
}


func Get() *Logger {
	if std == nil {
		Init("INFO", true)
	}
	return std
}


func SetLevel(level string) {
	Get().mu.Lock()
	defer Get().mu.Unlock()
	Get().level = parseLevel(level)
}

func parseLevel(s string) Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}


func Debug(format string, v ...interface{}) {
	Get().log(DEBUG, format, v...)
}


func Info(format string, v ...interface{}) {
	Get().log(INFO, format, v...)
}


func Warn(format string, v ...interface{}) {
	Get().log(WARN, format, v...)
}


func Error(format string, v ...interface{}) {
	Get().log(ERROR, format, v...)
}


func Fatal(format string, v ...interface{}) {
	Get().log(ERROR, format, v...)
	os.Exit(1)
}

func (l *Logger) log(level Level, format string, v ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, v...)
	levelStr := levelNames[level]

	var output string
	if l.useColor {
		color := levelColors[level]
		output = fmt.Sprintf("%s[%s]%s %s", color, levelStr, reset, msg)
	} else {
		output = fmt.Sprintf("[%s] %s", levelStr, msg)
	}

	if l.prefix != "" {
		output = fmt.Sprintf("[%s] %s", l.prefix, output)
	}


	log.SetOutput(l.out)
	log.SetFlags(log.Ldate | log.Ltime)
	log.Println(output)
}


func WithPrefix(prefix string) *Logger {
	parent := Get()
	return &Logger{
		level:    parent.level,
		out:      parent.out,
		prefix:   prefix,
		useColor: parent.useColor,
	}
}



func InitWithFile(level string, useColor bool, logFile string) {
	 stdOnce.Do(func() {
        l := parseLevel(level)
        
        var out io.Writer
        if logFile != "" {
            if file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
                out = file
            } else {
                out = os.Stderr
            }
        } else {
            out = os.Stderr
        }
        
        std = &Logger{
            level:    l,
            out:      out,
            useColor: false, 
        }
    })
}
