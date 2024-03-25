package Artifex

import (
	"strconv"
	"testing"
)

func Test_stdLogger_WithKeyValue(t *testing.T) {
	logger := DefaultLogger()
	logger1 := logger.WithKeyValue("user_id", strconv.Itoa(1313))
	logger2 := logger1.WithKeyValue("msg_id", GenerateRandomCode(3))
	logger3 := logger1.WithKeyValue("game_kind", "rpg")

	logger1.Info("test")
	logger2.Info("test")
	logger3.Info("test")
}
