package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kickback-app/api/server/handlers"
	"github.com/kickback-app/api/server/middlewares"
)

func (s S) AttachRoutes() {
	app := s.app
	app.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome to Kickback")
	})
	app.GET("/healthcheck", handlers.Healthcheck)
	// needs to be unauthorized bc its being called from twilio
	// https://www.twilio.com/docs/sms/send-messages#monitor-the-status-of-your-message
	app.POST("/sms-status", handlers.SentSMSStatus)

	// OTP Verification APIs  -- @todo should these be unauthorized?
	app.POST("/otp/send", s.SendOTP)
	app.POST("/otp/verify", s.VerifyOTP)

	// expose unauthorized endpoints for web app to interact with
	// @todo do manual validation
	app.GET("/events/:eventId", s.GetEvent)
	app.PUT("/events/:eventId/rsvp", s.RSVP)

	v1 := app.Group("/v1")
	v1.Use(middlewares.Authorize)
	{
		// Events APIs
		v1.GET("/events", s.GetUsersEvents)
		v1.GET("/events/:eventId", s.GetEvent)
		v1.POST("/events", s.CreateEvent)
		v1.PUT("/events/:eventId", s.UpdateEvent)
		v1.DELETE("/events/:eventId", s.DeleteEvent)
		// event settings
		v1.GET("/events/:eventId/settings", s.GetEventSettings)
		v1.PUT("/events/:eventId/settings", s.UpdateEventSettings)
		// event memership
		v1.GET("/events/:eventId/members", s.GetEventMembers)
		v1.POST("/events/:eventId/members", s.InviteEventMembers)
		v1.PUT("/events/:eventId/rsvp", s.RSVP)

		// Tasks APIs
		v1.POST("/kickbacks/:kickbackId/tasks", s.CreateTask)
		v1.GET("/kickbacks/:kickbackId/tasks", s.GetTasks)
		v1.GET("/tasks/:taskId", s.GetTask)
		v1.PUT("/tasks/:taskId", s.UpdateTask)
		v1.DELETE("/tasks/:taskId", s.DeleteTask)

		// Expenses APIs
		v1.GET("/kickbacks/:kickbackId/expenses", s.GetExpenses)
		v1.POST("/kickbacks/:kickbackId/expenses", s.CreateExpense)
		v1.GET("/expenses/:expenseId", s.GetExpense)
		v1.PUT("/expenses/:expenseId", s.UpdateExpense)
		v1.DELETE("/expenses/:expenseId", s.DeleteExpense)
		v1.PUT("/expenses/:expenseId/assignees", s.UpdateExpenseAssignee)

		// Notes APIs
		v1.GET("notes/:kickbackId", s.GetNote)
		v1.PUT("notes/:kickbackId", s.UpdateNote)

		// Media APIs
		v1.GET("/kickbacks/:kickbackId/media", s.GetMediaItemsMetadata)
		v1.GET("/kickbacks/:kickbackId/media/:itemId", s.GetMediaItem)
		v1.POST("/kickbacks/:kickbackId/media", s.CreateMediaItem)
		v1.PUT("/kickbacks/:kickbackId/media/:itemId/metadata", s.UpdateMediaItemMetadata)
		v1.DELETE("/kickbacks/:kickbackId/media/:itemId", s.DeleteMediaItem)
		v1.POST("/kickbacks/:kickbackId/media/:itemId/comment", s.AddMediaItemComment)
		v1.PUT("/kickbacks/:kickbackId/media/:itemId/comment/:commentId", s.UpdateMediaItemComment)
		v1.DELETE("/kickbacks/:kickbackId/media/:itemId/comment/:commentId", s.DeleteMediaItemComment)

		// Chat APIs
		v1.GET("/kickbacks/:kickbackId/channels", s.GetChannels)
		v1.POST("/kickbacks/:kickbackId/channels", s.CreateChannel)
		v1.PUT("/chats/:kickbackId/channels/:channelId", s.UpdateChannel)
		v1.DELETE("/chats/:kickbackId/channels/:channelId", s.DeleteChannel)
		v1.PUT("/chats/:kickbackId/channels/:channelId/members", s.UpdateChannelMembers)
		v1.POST("/chats/:kickbackId/channels/:channelId/pinned", s.PinChatMessage)
		v1.DELETE("/chats/:kickbackId/channels/:channelId/pinned", s.UnpinChatMessage)
		v1.GET("/chats/:kickbackId/pinned", s.ListPinnedChatMessages)

		// Notification APIs
		v1.GET("/notifications", s.GetNotifications)
		v1.POST("/notifications", s.SendNotification)
		v1.GET("/notifications/settings", s.GetUsersNotificationSettings)
		v1.PUT("/notifications/settings", s.UpdateUsersNotificationSettings)

		// Users APIs
		v1.GET("/users/:userId", s.GetUser) // @todo need to implment some access controls for users getting each others profiles
		v1.POST("/users", s.CreateUser)
		v1.PUT("/users", s.UpdateUser)
		v1.DELETE("/users", s.DeleteUser)
		v1.GET("/users/connections", s.GetUsersConnections)
		v1.GET("/users/following/default", s.GetSponsoredUsers)
		v1.POST("/users/search", s.SearchUsers)
		v1.POST("/users/invite", s.InviteUser) // invite new user to the platform
	}
}
