package gofyr

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// SessionClaims represents the JWT payload for a verified admin session.
type SessionClaims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// PreAuthClaims represents the short-lived JWT payload before MFA verification.
type PreAuthClaims struct {
	UserID  uint   `json:"user_id"`
	Email   string `json:"email"`
	PreAuth bool   `json:"pre_auth"`
	jwt.RegisteredClaims
}

// HashPassword hashes a raw password using bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword compares a bcrypt hash with a raw password.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateSessionToken creates a signed session JWT.
func (a *Admin) GenerateSessionToken(user *GofyrAdminUser) (string, error) {
	claims := SessionClaims{
		UserID: user.ID,
		Email:  user.Email,
		Name:   user.Name,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.cfg.SessionExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "gofyrAdmin",
			Subject:   user.Email,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.cfg.JWTSecret))
}

// GeneratePreAuthToken creates a short-lived pre-auth JWT.
func (a *Admin) GeneratePreAuthToken(user *GofyrAdminUser) (string, error) {
	claims := PreAuthClaims{
		UserID:  user.ID,
		Email:   user.Email,
		PreAuth: true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.cfg.PreAuthExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "gofyrAdmin",
			Subject:   user.Email,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.cfg.JWTSecret))
}

// ParseSessionToken validates and extracts claims from a session JWT.
func (a *Admin) ParseSessionToken(tokenString string) (*SessionClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &SessionClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(a.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*SessionClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid session token")
}

// ParsePreAuthToken validates and extracts claims from a pre-auth JWT.
func (a *Admin) ParsePreAuthToken(tokenString string) (*PreAuthClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &PreAuthClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(a.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*PreAuthClaims); ok && token.Valid && claims.PreAuth {
		return claims, nil
	}
	return nil, errors.New("invalid pre-auth token")
}

// LoginRequest defines credentials submitted to /api/auth/login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// handleLogin handles stage 1 authentication.
func (a *Admin) handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email and password are required"})
		return
	}

	primaryDB := a.PrimaryDB()
	if primaryDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Primary datasource unavailable"})
		return
	}

	var user GofyrAdminUser
	if err := primaryDB.Where("email = ?", strings.ToLower(strings.TrimSpace(req.Email))).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	if !CheckPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	preAuthToken, err := a.GeneratePreAuthToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate pre-auth token"})
		return
	}

	a.LogAudit(user.ID, user.Email, "auth_pre_auth_success", "", "", "", c.ClientIP(), "Stage 1 login successful")

	c.JSON(http.StatusOK, gin.H{
		"pre_auth_token": preAuthToken,
		"mfa_enrolled":   user.MFAEnabled && user.MFASecret != "",
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
	})
}

// handleMe returns current user info.
func (a *Admin) handleMe(c *gin.Context) {
	user := a.CurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

// handleLogout clears session.
func (a *Admin) handleLogout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// AuthMiddleware ensures endpoints are accessed with a valid session token.
func (a *Admin) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !a.cfg.AuthRequired {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// Also check cookie if present
			if cookieToken, err := c.Cookie("gofyr_token"); err == nil && cookieToken != "" {
				authHeader = "Bearer " + cookieToken
			}
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == "" || tokenStr == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication token missing or invalid"})
			c.Abort()
			return
		}

		claims, err := a.ParseSessionToken(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid: " + err.Error()})
			c.Abort()
			return
		}

		c.Set("gofyr_user", claims)
		c.Next()
	}
}

// CurrentUser extracts the authenticated session claims from gin.Context.
func (a *Admin) CurrentUser(c *gin.Context) *SessionClaims {
	if val, ok := c.Get("gofyr_user"); ok {
		if claims, ok := val.(*SessionClaims); ok {
			return claims
		}
	}
	return nil
}
