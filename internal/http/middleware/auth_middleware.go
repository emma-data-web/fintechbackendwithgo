package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"fintech-system/internal/domain"
	"fintech-system/internal/http/response"
	"fintech-system/pkg/jwt"
)

const (
	CtxUserID = "user_id"
	CtxEmail  = "user_email"
	CtxRole   = "user_role"
)

func RequireAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, http.StatusUnauthorized, "authorization header required")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			response.Error(c, http.StatusUnauthorized, "authorization format must be Bearer <token>")
			return
		}

		claims, err := jwt.ValidateToken(parts[1], jwtSecret)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxEmail, claims.Email)
		c.Set(CtxRole, claims.Role)

		c.Next()
	}
}

func RequireRole(allowedRoles ...domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get(CtxRole)
		if !exists {
			response.Error(c, http.StatusUnauthorized, "unauthenticated")
			return
		}

		userRole := domain.Role(roleVal.(string))
		for _, allowed := range allowedRoles {
			if userRole == allowed {
				c.Next()
				return
			}
		}

		response.Error(c, http.StatusForbidden, "you do not have permission to perform this action")
	}
}
