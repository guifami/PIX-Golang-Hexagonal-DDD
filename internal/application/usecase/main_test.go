package usecase_test

import (
	"os"
	"testing"

	"go-api/internal/infrastructure/logger"
)

func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}
