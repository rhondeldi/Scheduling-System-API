package Auth

import (
	"fmt"
	"os"

	"github.com/gin-contrib/sessions/cookie"
)

var SessionStore cookie.Store

func InitSessionStore() error {
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = os.Getenv("SESSION_HASH_KEY")
	}

	if sessionSecret == "" {
		return fmt.Errorf("SESSION_SECRET is not set")
	}

	SessionStore = cookie.NewStore([]byte(sessionSecret))
	return nil
}
