package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kickback-app/api/internal/services"
	"github.com/kickback-app/api/logger"
	"github.com/kickback-app/api/pkg/models"
	"github.com/kickback-app/api/server/handlers"
	"github.com/kickback-app/api/utils"
)

func (s S) GetUser(c *gin.Context) {
	param := "userId"
	userID := c.Param(param)
	if userID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	user, err := s.UserService.GetUserByID(c, userID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "successfully retrieved user %s", userID)
	handlers.EncodeSuccess(c, http.StatusOK, user)
}

func (s S) CreateUser(c *gin.Context) {
	var user models.User
	if err := json.NewDecoder(c.Request.Body).Decode(&user); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	userID, err := s.UserService.CreateUser(c, &user)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "created user %s", userID)
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{"userId": userID})
}

func (s S) UpdateUser(c *gin.Context) {
	userID := utils.CurrentUser(c).ID
	var updates models.UserUpdates
	if err := json.NewDecoder(c.Request.Body).Decode(&updates); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	err := s.UserService.UpdateUser(c, userID, &updates)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "successfully updated user %s", userID)
	handlers.EncodeSuccess(c, http.StatusNoContent, nil)
}

func (s S) DeleteUser(c *gin.Context) {
	userID := utils.CurrentUser(c).ID
	err := s.UserService.DeleteUser(c, userID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "successfully deleted user %s", userID)
	handlers.EncodeSuccess(c, http.StatusNoContent, nil)
}

func (s S) GetUsersConnections(c *gin.Context) {
	user, err := s.UserService.GetUserByID(c, utils.CurrentUser(c).ID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	connections := s.UserService.SummarizeUsers(c, user.Connections)
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{"connections": connections.Summaries})
}

// @todo should this live in the users_handlers or somewhere else?
func (s S) GetSponsoredUsers(c *gin.Context) {
	kickbackPowerUsers, err := s.UserService.GetSponsoredUsers(c)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{"users": kickbackPowerUsers})
}

// beef this up to support direct user searches + more fuzzy searches like "david"
// ensure to add filter to query {"is_public": true}
func (s S) SearchUsers(c *gin.Context) {
	var usersToFind models.UserSearch
	if err := json.NewDecoder(c.Request.Body).Decode(&usersToFind); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	searchOutput, err := s.UserService.FindUsers(c, &usersToFind)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{"users": searchOutput})
}

func (s S) InviteUser(c *gin.Context) {
	var invite struct {
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		PhoneNumber  string `json:"phone_number"`
		Email        string `json:"email"`
		SourceName   string `json:"source_name"`
		Description  string `json:"description"`
		KickbackName string `json:"kickback_name"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&invite); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	newUser := models.User{
		FirstName:     invite.FirstName,
		LastName:      invite.LastName,
		PhoneNumber:   invite.PhoneNumber,
		Email:         invite.Email,
		InvitedBy:     utils.CurrentUser(c).ID,
		AccountStatus: models.UnverifiedStatus,
	}
	// create the user
	ID, err := s.UserService.CreateUser(c, &newUser)
	alreadyCreated := false
	if err != nil {
		logger.Error(c, "unable to create new unverified user: %v", err)
		if _, ok := err.(services.UserExistsError); ok {
			alreadyCreated = true
		} else {
			handlers.EncodeError(c, err)
			return
		}
	}
	if alreadyCreated {
		exisitngUser, err := s.UserService.GetUserByPhoneNumber(c, invite.PhoneNumber)
		if err != nil {
			logger.Error(c, "cant get already existing user: %v", err)
			handlers.EncodeError(c, err)
			return
		}
		ID = exisitngUser.ID
	}
	logger.Info(c, "created new user with id %s", ID)
	// send notification
	msg := fmt.Sprintf("%s invited you to %s\n\nDescription: %s\nView in app: <link>\nView on web: <link>", invite.SourceName, invite.KickbackName, invite.Description)
	err = s.NotificationService.SendSMS(c, msg, invite.PhoneNumber)
	sentSMS := true
	if err != nil {
		logger.Error(c, "unable to send invite: %v", err)
		sentSMS = false
	}
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{
		"userId":         ID,
		"sentSMS":        sentSMS,
		"alreadyCreated": alreadyCreated,
	})
}
