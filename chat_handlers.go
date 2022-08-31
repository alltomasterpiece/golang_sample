package server

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kickback-app/api/logger"
	"github.com/kickback-app/api/pkg/models"
	"github.com/kickback-app/api/server/handlers"
	"github.com/kickback-app/api/utils"
	"gopkg.in/mgo.v2/bson"
)

func (s S) GetChannels(c *gin.Context) {
	param := "kickbackId"
	kickbackID := c.Param(param)
	if kickbackID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	channels, err := s.ChatService.GetChannels(c, kickbackID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	// get kickback info to add isHost data
	kickback, err := s.EventService.GetEvent(c, kickbackID)
	if err != nil {
		logger.Error(c, "unable to retrieve kickback information: %v", err)
		handlers.EncodeError(c, err)
		return
	}
	channelsWithInfo := []interface{}{}
	for _, channel := range channels {
		channelMap := utils.Normalize(channel)
		membersWithInfo := []interface{}{}
		totalMembers := channel.Members
		if !utils.ContainsString(totalMembers, channel.CreatedBy) {
			totalMembers = append(totalMembers, channel.CreatedBy)
		}
		memberUsers := s.UserService.SummarizeUsers(c, totalMembers)
		for _, member := range memberUsers.Summaries {
			memberUserID, ok := member["_id"].(string)
			if !ok {
				memberUserID = ""
				logger.Warn(c, "empty userId found for %+v", member)
			}
			member["isHost"] = utils.ContainsString(kickback.Hosts, memberUserID)
			membersWithInfo = append(membersWithInfo, member)
		}
		channelMap["members"] = membersWithInfo
		channelsWithInfo = append(channelsWithInfo, channelMap)
	}
	logger.Info(c, "retrieved %d chats for kickback %s", len(channelsWithInfo), kickbackID)
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{"chats": channelsWithInfo})
}

func (s S) CreateChannel(c *gin.Context) {
	param := "kickbackId"
	kickbackID := c.Param(param)
	if kickbackID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	var channel models.Channel
	if err := json.NewDecoder(c.Request.Body).Decode(&channel); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	ID, err := s.ChatService.CreateChannel(c, kickbackID, &channel)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "created new chat %s for kickback %s", ID, kickbackID)
	handlers.EncodeSuccess(c, http.StatusOK, bson.M{"chatId": ID})
}

func (s S) UpdateChannel(c *gin.Context) {
	param := "channelId"
	channelID := c.Param(param)
	if channelID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	var updates models.ChannelUpdates
	if err := json.NewDecoder(c.Request.Body).Decode(&updates); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	err := s.ChatService.UpdateChannel(c, channelID, &updates)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	handlers.EncodeSuccess(c, http.StatusNoContent, nil)
}

func (s S) DeleteChannel(c *gin.Context) {
	param := "channelId"
	channelID := c.Param(param)
	if channelID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	err := s.ChatService.DeleteChannel(c, channelID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	handlers.EncodeSuccess(c, http.StatusNoContent, nil)
}

func (s S) UpdateChannelMembers(c *gin.Context) {
	param := "channelId"
	channelID := c.Param(param)
	if channelID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	var updates struct {
		MembersToAdd    []string `json:"members_to_add"`
		MembersToRemove []string `json:"members_to_remove"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&updates); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	err := s.ChatService.UpdateChannelMembers(c, channelID, updates.MembersToAdd, updates.MembersToRemove)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	handlers.EncodeSuccess(c, http.StatusNoContent, nil)
}

func (s S) PinChatMessage(c *gin.Context) {
	param := "channelId"
	channelID := c.Param(param)
	if channelID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	param = "kickbackId"
	kickbackID := c.Param(param)
	if kickbackID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	var message models.PinnedMessage
	if err := json.NewDecoder(c.Request.Body).Decode(&message); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	err := s.ChatService.PinMessage(c, kickbackID, channelID, &message)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	handlers.EncodeSuccess(c, http.StatusNoContent, nil)
}

func (s S) UnpinChatMessage(c *gin.Context) {
	param := "channelId"
	channelID := c.Param(param)
	if channelID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	var message models.PinnedMessage
	if err := json.NewDecoder(c.Request.Body).Decode(&message); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	err := s.ChatService.UnpinMessage(c, channelID, &message)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	handlers.EncodeSuccess(c, http.StatusNoContent, nil)
}

func (s S) ListPinnedChatMessages(c *gin.Context) {
	param := "kickbackId"
	kickbackID := c.Param(param)
	if kickbackID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	msgs, err := s.ChatService.GetPinnedMessages(c, kickbackID, c.Query("channelId"))
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	messages := []interface{}{}
	for _, msg := range msgs {
		message := utils.Normalize(msg)
		message["sent_by"] = s.UserService.SummarizeUsers(c, []string{msg.SentBy}).Find(msg.SentBy)
		messages = append(messages, message)
	}
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{"pinned_messages": messages})
}
