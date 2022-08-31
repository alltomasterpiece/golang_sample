package server

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kickback-app/api/logger"
	"github.com/kickback-app/api/pkg/models"
	"github.com/kickback-app/api/server/handlers"
	"github.com/kickback-app/api/utils"
)

func (s S) GetMediaItemsMetadata(c *gin.Context) {
	param := "kickbackId"
	kickbackID := c.Param(param)
	if kickbackID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	items, err := s.MediaService.GetItemsMetadata(c, kickbackID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	result := []models.M{}
	for _, item := range items {
		result = append(result, s.MediaService.ResolveLinks(c, item))
	}
	logger.Info(c, "retrieved %d media objects for kickback %s", len(result), kickbackID)
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{"mediaItems": result})
}

func (s S) GetMediaItem(c *gin.Context) {
	param := "kickbackId"
	kickbackID := c.Param(param)
	if kickbackID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	param = "itemId"
	itemID := c.Param(param)
	if itemID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	item, err := s.MediaService.GetItem(c, kickbackID, itemID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "retrieved media item %v for kickback %s", item.ID, kickbackID)
	handlers.EncodeSuccess(c, http.StatusOK, s.MediaService.ResolveLinks(c, item))
}

func (s S) CreateMediaItem(c *gin.Context) {
	param := "kickbackId"
	kickbackID := c.Param(param)
	if kickbackID == "" {
		handlers.EncodeError(c, handlers.MissingBodyFieldError{Field: "parentId"})
		return
	}
	var item models.Media
	if err := json.NewDecoder(c.Request.Body).Decode(&item); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	itemCreated, err := s.MediaService.CreateItem(c, kickbackID, item)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "successfully added %v media objects to kickback %v", itemCreated.ID, kickbackID)
	handlers.EncodeSuccess(c, http.StatusOK, itemCreated)
}

func (s S) UpdateMediaItemMetadata(c *gin.Context) {
	param := "itemId"
	itemID := c.Param(param)
	if itemID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	var updates models.MediaUpdates
	if err := json.NewDecoder(c.Request.Body).Decode(&updates); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	err := s.MediaService.UpdateItemMetada(c, itemID, &updates)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	handlers.EncodeSuccess(c, http.StatusNoContent, nil)
}

func (s S) DeleteMediaItem(c *gin.Context) {
	param := "kickbackId"
	kickbackID := c.Param(param)
	if kickbackID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	param = "itemId"
	itemID := c.Param(param)
	if itemID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	err := s.MediaService.DeleteItem(c, kickbackID, itemID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	handlers.EncodeSuccess(c, http.StatusNoContent, nil)
}

func (s S) AddMediaItemComment(c *gin.Context) {
	param := "itemId"
	itemID := c.Param(param)
	if itemID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	var comment struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&comment); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	commentID, err := s.MediaService.AddComment(c, itemID, models.MediaComment{
		Message:   comment.Message,
		CreatedBy: utils.CurrentUser(c).ID,
	})
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "created new comment %v assocaited with media %v", commentID, itemID)
	handlers.EncodeSuccess(c, http.StatusNoContent, nil)
}

func (s S) UpdateMediaItemComment(c *gin.Context) {
	param := "itemId"
	itemID := c.Param(param)
	if itemID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	param = "commentId"
	commentID := c.Param(param)
	if commentID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	var comment struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&comment); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	err := s.MediaService.UpdateComment(c, itemID, commentID, comment.Message)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "updated comment %v assocaited with media %v", commentID, itemID)
	handlers.EncodeSuccess(c, http.StatusNoContent, nil)
}

func (s S) DeleteMediaItemComment(c *gin.Context) {
	param := "itemId"
	itemID := c.Param(param)
	if itemID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	param = "commentId"
	commentID := c.Param(param)
	if commentID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	err := s.MediaService.DeleteComment(c, itemID, commentID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "deleted comment %v assocaited with media %v", commentID, itemID)
	handlers.EncodeSuccess(c, http.StatusNoContent, nil)
}
