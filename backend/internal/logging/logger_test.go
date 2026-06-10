package logging

import (
	"bytes"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestNewHonorsDebugLevel(t *testing.T) {
	var buf bytes.Buffer
	logger, err := New("debug", &buf, false)
	if err != nil {
		t.Fatalf("expected debug logger: %v", err)
	}

	logger.Debug().Msg("debug enabled")

	output := buf.String()
	if !strings.Contains(output, `"level":"debug"`) || !strings.Contains(output, `"message":"debug enabled"`) {
		t.Fatalf("expected debug event, got %q", output)
	}
}

func TestNewSuppressesDebugAtInfoLevel(t *testing.T) {
	var buf bytes.Buffer
	logger, err := New("info", &buf, false)
	if err != nil {
		t.Fatalf("expected info logger: %v", err)
	}

	logger.Debug().Msg("debug suppressed")

	if buf.Len() != 0 {
		t.Fatalf("expected debug event to be suppressed, got %q", buf.String())
	}
}

func TestParseLevelNormalizesCaseAndWhitespace(t *testing.T) {
	level, err := ParseLevel(" DEBUG ")
	if err != nil {
		t.Fatalf("expected DEBUG to parse: %v", err)
	}
	if level != zerolog.DebugLevel {
		t.Fatalf("expected debug level, got %s", level.String())
	}
}

func TestParseLevelRejectsUnknownLevel(t *testing.T) {
	if _, err := ParseLevel("verbose"); err == nil {
		t.Fatal("expected invalid LOG_LEVEL error")
	}
}
