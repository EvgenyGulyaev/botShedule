package logger

import (
	"log"
	"os"
	"strconv"
	"sync"
)

type Logger struct {
	visible bool
}

var instance *Logger

var once sync.Once

func GetLogger() *Logger {
	once.Do(func() {
		visible, err := strconv.ParseBool(os.Getenv("LOG_SHOW"))
		if err != nil {
			panic(err)
		}
		instance = makeLogger(visible)
	})
	return instance
}

func (l *Logger) Printf(format string, v ...any) {
	if l.visible {
		log.Printf(format, v...)
	}
}

func (l *Logger) Fatal(v ...any) {
	if l.visible {
		log.Fatal(v...)
	}
}

func makeLogger(visible bool) *Logger {
	return &Logger{visible: visible}
}
