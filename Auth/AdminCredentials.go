package Auth

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const adminCredentialsFile = "admin_credentials.json"

type adminCredentials struct {
	Username     string `json:"username"`
	PasswordHash string `json:"passwordHash"`
	Password     string `json:"password,omitempty"`
}

type AdminPasswordUpdateForm struct {
	Username        string `json:"username"`
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

var adminCredentialsMu sync.Mutex

func defaultAdminCredentials() adminCredentials {
	username := os.Getenv("ADMIN_USERNAME")
	if username == "" {
		username = "admin"
	}

	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "admin123"
	}

	return adminCredentials{
		Username: username,
		Password: password,
	}
}

func readAdminCredentials() adminCredentials {
	credentials := defaultAdminCredentials()

	data, err := os.ReadFile(adminCredentialsFile)
	if err != nil {
		return credentials
	}

	storedCredentials := adminCredentials{}
	if err := json.Unmarshal(data, &storedCredentials); err != nil {
		return credentials
	}

	if strings.TrimSpace(storedCredentials.Username) != "" {
		credentials.Username = strings.TrimSpace(storedCredentials.Username)
	}
	if storedCredentials.PasswordHash != "" {
		credentials.PasswordHash = storedCredentials.PasswordHash
		credentials.Password = ""
	} else if storedCredentials.Password != "" {
		credentials.Password = storedCredentials.Password
	}

	return credentials
}

func writeAdminCredentials(credentials adminCredentials) error {
	data, err := json.MarshalIndent(credentials, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(adminCredentialsFile, data, 0600)
}

func isAdminPasswordMatch(credentials adminCredentials, password string) bool {
	if credentials.PasswordHash != "" {
		return bcrypt.CompareHashAndPassword(
			[]byte(credentials.PasswordHash),
			[]byte(password),
		) == nil
	}

	return subtleCompare(credentials.Password, password)
}

func subtleCompare(a string, b string) bool {
	if len(a) != len(b) {
		return false
	}

	result := 0
	for i := 0; i < len(a); i++ {
		result |= int(a[i] ^ b[i])
	}

	return result == 0
}

func isAdminCredentialMatch(username string, password string) bool {
	adminCredentialsMu.Lock()
	credentials := readAdminCredentials()
	adminCredentialsMu.Unlock()

	return subtleCompare(username, credentials.Username) &&
		isAdminPasswordMatch(credentials, password)
}

func PatchAdminCredentials(ctx *gin.Context) {
	if !isAdminSession(ctx) {
		ctx.String(http.StatusUnauthorized, "you're not allowed to access that, please login first")
		return
	}

	form := &AdminPasswordUpdateForm{}
	if err := ctx.BindJSON(form); err != nil {
		ctx.String(http.StatusBadRequest, "invalid admin account update request")
		return
	}

	username := strings.TrimSpace(form.Username)
	if username == "" || strings.TrimSpace(form.CurrentPassword) == "" || strings.TrimSpace(form.NewPassword) == "" {
		ctx.String(http.StatusUnprocessableEntity, "username, current password, and new password are required")
		return
	}

	if len(form.NewPassword) < 8 {
		ctx.String(http.StatusUnprocessableEntity, "password length should be equal or above 8 characters")
		return
	}

	adminCredentialsMu.Lock()
	defer adminCredentialsMu.Unlock()

	credentials := readAdminCredentials()
	if !isAdminPasswordMatch(credentials, form.CurrentPassword) {
		ctx.String(http.StatusUnauthorized, "current password is incorrect")
		return
	}

	passwordHash, hashErr := bcrypt.GenerateFromPassword([]byte(form.NewPassword), bcrypt.DefaultCost)
	if hashErr != nil {
		ctx.String(http.StatusInternalServerError, "we're unable to process your request right now")
		return
	}

	credentials.Username = username
	credentials.PasswordHash = string(passwordHash)
	credentials.Password = ""

	if err := writeAdminCredentials(credentials); err != nil {
		ctx.String(http.StatusInternalServerError, "we're unable to update the admin account right now")
		return
	}

	ctx.String(http.StatusOK, "admin account updated successfully")
}

func GetAdminAccount(ctx *gin.Context) {
	if !isAdminSession(ctx) {
		ctx.String(http.StatusUnauthorized, "you're not allowed to access that, please login first")
		return
	}

	adminCredentialsMu.Lock()
	credentials := readAdminCredentials()
	adminCredentialsMu.Unlock()

	ctx.JSON(http.StatusOK, gin.H{
		"username": credentials.Username,
	})
}
