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

func (s S) CreateTask(c *gin.Context) {
	param := "kickbackId"
	kickbackID := c.Param(param)
	if kickbackID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: "kickbackId"})
		return
	}
	var t models.Task
	if err := json.NewDecoder(c.Request.Body).Decode(&t); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	t.ParentID = kickbackID
	taskID, err := s.TaskService.CreateTask(c, &t)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "created new task with id %s within %s", taskID, t.ParentID)
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{"taskId": taskID})
}

func (s S) GetTask(c *gin.Context) {
	param := "taskId"
	taskID := c.Param(param)
	if taskID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	task, err := s.TaskService.GetTask(c, taskID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "retrieved task %s", taskID)
	handlers.EncodeSuccess(c, http.StatusOK, task)
}

func (s S) GetTasks(c *gin.Context) {
	param := "kickbackId"
	kickbackID := c.Param(param)
	if kickbackID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	res, err := s.TaskService.GetTasks(c, kickbackID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	tasksWithUserInfo := []models.M{}
	for _, task := range res {
		assignees := s.UserService.SummarizeUsers(c, task.Assignees).Summaries
		taskAsMap := utils.Normalize(task)
		taskAsMap["assignees"] = assignees
		taskAsMap["created_by"] = s.UserService.SummarizeUsers(c, []string{task.CreatedBy}).Find(task.CreatedBy)
		tasksWithUserInfo = append(tasksWithUserInfo, taskAsMap)
	}
	logger.Info(c, "retrieved %d tasks for kickback %s", len(tasksWithUserInfo), kickbackID)
	handlers.EncodeSuccess(c, http.StatusOK, gin.H{"tasks": tasksWithUserInfo})
}

func (s S) UpdateTask(c *gin.Context) {
	param := "taskId"
	taskID := c.Param(param)
	if taskID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	var updates models.TaskUpdates
	if err := json.NewDecoder(c.Request.Body).Decode(&updates); err != nil {
		handlers.EncodeError(c, handlers.MalformedBodyError{})
		return
	}
	err := s.TaskService.UpdateTask(c, taskID, updates)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "successfully updated task %s", taskID)
	handlers.EncodeSuccess(c, http.StatusNoContent, nil)
}

func (s S) DeleteTask(c *gin.Context) {
	param := "taskId"
	taskID := c.Param(param)
	if taskID == "" {
		handlers.EncodeError(c, handlers.MissingPathParamError{Param: param})
		return
	}
	err := s.TaskService.DeleteTask(c, taskID)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	logger.Info(c, "successfully deleted task %s", taskID)
	handlers.EncodeSuccess(c, http.StatusNoContent, nil)
}
