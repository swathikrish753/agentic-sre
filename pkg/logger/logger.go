package logger

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// Logger is the app-wide logger type. Aliasing logrus.Entry means the rest
// of the codebase imports our package, not logrus directly — so we could
// swap the backend later without touching call sites.
type Logger = *logrus.Entry

// New builds a configured logger from a level ("debug".."error") and
// format ("json"|"text"). It never fails: an unknown level falls back to info.
func New(level, format string) Logger {
	l := logrus.New()
	l.SetOutput(os.Stdout)

	lvl, err := logrus.ParseLevel(strings.ToLower(level))
	if err != nil {
		lvl = logrus.InfoLevel
	}
	l.SetLevel(lvl)

	if strings.EqualFold(format, "json") {
		l.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	} else {
		l.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "15:04:05.000",
		})
	}

	// Return an *Entry (not the *Logger) so callers always have a place to
	// attach structured fields. A base "service" field gets set by WithService.
	return logrus.NewEntry(l)
}

// WithService tags every line from a binary with its name — invaluable once
// five binaries are interleaving logs in one cluster.
func WithService(l Logger, name string) Logger {
	return l.WithField("service", name)
}
