package gofyr

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
)

// MFASetupResponse returns the enrollment payload.
type MFASetupResponse struct {
	Secret          string   `json:"secret"`
	ProvisioningURI string   `json:"provisioning_uri"`
	QRCodeBase64    string   `json:"qr_code_base64"`
	RecoveryCodes   []string `json:"recovery_codes"`
}

// MFAVerifyRequest defines the verification payload.
type MFAVerifyRequest struct {
	PreAuthToken string `json:"pre_auth_token" binding:"required"`
	Code         string `json:"code" binding:"required"`
}

// handleMFASetup generates TOTP credentials and single-use recovery codes.
func (a *Admin) handleMFASetup(c *gin.Context) {
	// Require pre-auth token
	preAuthToken := extractPreAuthToken(c)
	if preAuthToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Pre-authentication token required"})
		return
	}

	claims, err := a.ParsePreAuthToken(preAuthToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired pre-auth token"})
		return
	}

	primaryDB := a.PrimaryDB()
	if primaryDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Primary datasource unavailable"})
		return
	}

	var user GofyrAdminUser
	if err := primaryDB.First(&user, claims.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Generate TOTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      a.cfg.Title,
		AccountName: user.Email,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate TOTP key: " + err.Error()})
		return
	}

	secret := key.Secret()
	provisioningURI := key.URL()

	// Generate QR code PNG
	qrPNG, err := qrcode.Encode(provisioningURI, qrcode.Medium, 256)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate QR code: " + err.Error()})
		return
	}
	qrCodeBase64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(qrPNG)

	// Generate 8 recovery backup codes (e.g. "8A3F-44D1")
	var rawCodes []string
	// Remove old unused codes for this user
	primaryDB.Where("user_id = ?", user.ID).Delete(&GofyrMFARecoveryCode{})

	for i := 0; i < 8; i++ {
		code := generateRecoveryCode()
		rawCodes = append(rawCodes, code)

		codeHash, err := bcrypt.GenerateFromPassword([]byte(normalizeRecoveryCode(code)), bcrypt.DefaultCost)
		if err == nil {
			rec := GofyrMFARecoveryCode{
				UserID:    user.ID,
				CodeHash:  string(codeHash),
				Used:      false,
				CreatedAt: time.Now(),
			}
			primaryDB.Create(&rec)
		}
	}

	// Update user's MFA secret (not yet enabled until first successful verification)
	user.MFASecret = secret
	primaryDB.Save(&user)

	a.LogAudit(user.ID, user.Email, "auth_mfa_setup_initiated", "", "", "", c.ClientIP(), "MFA enrollment initiated")

	c.JSON(http.StatusOK, MFASetupResponse{
		Secret:          secret,
		ProvisioningURI: provisioningURI,
		QRCodeBase64:    qrCodeBase64,
		RecoveryCodes:   rawCodes,
	})
}

// handleMFAVerify verifies a 6-digit TOTP code or recovery code.
func (a *Admin) handleMFAVerify(c *gin.Context) {
	var req MFAVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pre_auth_token and code are required"})
		return
	}

	claims, err := a.ParsePreAuthToken(req.PreAuthToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired pre-auth token"})
		return
	}

	primaryDB := a.PrimaryDB()
	if primaryDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Primary datasource unavailable"})
		return
	}

	var user GofyrAdminUser
	if err := primaryDB.First(&user, claims.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	cleanCode := strings.TrimSpace(req.Code)
	isValid := false
	usedRecoveryCode := false

	// 1. Try TOTP code verification
	if user.MFASecret != "" && len(cleanCode) == 6 {
		isValid = totp.Validate(cleanCode, user.MFASecret)
	}

	// 2. If not valid TOTP, check single-use recovery codes
	if !isValid {
		var recoveryCodes []GofyrMFARecoveryCode
		if err := primaryDB.Where("user_id = ? AND used = ?", user.ID, false).Find(&recoveryCodes).Error; err == nil {
			normCode := normalizeRecoveryCode(cleanCode)
			for _, rc := range recoveryCodes {
				if bcrypt.CompareHashAndPassword([]byte(rc.CodeHash), []byte(normCode)) == nil {
					isValid = true
					usedRecoveryCode = true
					now := time.Now()
					rc.Used = true
					rc.UsedAt = &now
					primaryDB.Save(&rc)
					break
				}
			}
		}
	}

	if !isValid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication code or recovery code"})
		return
	}

	// Enable MFA if not already enabled
	if !user.MFAEnabled {
		user.MFAEnabled = true
		primaryDB.Save(&user)
	}

	// Issue full session token
	sessionToken, err := a.GenerateSessionToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session token"})
		return
	}

	action := "auth_login_mfa_success"
	if usedRecoveryCode {
		action = "auth_login_recovery_code_used"
	}
	a.LogAudit(user.ID, user.Email, action, "", "", "", c.ClientIP(), "MFA verification successful")

	c.JSON(http.StatusOK, gin.H{
		"token": sessionToken,
		"user": gin.H{
			"id":          user.ID,
			"email":       user.Email,
			"name":        user.Name,
			"role":        user.Role,
			"mfa_enabled": user.MFAEnabled,
		},
	})
}

func extractPreAuthToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	var body struct {
		PreAuthToken string `json:"pre_auth_token"`
	}
	_ = c.ShouldBindBodyWithJSON(&body)
	if body.PreAuthToken != "" {
		return body.PreAuthToken
	}
	return c.Query("pre_auth_token")
}

func generateRecoveryCode() string {
	const charset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	part1 := make([]byte, 4)
	part2 := make([]byte, 4)
	for i := range part1 {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		part1[i] = charset[num.Int64()]
	}
	for i := range part2 {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		part2[i] = charset[num.Int64()]
	}
	return fmt.Sprintf("%s-%s", string(part1), string(part2))
}

func normalizeRecoveryCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, "-", "")
	code = strings.ReplaceAll(code, " ", "")
	return code
}
