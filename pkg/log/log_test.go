package log_test

import (
	"os"
	"testing"

	"github.com/pod-arcade/pod-arcade/pkg/log"
	"github.com/rs/zerolog"
)

func TestLevelForComponent(t *testing.T) {
	os.Setenv(log.EnvLogLevel, "info")
	os.Setenv("LOG_LEVEL_TEST", "debug")
	os.Setenv("LOG_LEVEL_TEST_2", "warn")
	os.Setenv("LOG_LEVEL_TEST_3", "trace")
	os.Setenv("LOG_LEVEL_TEST_4", "none")
	os.Setenv("LOG_LEVEL_HDMI_AUDIO_CAPTURE", "none")

	if log.LevelForComponent("test") != zerolog.DebugLevel {
		t.Errorf("Expected debug, got %v", log.LevelForComponent("test"))
	}
	if log.LevelForComponent("test-2") != zerolog.WarnLevel {
		t.Errorf("Expected warn, got %v", log.LevelForComponent("test-2"))
	}
	if log.LevelForComponent("test.3") != zerolog.TraceLevel {
		t.Errorf("Expected trace, got %v", log.LevelForComponent("test.3"))
	}
	if log.LevelForComponent("test_4") != zerolog.Disabled {
		t.Errorf("Expected disabled, got %v", log.LevelForComponent("test_4"))
	}
	if log.LevelForComponent("HDMI Audio Capture") != zerolog.Disabled {
		t.Errorf("Expected disabled, got %v", log.LevelForComponent("test_4"))
	}
}
