package handlers_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kickback-app/api/server/handlers"
	"github.com/stretchr/testify/assert"
)

func TestHealthcheck(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	handlers.Healthcheck(c)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, 200, w.Code)
	var got gin.H
	err := json.Unmarshal(w.Body.Bytes(), &got)
	if err != nil {
		t.Fatal(err)
	}
	want := gin.H{"status": "ok"}
	assert.Equal(t, want, got)
}

func TestGetEvent(t *testing.T) {
	t.Skip()
}
