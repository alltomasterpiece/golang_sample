package server

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kickback-app/api/server/handlers"
)

func (s S) SendOTP(c *gin.Context) {
	var reqBody struct {
		PhoneNumber string `json:"phoneNumber"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "malformed request body"})
		return
	}
	if reqBody.PhoneNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phoneNumber must be a valid phone number"})
		return
	}
	err := s.IdentityVerificationService.SendOTP(c, reqBody.PhoneNumber)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	c.Status(http.StatusCreated)
}

func (s S) VerifyOTP(c *gin.Context) {
	var reqBody struct {
		PhoneNumber string `json:"phoneNumber"`
		Code        string `json:"code"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "malformed request body"})
		return
	}
	verificationReport, err := s.IdentityVerificationService.VerifyOTP(c, reqBody.PhoneNumber, reqBody.Code)
	if err != nil {
		handlers.EncodeError(c, err)
		return
	}
	c.JSON(http.StatusOK, verificationReport)
}
