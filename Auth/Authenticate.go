package Auth

import (
	"log"
	"net/http"
	"os"
	"reflect"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func isAdminSession(ctx *gin.Context) bool {
	session := sessions.Default(ctx)
	adminUser := session.Get("admin_user")

	switch value := adminUser.(type) {
	case bool:
		return value
	case string:
		return value == "true"
	default:
		return false
	}
}

func getSessionDepartmentID(ctx *gin.Context) (uint16, bool) {
	session := sessions.Default(ctx)
	user := session.Get("department_user")

	if user == nil {
		return 0, false
	}

	switch value := user.(type) {
	case uint16:
		return value, true
	case uint:
		return uint16(value), true
	case int:
		if value < 0 {
			return 0, false
		}
		return uint16(value), true
	case int32:
		if value < 0 {
			return 0, false
		}
		return uint16(value), true
	case int64:
		if value < 0 {
			return 0, false
		}
		return uint16(value), true
	case float64:
		if value < 0 {
			return 0, false
		}
		return uint16(value), true
	case string:
		parsed, err := strconv.ParseUint(value, 10, 16)
		if err != nil {
			return 0, false
		}
		return uint16(parsed), true
	default:
		return 0, false
	}
}

func IsAuthSuccess(ctx *gin.Context) bool {
	if os.Getenv("AUTH") == "enable" {
		if isAdminSession(ctx) {
			log.Print("[IsAuthSuccess] authenticated admin user")
			return true
		}

		sessionDepartmentID, isFound := getSessionDepartmentID(ctx)

		log.Print("authenticated department user      [IsAuthSuccess] : ", sessionDepartmentID)
		log.Print("authenticated department reflected [IsAuthSuccess] : ", reflect.TypeOf(sessionDepartmentID))

		if !isFound {
			ctx.String(http.StatusUnauthorized, "you're not allowed to access that, please login first")
			return false
		} else {
			log.Print("[IsAuthSuccess] authenticated department user")
		}
	}

	return true
}

func IsDepartmentAllowed(ctx *gin.Context, target_department_id uint16) bool {
	if os.Getenv("AUTH") == "enable" {
		if isAdminSession(ctx) {
			log.Print("[IsDepartmentAllowed] authenticated admin user")
			return true
		}

		sessionDepartmentID, isFound := getSessionDepartmentID(ctx)

		log.Print("target department id [IsDepartmentAllowed] : ", target_department_id)
		log.Print("authenticated department user      [IsDepartmentAllowed] : ", sessionDepartmentID)
		log.Print("authenticated department reflected [IsDepartmentAllowed] : ", reflect.TypeOf(sessionDepartmentID))

		if !isFound {
			ctx.String(http.StatusUnauthorized, "you're not allowed to access that, please login first")
			return false
		}

		if !(target_department_id == 0 || sessionDepartmentID == target_department_id) {
			ctx.String(http.StatusForbidden, "your department is not allowed to edit, update, or add data to other departments for this specific operation")
			return false
		} else {
			log.Print("[IsDepartmentAllowed] authenticated department user")
		}
	}

	return true
}
