package Auth

import (
	"encoding/json"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	AdminResource "github.com/mrdcvlsc/scheduling-system-backend/Resources/Admin"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
	"golang.org/x/crypto/bcrypt"
)

const adminCredentialsFile = "admin_credentials.json"

type adminCredentials = AdminResource.AdminCredentials

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

	storedCredentials, persistErr := readAdminCredentialsFromPersistence()
	if persistErr == nil && storedCredentials != nil {
		credentials = mergeAdminCredentials(credentials, *storedCredentials)
		credentials, hashed := ensureAdminPasswordHash(credentials)
		if hashed {
			_ = writeAdminCredentials(credentials)
		}
		return credentials
	}

	legacyCredentials, legacyErr := readAdminCredentialsFromFile()
	if legacyErr == nil && legacyCredentials != nil {
		credentials = mergeAdminCredentials(credentials, *legacyCredentials)
	}

	credentials, hashed := ensureAdminPasswordHash(credentials)
	if persistErr == nil && (storedCredentials == nil || hashed) {
		_ = writeAdminCredentials(credentials)
	}

	return credentials
}

func writeAdminCredentials(credentials adminCredentials) error {
	if credentials.PasswordHash != "" {
		credentials.Password = ""
	}

	if RouteGlobals.ResourcesPersistence == nil || RouteGlobals.ResourcesPersistence.WriterService == nil {
		return writeAdminCredentialsFile(credentials)
	}

	return RouteGlobals.ResourcesPersistence.WriterService.UpsertAdminCredentials(credentials)
}

func readAdminCredentialsFromPersistence() (*adminCredentials, error) {
	if RouteGlobals.ResourcesPersistence == nil || RouteGlobals.ResourcesPersistence.ReaderService == nil {
		return nil, nil
	}

	return RouteGlobals.ResourcesPersistence.ReaderService.ReadAdminCredentials()
}

func readAdminCredentialsFromFile() (*adminCredentials, error) {
	projectRoot, errProjectRoot := Utils.FindProjectRoot()
	if errProjectRoot != nil {
		return nil, errProjectRoot
	}

	credentialsFile := path.Join(projectRoot, adminCredentialsFile)
	data, err := os.ReadFile(credentialsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	storedCredentials := adminCredentials{}
	if err := json.Unmarshal(data, &storedCredentials); err != nil {
		return nil, err
	}

	return &storedCredentials, nil
}

func writeAdminCredentialsFile(credentials adminCredentials) error {
	projectRoot, errProjectRoot := Utils.FindProjectRoot()
	if errProjectRoot != nil {
		return errProjectRoot
	}

	credentialsFile := path.Join(projectRoot, adminCredentialsFile)
	data, err := json.MarshalIndent(credentials, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(credentialsFile, data, 0600)
}

func mergeAdminCredentials(defaults adminCredentials, stored adminCredentials) adminCredentials {
	if strings.TrimSpace(stored.Username) != "" {
		defaults.Username = strings.TrimSpace(stored.Username)
	}

	if stored.PasswordHash != "" {
		defaults.PasswordHash = stored.PasswordHash
		defaults.Password = ""
	} else if stored.Password != "" {
		defaults.Password = stored.Password
	}

	return defaults
}

func ensureAdminPasswordHash(credentials adminCredentials) (adminCredentials, bool) {
	if credentials.PasswordHash != "" {
		credentials.Password = ""
		return credentials, false
	}

	if strings.TrimSpace(credentials.Password) == "" {
		return credentials, false
	}

	passwordHash, hashErr := bcrypt.GenerateFromPassword([]byte(credentials.Password), bcrypt.DefaultCost)
	if hashErr != nil {
		return credentials, false
	}

	credentials.PasswordHash = string(passwordHash)
	credentials.Password = ""
	return credentials, true
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
