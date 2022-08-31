package handlers

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kickback-app/api/logger"
)

// todo: turn into internal package?

type APIResponse struct {
	Result interface{} `json:"result,omitempty"`
	Meta   meta        `json:"meta"`
}

type meta struct {
	RequestID  string      `json:"requestId"`
	HTTPStatus string      `json:"httpStatus"`
	Error      *errorState `json:"error,omitempty"`
}

type errorState struct {
	Message string `json:"errorMessage"`
	Code    string `json:"errorCode"`
}

type APIError interface {
	error
	Code() int
}

func EncodeSuccess(c *gin.Context, statusCode int, result interface{}) {
	resp := APIResponse{
		Result: result,
		Meta: meta{
			RequestID:  c.Writer.Header().Get("X-Request-ID"),
			HTTPStatus: http.StatusText(statusCode),
			Error:      nil,
		},
	}
	c.JSON(statusCode, resp)
}

func EncodeError(c *gin.Context, err error) {
	if e, ok := err.(APIError); ok {
		logger.Error(c, "returning handled error: %s (code: %d)", err.Error(), e.Code())
		resp := APIResponse{
			Result: nil,
			Meta: meta{
				RequestID:  c.Writer.Header().Get("X-Request-ID"),
				HTTPStatus: http.StatusText(e.Code()),
				Error: &errorState{
					Message: e.Error(),
				},
			},
		}
		c.JSON(e.Code(), resp)
		return
	}
	logger.Error(c, "unhandled error: %s", err.Error())
	resp := APIResponse{
		Result: nil,
		Meta: meta{
			RequestID:  c.Writer.Header().Get("X-Request-ID"),
			HTTPStatus: http.StatusText(http.StatusInternalServerError),
			Error: &errorState{
				Message: "internal service error",
			},
		},
	}
	c.JSON(http.StatusInternalServerError, resp)
}

func Healthcheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// https://www.twilio.com/docs/sms/send-messages
// this is the StatusCallbackURL
func SentSMSStatus(c *gin.Context) {
	bodyReader, err := c.Request.GetBody()
	if err != nil {
		logger.Error(c, "unable to read twilio message status: %v", err)
		EncodeError(c, err)
		return
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(bodyReader)
	logger.Info(c, "twilio sms status response: %v", buf.String())
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type MissingPathParamError struct {
	Param string
}

func (e MissingPathParamError) Error() string {
	return fmt.Sprintf("missing required path parameter '%s'", e.Param)
}

func (e MissingPathParamError) Code() int {
	return http.StatusBadRequest
}

type MissingBodyFieldError struct {
	Field string
}

func (e MissingBodyFieldError) Error() string {
	return fmt.Sprintf("missing required body field '%s'", e.Field)
}

func (e MissingBodyFieldError) Code() int {
	return http.StatusBadRequest
}

type MalformedBodyError struct {
	Field string
}

func (e MalformedBodyError) Error() string {
	return "malformed request body"
}

func (e MalformedBodyError) Code() int {
	return http.StatusBadRequest
}
