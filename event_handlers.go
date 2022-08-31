package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kickback-app/api/internal/services"
	"github.com/kickback-app/api/logger"
	"github.com/kickback-app/api/pkg/models"
	"github.com/kickback-app/api/server/handlers"
	"github.com/kickback-app/api/utils"
	"gopkg.in/mgo.v2/bson"
)

func (s S) GetUsersEvents(c *gin.Context) {
	userID := utils.CurrentUser(c).ID
	before := c.Query("before")
	after := c.Query("after")
	filters := models.GetFilters{}
	if b, err := strconv.Atoi(before); err == nil {
		filters.Before = b
	}
	if a, err := strconv.Atoi(after); err == nil {
		filters.After = a
	}
	userEvents, err := s.EventService.GetUsersEvents(c, userID, &filters)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	userEventsWithInfo := []models.M{}
	for _, event := range userEvents {
		mediaID := event.BackgroundImg
		var backgroundImgInfo models.Media
		var err error
		if strings.HasPrefix(mediaID, "MDA_") {
			backgroundImgInfo, err = s.MediaService.GetItem(c, event.ID, mediaID)
			if err != nil {
				logger.Warn(c, "unable to get additional background image for %v-%v: %v", event.ID, mediaID, err)
			}
		}
		userEventsWithInfo = append(userEventsWithInfo, event.ToMap(models.M{
			"background_img_info": backgroundImgInfo,
		}))
	}
	logger.Info(c, "retrieved %d events %s", len(userEvents), userID)
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{"events": userEventsWithInfo})
}

func (s S) GetEvent(c *gin.Context) {
	param := "eventId"
	eventID := c.Param(param)
	if eventID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	event, err := s.EventService.GetEvent(c, eventID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	resolvedEvent := s.EventService.ResolveLinks(c, event)
	mediaID := event.BackgroundImg
	if strings.HasPrefix(mediaID, "MDA_") {
		backgroundImgInfo, err := s.MediaService.GetItem(c, event.ID, mediaID)
		if err != nil {
			logger.Warn(c, "unable to get additional background image for %v-%v: %v", event.ID, mediaID, err)
		}
		resolvedEvent["background_img_info"] = backgroundImgInfo
	}
	logger.Info(c, "retrieved event %s", eventID)
	handlers.EncodeSuccess(c, http.StatusOK, resolvedEvent)
}

func (s S) CreateEvent(c *gin.Context) {
	var event models.Event
	if err := json.NewDecoder(c.Request.Body).Decode(&event); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	createdEvent, err := s.EventService.CreateEvent(c, &event)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	err = s.UserService.AddEventToUser(c, utils.CurrentUser(c).ID, createdEvent.ID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	noteID, err := s.NoteService.CreateNote(c, &models.Note{ParentID: event.ID})
	if err != nil {
		logger.Error(c, "unable to create a new note for the event: %v", err)
		handlers.EncodeError(c, err)
	}
	logger.Info(c, "created new note '%s'", noteID)
	memberIDs := []string{}
	for _, member := range createdEvent.Members {
		memberIDs = append(memberIDs, member.UserID)
	}
	mainChannelID, err := s.ChatService.CreateChannel(c, event.ID, &models.Channel{
		Name:     event.Name,
		IsPublic: true,
		Members:  memberIDs,
	})
	if err != nil {
		logger.Error(c, "unable to create main channel for the event: %v", err)
		handlers.EncodeError(c, err)
	}
	logger.Info(c, "created main event channel with id: %s", mainChannelID)
	logger.Info(c, "created new event with id: %s", createdEvent.ID)
	handlers.EncodeSuccess(c, http.StatusOK, s.EventService.ResolveLinks(c, createdEvent))
}

func (s S) UpdateEvent(c *gin.Context) {
	eventID := c.Param("eventId")
	if eventID == "" {
		handlers.EncodeError(c, handlers.MissingBodyFieldError{Field: "eventId"})
		return
	}
	var eventUpdates models.EventUpdates
	if err := json.NewDecoder(c.Request.Body).Decode(&eventUpdates); err != nil {
		logger.Error(c, err.Error())
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	err := s.EventService.UpdateEvent(c, eventID, &eventUpdates)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	// retrieve newly updated event to return to caller
	updatedEvent, err := s.EventService.GetEvent(c, eventID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	if eventUpdates.RequiresNotification() {
		d := time.Since(time.Unix(updatedEvent.CreatedAt, 0))
		if d < 1*time.Hour {
			logger.Warn(c, "skipping update notification within first hour: its been %v since created", d.String())
		} else {
			usersToNotify := []string{}
			for _, userID := range updatedEvent.MemberUserIDs() {
				// notify event members excpet for the one making the update
				if userID != utils.CurrentUser(c).ID {
					usersToNotify = append(usersToNotify, userID)
				}
			}
			s.doSendNotification(c, models.Notification{
				Type:     models.EventUpdated,
				Channels: []string{"push"},
				To:       usersToNotify,
				Title:    fmt.Sprintf("Your event %v has been updated", updatedEvent.Name),
				Body:     "check your new event details",
				Data: map[string]string{
					"eventId": eventID,
				},
			})
		}
	}
	logger.Info(c, "successfully updated event %s", eventID)
	handlers.EncodeSuccess(c, http.StatusOK, s.EventService.ResolveLinks(c, updatedEvent))
}

func (s S) DeleteEvent(c *gin.Context) {
	param := "eventId"
	eventID := c.Param(param)
	if eventID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	err := s.EventService.DeleteEvent(c, eventID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	err = s.TaskService.CleanupTasks(c, eventID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	err = s.ExpenseService.Cleanup(c, eventID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	err = s.MediaService.Cleanup(c, eventID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "successfully deleted event %s", eventID)
	handlers.EncodeSuccess(c, http.StatusNoContent, nil)
}

func (s S) UpdateEventSettings(c *gin.Context) {
	param := "eventId"
	eventID := c.Param(param)
	if eventID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	var settingUpdates models.M
	if err := json.NewDecoder(c.Request.Body).Decode(&settingUpdates); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	err := s.EventService.UpdateEventSettings(c, eventID, settingUpdates)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "successfully updated event settings for event %s", eventID)
	handlers.EncodeSuccess(c, http.StatusNoContent, nil)
}

func (s S) GetEventMembers(c *gin.Context) {
	param := "eventId"
	eventID := c.Param(param)
	if eventID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	event, err := s.EventService.GetEvent(c, eventID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "retrieved %d members for event %s", len(event.Members), eventID)
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{"members": s.EventService.EventMembersList(c, event)})
}

func (s S) InviteEventMembers(c *gin.Context) {
	param := "eventId"
	eventID := c.Param(param)
	if eventID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	var body struct {
		Users []struct {
			IsHost bool   `json:"is_host"`
			UserID string `json:"userId"`
		} `json:"users"`
		NewUsers []struct {
			IsHost      bool   `json:"is_host"`
			PhoneNumber string `json:"phone_number"`
			FirstName   string `json:"first_name"`
			LastName    string `json:"last_name"`
		} `json:"new_users"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	// set invited by to the user making the request
	currentUser := utils.CurrentUser(c).ID
	invitesToSend := []models.Member{}
	failures := models.M{}
	hosts := []string{}

	for _, newUser := range body.NewUsers {
		newUserID, err := s.UserService.CreateUser(c, &models.User{
			PhoneNumber: newUser.PhoneNumber,
			FirstName:   newUser.FirstName,
			LastName:    newUser.LastName,
		})
		if err != nil {
			if e, ok := err.(services.UserExistsError); ok {
				// phone number is already associated with an account
				invitesToSend = append(invitesToSend, models.Member{
					UserID:    e.ExistingUserID,
					Status:    models.MemberStatusInvited,
					InvitedBy: currentUser,
				})
				if newUser.IsHost {
					hosts = append(hosts, e.ExistingUserID)
				}
				continue
			}
			failures[newUser.PhoneNumber] = fmt.Sprintf("Failed: %v", err)
		}
		invitesToSend = append(invitesToSend, models.Member{
			UserID:    newUserID,
			Status:    models.MemberStatusInvited,
			InvitedBy: currentUser,
		})
		if newUser.IsHost {
			hosts = append(hosts, newUserID)
		}
	}

	for _, user := range body.Users {
		invitesToSend = append(invitesToSend, models.Member{
			UserID:    user.UserID,
			Status:    models.MemberStatusInvited,
			InvitedBy: currentUser,
		})
		if user.IsHost {
			hosts = append(hosts, user.UserID)
		}
	}
	membersAdded, err := s.EventService.Invite(c, eventID, invitesToSend)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	err = s.EventService.AddEventHosts(c, eventID, hosts)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	event, err := s.EventService.GetEvent(c, eventID)
	if err != nil {
		logger.Error(c, "unable to get event details: %v", err)
		handlers.EncodeError(c, err)
		return
	}
	host, err := s.UserService.GetUserByID(c, event.CreatedBy)
	if err != nil {
		logger.Error(c, "unable to get host user details: %v", err)
		handlers.EncodeError(c, err)
		return
	}
	s.doSendNotification(c, models.Notification{
		Type:     models.EventRSVP,
		Channels: []string{"push", "sms"}, // this only triggers a push because SMSMsg attribute is empty
		To:       memberListToUserIDs(membersAdded),
		Title:    "You've been invited to a new Kickback event",
		Body:     fmt.Sprintf("%v invited you to %v", host.Name(), event.Name),
		Data: map[string]string{
			"eventId": eventID,
		},
	})
	invitedUsers, err := s.UserService.GetUsers(c, bson.M{"_id": bson.M{"$in": memberListToUserIDs(membersAdded)}})
	if err != nil {
		logger.Error(c, "unable to get invited users info: %v", err)
		handlers.EncodeError(c, err)
		return
	}
	// need to use sms method directly because we need to make each invite personalized by user phone number in the embedded link
	for _, invited := range invitedUsers {
		msg := fmt.Sprintf("%v has invited you to %v\n\n"+
			"Description: %v\n\n"+
			"See event details and RSVP: [Link to app]\n\n"+
			"https://kickbackapp.io/invited?eventId=%v&userId=%v",
			host.Name(), event.Name, event.Description, eventID, invited.ID)
		if invited.PhoneNumber != "" {
			s.NotificationService.SendSMS(c, msg, invited.PhoneNumber)
		}
	}
	err = s.UserService.Connect(c, event.MemberUserIDs())
	if err != nil {
		logger.Error(c, "unable to add user connections: %v", err)
		handlers.EncodeError(c, err)
	}
	logger.Info(c, "successfully added %d members to event %s", len(membersAdded), eventID)
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{
		"failures":     failures,
		"invites_sent": len(membersAdded),
	})
}

func (s S) RSVP(c *gin.Context) {
	param := "eventId"
	eventID := c.Param(param)
	if eventID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	source := c.Query("source")
	var rsvp struct {
		// if empty, it will be the authenticated user
		// this allows the front-end web app to make unathenticated calls to rsvp a user
		UserID string              `json:"userId"`
		Status models.MemberStatus `json:"status"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&rsvp); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	currentUserID := utils.CurrentUser(c).ID
	if currentUserID == "" && rsvp.UserID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	userID := rsvp.UserID
	if userID == "" {
		userID = currentUserID
	}
	err := s.EventService.RSVP(c, eventID, userID, rsvp.Status)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	event, err := s.EventService.GetEvent(c, eventID)
	if err != nil {
		logger.Error(c, "unable to get event details: %v", err)
		handlers.EncodeError(c, err)
		return
	}
	user, err := s.UserService.GetUserByID(c, userID)
	if err != nil {
		logger.Error(c, "unable to get user: %v", err)
		handlers.EncodeError(c, err)
		return
	}
	if rsvp.Status == models.MemberStatusGoing {
		// if rsvp'ing from web, trigger text
		if source == "web" {
			msg := fmt.Sprintf("Thank you for RSVPing %s."+
				" The other attendees have been added to your Kickback contacts and you can invite them to events going forward."+
				" To host your own event, get our app", event.Name)
			s.doSendNotification(c, models.Notification{
				Type:     models.EventMemberAttending,
				Channels: []string{"sms"},
				To:       []string{user.ID},
				Title:    "",
				Body:     "",
				Data: map[string]string{
					"eventId": eventID,
				},
				SMSmessage: msg,
			})
		}
		// if is host, send push notification
		if utils.ContainsString(event.Hosts, user.ID) {
			s.doSendNotification(c, models.Notification{
				Type:     models.EventCoHost,
				Channels: []string{"push"},
				To:       []string{user.ID},
				Title:    "You've been been made a co-host",
				Body:     fmt.Sprintf("You are now a cohost of %v", event.Name),
				Data: map[string]string{
					"eventId": eventID,
				},
			})
		}
	}
	// let creator of event know someone rsvp'ed
	s.doSendNotification(c, models.Notification{
		Type:     models.EventRSVP,
		Channels: []string{"push"},
		To:       []string{event.CreatedBy},
		Title:    "Someone RSVPed to your Kickback event",
		Body:     fmt.Sprintf("%v set their status as %v for your event %v", user.Name(), rsvp.Status, event.Name),
		Data: map[string]string{
			"eventId": eventID,
		},
	})
	logger.Info(c, "set member %s status to %s for event %s", userID, rsvp.Status, eventID)
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{userID: rsvp.Status})
}

func (s S) GetEventSettings(c *gin.Context) {
	param := "eventId"
	eventID := c.Param(param)
	if eventID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	settings, err := s.EventService.GetEventSettings(c, eventID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	channels, err := s.ChatService.GetChannels(c, eventID)
	if err != nil {
		handlers.EncodeError(c, fmt.Errorf("unable to list chat channels: %v", err))
		return
	}
	totalChannels := []models.M{}
	for _, channel := range channels {
		totalChannels = append(totalChannels, models.M{
			"channelId": channel.ID,
			"name":      channel.Name,
			"is_open":   utils.ContainsString(settings.Chat.OpenChannels, channel.ID),
			"is_muted":  utils.ContainsString(settings.Chat.MutedChannels, channel.ID),
		})
	}
	settingsAsMap := utils.Normalize(settings)
	chatSettings := settings.Chat
	chatSettingsAsMap := utils.Normalize(chatSettings)
	chatSettingsAsMap["channels"] = totalChannels
	settingsAsMap["chats"] = chatSettingsAsMap
	logger.Info(c, "retrieved settings for event %s", eventID)
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{"settings": settingsAsMap})
}

func memberListToUserIDs(members []models.Member) []string {
	userIDs := []string{}
	for _, m := range members {
		userIDs = append(userIDs, m.UserID)
	}
	return userIDs
}
