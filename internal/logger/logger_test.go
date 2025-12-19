package logger

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	logger := New("info", "text")
	assert.NotNil(t, logger)
	assert.Equal(t, logrus.InfoLevel, logger.Level)
}

func TestNewWithConfig_DefaultValues(t *testing.T) {
	// Test with minimal config
	config := Config{
		Level:  "debug",
		Format: "text",
	}
	
	logger := NewWithConfig(config)
	assert.NotNil(t, logger)
	assert.Equal(t, logrus.DebugLevel, logger.Level)
}

func TestNewWithConfig_InvalidLevel(t *testing.T) {
	// Test with invalid level, should default to info
	config := Config{
		Level:  "invalid",
		Format: "text",
	}
	
	logger := NewWithConfig(config)
	assert.NotNil(t, logger)
	assert.Equal(t, logrus.InfoLevel, logger.Level)
}

func TestNewWithConfig_JSONFormat(t *testing.T) {
	config := Config{
		Level:  "warn",
		Format: "json",
	}
	
	logger := NewWithConfig(config)
	assert.NotNil(t, logger)
	assert.Equal(t, logrus.WarnLevel, logger.Level)
	
	// Check that it uses JSON formatter by checking output
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	
	logger.Warn("test message")
	output := buf.String()
	
	// JSON logs should start with { and contain expected fields
	assert.True(t, strings.HasPrefix(output, "{"))
	assert.Contains(t, output, "level\":\"warning")
	assert.Contains(t, output, "msg\":\"test message")
}

func TestNewWithConfig_TextFormat(t *testing.T) {
	config := Config{
		Level:  "error",
		Format: "text",
	}
	
	logger := NewWithConfig(config)
	assert.NotNil(t, logger)
	assert.Equal(t, logrus.ErrorLevel, logger.Level)
	
	// Check that it uses Text formatter by checking output
	var buf bytes.Buffer
	logger.SetOutput(&buf)
	
	logger.Error("test error")
	output := buf.String()
	
	// Text logs should contain timestamp and level
	assert.Contains(t, output, "level=error")
	assert.Contains(t, output, "msg=\"test error\"")
}

func TestNewWithConfig_FileOutput(t *testing.T) {
	// Create a temporary directory for log files
	tempDir, err := ioutil.TempDir("", "logger_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	logFile := filepath.Join(tempDir, "test.log")
	
	config := Config{
		Level:      "info",
		Format:     "text",
		Output:     logFile,
		MaxSize:    1,
		MaxAge:     1,
		MaxBackups: 1,
		LocalTime:  true,
		Compress:   false,
	}
	
	logger := NewWithConfig(config)
	assert.NotNil(t, logger)
	
	// Test logging to file
	logger.Info("test message")
	
	// Check that log file was created and contains our message
	content, err := ioutil.ReadFile(logFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "test message")
}

func TestNewWithConfig_DefaultFileConfig(t *testing.T) {
	// Test that default values are set when not provided for file output
	tempDir, err := ioutil.TempDir("", "logger_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	logFile := filepath.Join(tempDir, "test.log")
	
	config := Config{
		Level:  "info",
		Format: "text",
		Output: logFile,
		// Not setting MaxSize, MaxAge, MaxBackups to test defaults
	}
	
	logger := NewWithConfig(config)
	assert.NotNil(t, logger)
	
	// Log something to make sure it works
	logger.Info("test message")
	
	// Check that log file was created
	_, err = os.Stat(logFile)
	assert.NoError(t, err)
}

func TestNewWithConfig_StdoutOutput(t *testing.T) {
	// Test explicit stdout output
	config := Config{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	}
	
	logger := NewWithConfig(config)
	assert.NotNil(t, logger)
	
	// Test with empty output (defaults to stdout)
	config.Output = ""
	logger2 := NewWithConfig(config)
	assert.NotNil(t, logger2)
}