package Auth

import (
	"os"

	"github.com/gin-contrib/sessions/cookie"
)

// in memory session secret should be 32 bytes to use AES256
var SessionStore = cookie.NewStore([]byte(os.Getenv("SESSION_SECRET")))
