package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"fintech-system/internal/domain"
	"fintech-system/internal/http/middleware"
	"fintech-system/internal/http/response"
	"fintech-system/internal/service"
)

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.CtxUserID)
	if !exists {
		response.Error(c, http.StatusUnauthorized, "unauthenticated")
		return
	}

	userID := userIDVal.(string)
	user, err := h.userService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			response.Error(c, http.StatusNotFound, "user profile not found")
			return
		}
		response.ServerError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "profile fetched successfully", user)
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	users, err := h.userService.ListUsers(c.Request.Context(), page, limit)
	if err != nil {
		response.ServerError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "users listed successfully", gin.H{
		"page":  page,
		"limit": limit,
		"users": users,
	})
}
