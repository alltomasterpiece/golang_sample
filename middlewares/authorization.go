package middlewares

import (
	"fmt"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/kickback-app/api/logger"
)

func Authorize(c *gin.Context) {
	tok := c.GetHeader(headerJWTToken)
	if tok == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing JWT header"})
		return
	}
	token, err := jwt.Parse(tok, func(token *jwt.Token) (interface{}, error) {

		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected token type")
		}
		if jwtSigningToken == "" {
			return nil, fmt.Errorf("signing token is empty")
		}
		return []byte(jwtSigningToken), nil
	})
	if err != nil {
		logger.Error(c, "auth err: "+err.Error())
	}
	// @todo uncomment once we finish development with hyperlink
	// if err != nil {
	// 	logger.Error(c, err.Error())
	// 	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "unexpected error validating auth token"})
	// 	return
	// }
	// if !token.Valid {
	// 	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "not authorized"})
	// 	return
	// }
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		iss := claims["iss"].(string)
		aud := claims["aud"].(string)
		userId := claims["userId"].(string)
		c.Set("iss", iss)
		c.Set("aud", aud)
		c.Set("userId", userId)
		logger.Info(c, "jwt info: iss '%s', aud '%s', userId '%s'", iss, aud, userId)
	} else {
		logger.Error(c, "unable to parse JWT claims")
	}
	c.Next()
}
