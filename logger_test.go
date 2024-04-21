package Artifex

import (
	"strconv"
	"testing"
	"time"
)

func Test_stdLogger_WithKeyValue(t *testing.T) {
	logger := DefaultLogger()
	logger1 := logger.WithKeyValue("user_id", strconv.Itoa(1313))
	logger2 := logger1.WithKeyValue("msg_id", GenerateRandomCode(3))
	logger3 := logger1.WithKeyValue("game_kind", "rpg")
	logger4 := logger3.WithKeyValue("time", time.Minute+23*time.Second)

	logger1.Info("test1")
	logger2.Info("test2")
	logger3.Info("test3")
	logger4.Info("test4")
}
