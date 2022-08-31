package server_test

import (
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kickback-app/api/logger"
)

func TestMain(m *testing.M) {
	logger.Init("DEBUG")
	gin.SetMode(gin.ReleaseMode) // to turn off annoying logs in tests
	exitVal := m.Run()
	os.Exit(exitVal)
}
