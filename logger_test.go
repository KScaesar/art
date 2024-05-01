package Artifex

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
	"time"
)

func Test_Logger_WithKeyValue(t *testing.T) {
	buffer := bytes.NewBuffer([]byte{})
	logger := NewWriterLogger(buffer, false, LogLevelDebug)

	logger1 := logger.WithKeyValue("user_id", strconv.Itoa(1313))
	logger2 := logger1.WithKeyValue("msg_id", "f1017")
	logger3 := logger1.WithKeyValue("game_kind", "rpg")
	logger4 := logger3.WithKeyValue("cost", time.Minute+23*time.Second)

	logger1.Info("test1")
	logger2.Info("test2")
	logger3.Info("test3")
	logger4.Info("test4")

	expectedOutput := []string{
		"user_id=1313 test1",
		"user_id=1313 msg_id=f1017 test2",
		"user_id=1313 game_kind=rpg test3",
		"user_id=1313 game_kind=rpg cost=1m23s test4",
	}

	lines := strings.Split(buffer.String(), "\n")
	for i := 0; i < len(expectedOutput); i++ {
		line := lines[i]
		index := strings.Index(line, "[ Info] ")
		if expectedOutput[i] != line[index+8:] {
			t.Errorf("unexpected output: want=%v got=%v", expectedOutput[i], line[index+8:])
		}
	}
}
