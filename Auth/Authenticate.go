package Auth

import (
	"log"
	"net/http"
	"os"
	"reflect"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func IsAuthSuccess(ctx *gin.Context) bool {
	if os.Getenv("AUTH") == "enable" {
		session := sessions.Default(ctx)
		user := session.Get("department_user")

		log.Print("authenticated department user      [IsAuthSuccess] : ", user)
		log.Print("authenticated department reflected [IsAuthSuccess] : ", reflect.TypeOf(user))

		if user == nil {
			ctx.String(http.StatusUnauthorized, "you're not allowed to access that, please login first")
			return false
		} else {
			log.Print("[IsAuthSuccess] failed to authenticate department user")
		}
	}

	return true
}

func IsDepartmentAllowed(ctx *gin.Context, target_department_id uint16) bool {
	if os.Getenv("AUTH") == "enable" {
		session := sessions.Default(ctx)
		user := session.Get("department_user")

		log.Print("target department id [IsDepartmentAllowed] : ", target_department_id)
		log.Print("authenticated department user      [IsDepartmentAllowed] : ", user)
		log.Print("authenticated department reflected [IsDepartmentAllowed] : ", reflect.TypeOf(user))

		if !(target_department_id == 0 || user == target_department_id) {
			ctx.String(http.StatusForbidden, "your department is not allowed to edit, update, or add data to other departments for this specific operation")
			return false
		} else {
			log.Print("[IsDepartmentAllowed] failed to authenticate department user")
		}
	}

	return true
}
