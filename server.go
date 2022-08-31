package server

import (
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/kickback-app/api/internal/database"
	"github.com/kickback-app/api/internal/services"
	"github.com/kickback-app/api/pkg/models"
	"github.com/patrickmn/go-cache"
)

type S struct {
	app                         *gin.Engine
	IdentityVerificationService models.IdenitityVerifier
	MediaService                models.MediaManager
	ChatService                 models.ChatManager
	NotificationService         models.NotificationManager
	NoteService                 models.NoteManager
	TaskService                 models.TaskManager
	UserService                 models.UserManager
	ExpenseService              models.ExpenseManager
	EventService                models.EventManager
}

func (s S) Engine() *gin.Engine {
	return s.app
}

func New(app *gin.Engine, s3Client *s3.S3, dbClient database.Manager) S {
	userservice := services.UserService{
		Collection: "users",
		DBClient:   dbClient,
		Cache:      cache.New(5*time.Minute, 10*time.Minute),
	}
	return S{
		app: app,
		IdentityVerificationService: services.IdentityVerificationManager{
			Client:      &http.Client{Timeout: 1 * time.Minute},
			UserService: userservice,
		},
		ChatService: services.ChatService{
			Collection: "chats",
			DBClient:   dbClient,
		},
		NotificationService: services.NotificationService{
			Collection: "notifications",
			DBClient:   dbClient,
			HTTPClient: &http.Client{Timeout: 1 * time.Minute},
		},
		NoteService: services.NoteService{
			Collection: "notes",
			DBClient:   dbClient,
		},
		MediaService: services.MediaService{
			Collection:  "media",
			DBClient:    dbClient,
			S3Client:    s3Client,
			UserService: userservice,
		},
		TaskService: services.TaskService{
			Collection: "tasks",
			DBClient:   dbClient,
		},
		ExpenseService: services.ExpenseService{
			Collection:  "expenses",
			DBClient:    dbClient,
			UserService: userservice,
		},
		EventService: services.EventService{
			Collection:  "events",
			DBClient:    dbClient,
			UserService: userservice,
		},
		UserService: userservice,
	}
}
