package log

import (
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type ConfdFormatter struct {
}

func (c *ConfdFormatter) Format(entry *log.Entry) ([]byte, error) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	hostname, _ := os.Hostname()
	return []byte(fmt.Sprintf("%s %s %s[%d]: %s %s\n", timestamp, hostname, tag, os.Getpid(), strings.ToUpper(entry.Level.String()), entry.Message)), nil
}

var tag string

func init() {
	tag = os.Args[0]
	log.SetFormatter(&ConfdFormatter{})
}

func SetTag(t string) {
	tag = t
}

func SetLevel(level string) {
	lvl, err := log.ParseLevel(level)
	if err != nil {
		Fatal(fmt.Sprintf(`not a valid level: "%s"`, level))
	}
	log.SetLevel(lvl)
}

func Debug(format string, v ...interface{}) {
	log.Debug(fmt.Sprintf(format, v...))
}

func Error(format string, v ...interface{}) {
	log.Error(fmt.Sprintf(format, v...))
}

func Fatal(format string, v ...interface{}) {
	log.Fatal(fmt.Sprintf(format, v...))
}

func Info(format string, v ...interface{}) {
	log.Info(fmt.Sprintf(format, v...))
}

func Warning(format string, v ...interface{}) {
	log.Warning(fmt.Sprintf(format, v...))
}
