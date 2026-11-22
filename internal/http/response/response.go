package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func Success(c *gin.Context, status int, message string, data any) {
	c.JSON(status, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Error(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, APIResponse{
		Success: false,
		Error:   message,
	})
}

func ServerError(c *gin.Context, err error) {
	c.AbortWithStatusJSON(http.StatusInternalServerError, APIResponse{
		Success: false,
		Error:   "an unexpected error occurred",
	})
}
