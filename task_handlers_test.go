package server_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kickback-app/api/internal/services"
	"github.com/kickback-app/api/server"
	"github.com/kickback-app/api/utils"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestGetTaskHandler(t *testing.T) {
	cases := []struct {
		Name               string
		Params             []gin.Param
		DBResponse         interface{}
		ExpectedStatusCode int
		PathToResult       string
		ExpectedResult     string
	}{
		{
			Name:   "happy path - can get task",
			Params: []gin.Param{{Key: "taskId", Value: "mockTaskId"}},
			DBResponse: `{
				"_id": "TSK_Mock",
				"name": "mock task",
				"parentId": "mockParentId",
				"is_completed": false,
				"is_private": false,
				"assignees": [],
				"completed_by": "",
				"created_by": "mockUserId",
				"created_at": 333,
				"updated_at":666
			}`,
			ExpectedStatusCode: http.StatusOK,
			PathToResult:       "result",
			ExpectedResult: `{
				"_id": "TSK_Mock",
				"name": "mock task",
				"parentId": "mockParentId",
				"is_completed": false,
				"is_private": false,
				"assignees": [],
				"completed_by": "",
				"created_by": "mockUserId",
				"created_at": 333,
				"updated_at":666
			}`,
		},
		{
			Name:               "no taskId in path throws error",
			Params:             []gin.Param{},
			ExpectedStatusCode: http.StatusBadRequest,
			PathToResult:       "meta.error",
			ExpectedResult: `{
				"errorMessage": "missing required path parameter 'taskId'",
				"errorCode": ""
				}`,
		},
		{
			Name:               "internal caught error",
			Params:             []gin.Param{{Key: "taskId", Value: "mockTaskId"}},
			DBResponse:         utils.MockCaughtError{StatusCode: 861},
			ExpectedStatusCode: 861,
			PathToResult:       "meta.error",
			ExpectedResult: `{
				"errorMessage": "caught err",
				"errorCode": ""
				}`,
		},
		{
			Name:               "internal service error",
			Params:             []gin.Param{{Key: "taskId", Value: "mockTaskId"}},
			DBResponse:         utils.MockUncaughtError{},
			ExpectedStatusCode: http.StatusInternalServerError,
			PathToResult:       "meta.error",
			ExpectedResult: `{
				"errorMessage": "internal service error",
				"errorCode": ""
				}`,
		},
	}
	for i, c := range cases {
		fmt.Printf("executing case %d: %v\n", i, c.Name)
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		mockUserID := "mockUserId"
		ctx.Set("userId", mockUserID)
		ctx.Request = &http.Request{Header: make(http.Header)}
		ctx.Params = c.Params
		callcount := 0
		mockServer := server.S{
			TaskService: services.TaskService{
				DBClient: utils.MockDBClient{
					CallCount:       &callcount,
					DefaultResponse: "default",
					Responses:       []interface{}{c.DBResponse},
				},
			},
		}
		utils.MockRequest(ctx, http.MethodGet, "")
		mockServer.GetTask(ctx)
		assert.EqualValues(t, c.ExpectedStatusCode, w.Code)
		actual := gjson.Get(w.Body.String(), c.PathToResult).String()
		assert.JSONEq(t, actual, c.ExpectedResult)
	}
}

func TestGetTasksHandler(t *testing.T) {
	cases := []struct {
		Name                   string
		Params                 []gin.Param
		TaskServiceResponses   []interface{}
		UserServiceDBResponses []interface{}
		ExpectedStatusCode     int
		PathToResult           string
		ExpectedResult         string
	}{
		{
			Name:   "happy path - can get task",
			Params: []gin.Param{{Key: "kickbackId", Value: "mockKickbackId"}},
			TaskServiceResponses: []interface{}{`[{
				"_id": "TSK_Mock",
				"name": "mock task",
				"parentId": "mockParentId",
				"is_completed": false,
				"is_private": false,
				"assignees": ["mockAssigneeUserId"],
				"completed_by": "",
				"created_by": "mockUserId",
				"created_at": 333,
				"updated_at":666,
				"due_by": 12
			}]`},
			UserServiceDBResponses: []interface{}{
				`[{
					"_id": "mockAssigneeUserId",
					"profile_img_url": "imgA"
				}]`,
				`[{
					"_id": "mockUserId",
					"profile_img_url": "img"
				}]`,
			},
			ExpectedStatusCode: http.StatusOK,
			PathToResult:       "result",
			ExpectedResult: `{"tasks": [{
				"_id": "TSK_Mock",
				"name": "mock task",
				"parentId": "mockParentId",
				"is_completed": false,
				"is_private": false,
				"assignees": [
					{
						"_id":"mockAssigneeUserId", 
						"account_status":"", 
						"first_name":"", 
						"last_name":"",
						"phone_number":"", 
						"profile_img_url":"imgA", 
						"venmo_username":""
					}
				],
				"completed_by": "",
				"created_by": {
					"_id":"mockUserId", 
					"account_status":"", 
					"first_name":"", 
					"last_name":"",
					"phone_number":"", 
					"profile_img_url":"img", 
					"venmo_username":""
				},
				"due_by": 12,
				"created_at": 333,
				"updated_at":666
			}]}`,
		},
		{
			Name:               "no taskId in path throws error",
			Params:             []gin.Param{},
			ExpectedStatusCode: http.StatusBadRequest,
			PathToResult:       "meta.error",
			ExpectedResult: `{
				"errorMessage": "missing required path parameter 'kickbackId'",
				"errorCode": ""
				}`,
		},
		{
			Name:                 "internal caught error",
			Params:               []gin.Param{{Key: "kickbackId", Value: "mockKickbackId"}},
			TaskServiceResponses: []interface{}{utils.MockCaughtError{StatusCode: 861}},
			ExpectedStatusCode:   861,
			PathToResult:         "meta.error",
			ExpectedResult: `{
				"errorMessage": "caught err",
				"errorCode": ""
				}`,
		},
		{
			Name:                 "internal service error",
			Params:               []gin.Param{{Key: "kickbackId", Value: "mockKickbackId"}},
			TaskServiceResponses: []interface{}{utils.MockUncaughtError{}},
			ExpectedStatusCode:   http.StatusInternalServerError,
			PathToResult:         "meta.error",
			ExpectedResult: `{
				"errorMessage": "internal service error",
				"errorCode": ""
				}`,
		},
	}
	for i, c := range cases {
		fmt.Printf("executing case %d: %v\n", i, c.Name)
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		mockUserID := "mockUserId"
		ctx.Set("userId", mockUserID)
		ctx.Request = &http.Request{Header: make(http.Header)}
		ctx.Params = c.Params
		taskserviceCallcount := 0
		userserviceCallcount := 0
		mockServer := server.S{
			TaskService: services.TaskService{
				DBClient: utils.MockDBClient{
					CallCount: &taskserviceCallcount,
					Responses: c.TaskServiceResponses,
				},
			},
			UserService: services.UserService{
				DBClient: utils.MockDBClient{
					CallCount: &userserviceCallcount,
					Responses: c.UserServiceDBResponses,
				},
				Cache: &utils.MockCache{
					Callcount: new(int),
					Items:     map[string]interface{}{},
				},
			},
		}
		utils.MockRequest(ctx, http.MethodGet, "")
		mockServer.GetTasks(ctx)
		assert.EqualValues(t, c.ExpectedStatusCode, w.Code)
		actual := gjson.Get(w.Body.String(), c.PathToResult).String()
		assert.JSONEq(t, actual, c.ExpectedResult)
	}
}

func TestCreateTaskHandler(t *testing.T) {
	cases := []struct {
		Name               string
		Params             []gin.Param
		RequestBody        string
		DBResponse         interface{}
		ExpectedStatusCode int
		PathToResult       string
		ExpectedResult     string
	}{
		{
			Name:               "happy path - can create task",
			Params:             []gin.Param{{Key: "kickbackId", Value: "EVT_mock"}},
			RequestBody:        `{"name": "mock task"}`,
			DBResponse:         "newTaskId",
			ExpectedStatusCode: http.StatusOK,
			PathToResult:       "result",
			ExpectedResult:     `{"taskId": "newTaskId"}`,
		},
		{
			Name:               "missing path param throws error",
			Params:             []gin.Param{},
			ExpectedStatusCode: http.StatusBadRequest,
			PathToResult:       "meta.error",
			ExpectedResult: `{
				"errorMessage": "missing required path parameter 'kickbackId'",
				"errorCode": ""
				}`,
		},
		{
			Name:               "bad body",
			Params:             []gin.Param{{Key: "kickbackId", Value: "EVT_mock"}},
			RequestBody:        "some malformed body",
			ExpectedStatusCode: http.StatusBadRequest,
			PathToResult:       "meta.error",
			ExpectedResult: `{
				"errorMessage": "malformed request body",
				"errorCode": ""
				}`,
		},
		{
			Name:               "internal caught error",
			Params:             []gin.Param{{Key: "kickbackId", Value: "EVT_mock"}},
			RequestBody:        "{}",
			DBResponse:         utils.MockCaughtError{StatusCode: 861},
			ExpectedStatusCode: 861,
			PathToResult:       "meta.error",
			ExpectedResult: `{
				"errorMessage": "caught err",
				"errorCode": ""
				}`,
		},
		{
			Name:               "internal service error",
			Params:             []gin.Param{{Key: "kickbackId", Value: "EVT_mock"}},
			RequestBody:        "{}",
			DBResponse:         utils.MockUncaughtError{},
			ExpectedStatusCode: http.StatusInternalServerError,
			PathToResult:       "meta.error",
			ExpectedResult: `{
				"errorMessage": "internal service error",
				"errorCode": ""
				}`,
		},
	}
	for i, c := range cases {
		fmt.Printf("executing case %d: %v\n", i, c.Name)
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		mockUserID := "mockUserId"
		ctx.Set("userId", mockUserID)
		ctx.Request = &http.Request{Header: make(http.Header)}
		ctx.Params = c.Params
		callcount := 0
		mockServer := server.S{
			TaskService: services.TaskService{
				DBClient: utils.MockDBClient{
					CallCount:       &callcount,
					DefaultResponse: "default",
					Responses:       []interface{}{c.DBResponse},
				},
			},
		}
		utils.MockRequest(ctx, http.MethodPost, c.RequestBody)
		mockServer.CreateTask(ctx)
		assert.EqualValues(t, c.ExpectedStatusCode, w.Code)
		actual := gjson.Get(w.Body.String(), c.PathToResult).String()
		assert.JSONEq(t, actual, c.ExpectedResult)
	}
}

func TestUpdateTaskHandler(t *testing.T) {
	cases := []struct {
		Name               string
		Params             []gin.Param
		RequestBody        string
		DBResponse         interface{}
		ExpectedStatusCode int
		PathToResult       string
		ExpectedResult     string
	}{
		{
			Name:               "happy path - can update task",
			Params:             []gin.Param{{Key: "taskId", Value: "mockTaskId"}},
			RequestBody:        `{"name": "mock task name update"}`,
			DBResponse:         "newTaskId",
			ExpectedStatusCode: http.StatusNoContent,
			PathToResult:       "",
			ExpectedResult:     "",
		},
		{
			Name:               "missing path param throws error",
			Params:             []gin.Param{},
			ExpectedStatusCode: http.StatusBadRequest,
			PathToResult:       "meta.error",
			ExpectedResult: `{
				"errorMessage": "missing required path parameter 'taskId'",
				"errorCode": ""
				}`,
		},
		{
			Name:               "bad body",
			Params:             []gin.Param{{Key: "taskId", Value: "mockTaskId"}},
			RequestBody:        "some malformed body",
			ExpectedStatusCode: http.StatusBadRequest,
			PathToResult:       "meta.error",
			ExpectedResult: `{
				"errorMessage": "malformed request body",
				"errorCode": ""
				}`,
		},
		{
			Name:               "internal caught error",
			Params:             []gin.Param{{Key: "taskId", Value: "mockTaskId"}},
			RequestBody:        "{}",
			DBResponse:         utils.MockCaughtError{StatusCode: 861},
			ExpectedStatusCode: 861,
			PathToResult:       "meta.error",
			ExpectedResult: `{
				"errorMessage": "caught err",
				"errorCode": ""
				}`,
		},
		{
			Name:               "internal service error",
			Params:             []gin.Param{{Key: "taskId", Value: "mockTaskId"}},
			RequestBody:        "{}",
			DBResponse:         utils.MockUncaughtError{},
			ExpectedStatusCode: http.StatusInternalServerError,
			PathToResult:       "meta.error",
			ExpectedResult: `{
				"errorMessage": "internal service error",
				"errorCode": ""
				}`,
		},
	}
	for i, c := range cases {
		fmt.Printf("executing case %d: %v\n", i, c.Name)
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		mockUserID := "mockUserId"
		ctx.Set("userId", mockUserID)
		ctx.Request = &http.Request{Header: make(http.Header)}
		ctx.Params = c.Params
		callcount := 0
		mockServer := server.S{
			TaskService: services.TaskService{
				DBClient: utils.MockDBClient{
					CallCount:       &callcount,
					DefaultResponse: 0,
					Responses:       []interface{}{c.DBResponse},
				},
			},
		}
		utils.MockRequest(ctx, http.MethodPost, c.RequestBody)
		mockServer.UpdateTask(ctx)
		assert.EqualValues(t, c.ExpectedStatusCode, w.Code, c.Name)
		if c.ExpectedResult != "" {
			actual := gjson.Get(w.Body.String(), c.PathToResult).String()
			assert.JSONEq(t, actual, c.ExpectedResult, c.Name)
		}
	}
}

func TestDeleteTaskHandler(t *testing.T) {
	cases := []struct {
		Name               string
		Params             []gin.Param
		RequestBody        string
		DBResponse         interface{}
		ExpectedStatusCode int
		PathToResult       string
		ExpectedResult     string
	}{
		{
			Name:               "happy path - can delete task",
			Params:             []gin.Param{{Key: "taskId", Value: "mockTaskId"}},
			RequestBody:        "",
			DBResponse:         int64(1),
			ExpectedStatusCode: http.StatusNoContent,
			PathToResult:       "",
			ExpectedResult:     "",
		},
		{
			Name:               "missing path param throws error",
			Params:             []gin.Param{},
			ExpectedStatusCode: http.StatusBadRequest,
			PathToResult:       "meta.error",
			ExpectedResult: `{
				"errorMessage": "missing required path parameter 'taskId'",
				"errorCode": ""
				}`,
		},
		{
			Name:               "internal caught error",
			Params:             []gin.Param{{Key: "taskId", Value: "mockTaskId"}},
			RequestBody:        "{}",
			DBResponse:         utils.MockCaughtError{StatusCode: 861},
			ExpectedStatusCode: 861,
			PathToResult:       "meta.error",
			ExpectedResult: `{
				"errorMessage": "caught err",
				"errorCode": ""
				}`,
		},
		{
			Name:               "internal service error",
			Params:             []gin.Param{{Key: "taskId", Value: "mockTaskId"}},
			RequestBody:        "{}",
			DBResponse:         utils.MockUncaughtError{},
			ExpectedStatusCode: http.StatusInternalServerError,
			PathToResult:       "meta.error",
			ExpectedResult: `{
				"errorMessage": "internal service error",
				"errorCode": ""
				}`,
		},
	}
	for i, c := range cases {
		fmt.Printf("executing case %d: %v\n", i, c.Name)
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)
		mockUserID := "mockUserId"
		ctx.Set("userId", mockUserID)
		ctx.Request = &http.Request{Header: make(http.Header)}
		ctx.Params = c.Params
		callcount := 0
		mockServer := server.S{
			TaskService: services.TaskService{
				DBClient: utils.MockDBClient{
					CallCount:       &callcount,
					DefaultResponse: 0,
					Responses:       []interface{}{c.DBResponse},
				},
			},
		}
		utils.MockRequest(ctx, http.MethodPost, c.RequestBody)
		mockServer.DeleteTask(ctx)
		assert.EqualValues(t, c.ExpectedStatusCode, w.Code, c.Name)
		if c.ExpectedResult != "" {
			actual := gjson.Get(w.Body.String(), c.PathToResult).String()
			assert.JSONEq(t, actual, c.ExpectedResult, c.Name)
		}
	}
}
