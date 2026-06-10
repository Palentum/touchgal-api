package logging

import (
	"fmt"
	"io"
	"strings"

	"github.com/rs/zerolog"
)

const validLevels = "trace, debug, info, warn, error, fatal"

func ParseLevel(name string) (zerolog.Level, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	switch normalized {
	case "trace", "debug", "info", "warn", "error", "fatal":
	default:
		return zerolog.InfoLevel, fmt.Errorf("LOG_LEVEL must be one of: %s", validLevels)
	}
	level, _ := zerolog.ParseLevel(normalized)
	return level, nil
}

func New(levelName string, out io.Writer, console bool) (zerolog.Logger, error) {
	level, err := ParseLevel(levelName)
	if err != nil {
		return zerolog.Nop(), err
	}

	var writer io.Writer = out
	if console {
		writer = zerolog.ConsoleWriter{Out: out}
	}

	return zerolog.New(writer).Level(level).With().Timestamp().Logger(), nil
}
