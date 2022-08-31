package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kickback-app/api/logger"
	"github.com/kickback-app/api/pkg/models"
	"github.com/kickback-app/api/server/handlers"
	"github.com/kickback-app/api/utils"
	expo "github.com/oliveroneill/exponent-server-sdk-golang/sdk"
	"gopkg.in/mgo.v2/bson"
)

func (s S) GetNotifications(c *gin.Context) {
	res, err := s.NotificationService.GetNotifications(c)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "retrieved %d notifcations", len(res))
	handlers.EncodeSuccess(c, http.StatusOK, res)
}

func (s S) SendNotification(c *gin.Context) {
	var notification models.Notification
	if err := json.NewDecoder(c.Request.Body).Decode(&notification); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	notifcationID, errReport, err := s.doSendNotification(c, notification)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{"notificationId": notifcationID, "errors": errReport})
}

func (s S) doSendNotification(c *gin.Context, notification models.Notification) (string, models.NotificationErrorReport, error) {
	errReport := models.NotificationErrorReport{
		Push: map[string]string{},
		SMS:  map[string]string{},
	}
	if !notification.Validate() {
		return "", errReport, handlers.MalformedBodyError{}
	}
	notifcationID, err := s.NotificationService.SendNotification(c, notification)
	if err != nil {
		return "", errReport, err
	}
	logger.Info(c, "created new notication in db %s", notifcationID)
	users, err := s.UserService.GetUsers(c, bson.M{"_id": bson.M{"$in": notification.To}})
	if err != nil {
		return "", errReport, err
	}
	if utils.ContainsString(notification.Channels, "push") {
		pushTokens := []expo.ExponentPushToken{}
		for _, user := range users {
			token, err := expo.NewExponentPushToken(user.ExpoPushNotificationToken)
			if err != nil {
				errReport.Push[user.ID] = "failed to get expo token"
				continue
			}
			pushTokens = append(pushTokens, token)
		}
		err = s.NotificationService.SendPushNotification(c, &expo.PushMessage{
			To:       pushTokens,
			Title:    notification.Title,
			Body:     notification.Body,
			Data:     notification.Data,
			Sound:    "default",
			Priority: expo.DefaultPriority,
		})
		if err != nil {
			errReport.Push["general"] = fmt.Sprintf("failed to send push: %v", err)
		}
	}
	if utils.ContainsString(notification.Channels, "sms") && notification.SMSmessage != "" {
		for _, user := range users {

			err := s.NotificationService.SendSMS(c, notification.SMSmessage, user.PhoneNumber)
			if err != nil {
				errReport.Push[user.ID] = fmt.Sprintf("failed to send SMS: %v", err)
			}
		}
	}
	logger.Info(c, "notification err report: %+v", errReport)
	return notifcationID, errReport, nil
}

func (s S) GetUsersNotificationSettings(c *gin.Context) {
	userID := utils.CurrentUser(c).ID
	settings, err := s.UserService.GetNotificationSettings(c)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	allEventsMuted := true
	allGroupsMuted := true
	userEvents, err := s.EventService.GetUsersEvents(c, userID, &models.GetFilters{})
	if err != nil {
		logger.Error(c, "unable to get users events: %v", err)
		handlers.EncodeError(c, err)
		return
	}
	eventNotifcationSettings := []interface{}{}
	for _, eventSummary := range userEvents {
		muted := utils.ContainsString(settings.NotificationSettings.MutedEvents, eventSummary.ID)
		if !muted {
			allEventsMuted = false
		}
		additionalInfo := models.M{"muted": muted}
		eventNotifcationSettings = append(eventNotifcationSettings, eventSummary.ToMap(additionalInfo))
	}
	groupNotifcationSettings := []interface{}{} // groups are not yet implemented
	logger.Info(c, "successfully retrieved settings")
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{
		"mute_all_notifications": settings.NotificationSettings.MuteAllNotifications,
		"mute_all_events":        allEventsMuted,
		"mute_all_groups":        allGroupsMuted,
		"events":                 eventNotifcationSettings,
		"groups":                 groupNotifcationSettings,
	})
}

func (s S) UpdateUsersNotificationSettings(c *gin.Context) {
	var updates models.UserNotificationSettingUpdates
	if err := json.NewDecoder(c.Request.Body).Decode(&updates); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	err := s.UserService.UpdateNotificationSettings(c, &updates)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	handlers.EncodeSuccess(c, http.StatusNoContent, gin.H{})
}
