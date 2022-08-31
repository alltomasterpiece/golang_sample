package server

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kickback-app/api/server/handlers"
)

func (s S) GetNote(c *gin.Context) {
	param := "kickbackId"
	kickbackID := c.Param(param)
	if kickbackID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	note, err := s.NoteService.GetNote(c, kickbackID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	handlers.EncodeSuccess(c, http.StatusOK, note)
}

func (s S) UpdateNote(c *gin.Context) {
	param := "kickbackId"
	kickbackID := c.Param(param)
	if kickbackID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	var data map[string]string
	if err := json.NewDecoder(c.Request.Body).Decode(&data); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	content, ok := data["content"]
	if !ok {
		handlers.EncodeError(c, handlers.MalformedBodyError{Field: "content"})
		return
	}
	err := s.NoteService.UpdateNote(c, kickbackID, content)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	handlers.EncodeSuccess(c, http.StatusNoContent, nil)
}
