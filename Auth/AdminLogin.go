package Auth

import (
	"crypto/subtle"
	"net/http"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type AdminLoginForm struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func AdminLogin(ctx *gin.Context) {
	adminLoginForm := &AdminLoginForm{}

	if err := ctx.BindJSON(adminLoginForm); err != nil {
		ctx.String(http.StatusBadRequest, "invalid login request")
		return
	}

	adminUsername := os.Getenv("ADMIN_USERNAME")
	if adminUsername == "" {
		adminUsername = "admin"
	}

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin123"
	}

	isUsernameMatch := subtle.ConstantTimeCompare(
		[]byte(adminLoginForm.Username),
		[]byte(adminUsername),
	) == 1

	isPasswordMatch := subtle.ConstantTimeCompare(
		[]byte(adminLoginForm.Password),
		[]byte(adminPassword),
	) == 1

	if !isUsernameMatch || !isPasswordMatch {
		ctx.String(http.StatusUnauthorized, "invalid admin credentials")
		return
	}

	session := sessions.Default(ctx)
	session.Set("admin_user", true)
	session.Delete("department_user")

	if err := session.Save(); err != nil {
		ctx.String(http.StatusInternalServerError, "admin login failed")
		return
	}

	ctx.String(http.StatusOK, "admin login successful")
}

func AdminWho(ctx *gin.Context) {
	if isAdminSession(ctx) {
		ctx.String(http.StatusOK, "admin")
		return
	}

	ctx.String(http.StatusOK, "no admin is logged in")
}
