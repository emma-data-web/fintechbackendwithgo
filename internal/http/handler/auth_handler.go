package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"fintech-system/internal/domain"
	"fintech-system/internal/http/response"
	"fintech-system/internal/service"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

type RegisterRequest struct {
	Name     string      `json:"name" binding:"required,min=2"`
	Email    string      `json:"email" binding:"required,email"`
	Password string      `json:"password" binding:"required,min=6"`
	Address  string      `json:"address"`
	Role     domain.Role `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponseData struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.authService.Register(c.Request.Context(), req.Name, req.Email, req.Password, req.Address, req.Role)
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			response.Error(c, http.StatusConflict, err.Error())
			return
		}
		response.ServerError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "user registered successfully", user)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	token, user, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			response.Error(c, http.StatusUnauthorized, "invalid email or password")
			return
		}
		response.ServerError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "login successful", AuthResponseData{
		Token: token,
		User:  user,
	})
}
