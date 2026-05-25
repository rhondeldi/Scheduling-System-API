package Auth

import (
	"crypto/subtle"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Departments"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
	"golang.org/x/crypto/bcrypt"
)

type DepartmentLoginForm struct {
	ID       uint16 `json:"id"`
	Code     string `json:"code"`
	Password string `json:"password"`
}

func Login(ctx *gin.Context) {
	log.Print("user login attempt")

	login_department := &DepartmentLoginForm{}

	if err := ctx.BindJSON(&login_department); err != nil {
		log.Print("error binding form data")
		ctx.String(http.StatusBadRequest, "we are unable to properly read the department to be added")
		return
	}

	if login_department.ID == 0 {
		log.Print("general department login rejected")
		ctx.String(http.StatusBadRequest, "we are unable to properly read the department to be added")
		return
	}

	// find

	departments, err_read_all_departments := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllDepartments()

	if err_read_all_departments != nil {
		log.Printf("error - %s", err_read_all_departments.Error())
		ctx.String(http.StatusInternalServerError, "we're currently unable to get the list of all departments")
		return
	}

	var department_found *Departments.Department

	for _, department := range departments {

		is_equal_id := department.DepartmentID == login_department.ID
		is_equal_code := Utils.IsEqualStrCaseInsensitiveIgnoreWhiteSpace(department.Code, login_department.Code)

		if is_equal_id && is_equal_code {
			dept := department
			department_found = &dept
		}
	}

	if department_found == nil {
		log.Print("department not found")
		ctx.String(http.StatusNotFound, "that department either does not exist in the records, or the credentials might be incorrect")
		return
	}

	log.Print("department found")
	Utils.PrettyPrint(department_found)

	log.Print("department login")
	Utils.PrettyPrint(login_department)

	/////////////////////// validate user password ///////////////////////

	if len(os.Getenv("MASTER_KEY")) >= 32 {
		is_equal := subtle.ConstantTimeCompare(
			[]byte(os.Getenv("MASTER_KEY")),
			[]byte(login_department.Password),
		) == 1

		if !is_equal {
			log.Print("master key did not match")
			ctx.String(http.StatusUnauthorized, "incorrect credentials")
			return
		}
	} else {
		if err := bcrypt.CompareHashAndPassword([]byte(department_found.SaltedHashedPassword), []byte(login_department.Password)); err != nil {
			log.Print("password did not match")
			ctx.String(http.StatusUnauthorized, "incorrect credentials")
			return
		}
	}

	/////////////////////// create user session ///////////////////////

	session := sessions.Default(ctx)
	department_logged_in := session.Get("department_user")

	log.Printf("login session get user %+v", department_logged_in)

	log.Printf("department ID to login : %d", department_found.DepartmentID)

	if department_logged_in == nil {
		session.Set("department_user", department_found.DepartmentID)
		session.Delete("admin_user")

		log.Print("Login: [saving session]")
		err_save_session := session.Save()

		if err_save_session != nil {
			log.Print("Login [session-save-error] : ", err_save_session.Error())
			ctx.String(http.StatusInternalServerError, "login failed")
			return
		} else {
			log.Print("Login [session-save-success] ")
		}

		log.Print("Login [session-got] : ", session.Get("department_user"))
		log.Print("Login [session-reflected] : ", reflect.TypeOf(session.Get("department_user")))

		ctx.String(http.StatusOK, "login successful")
		return
	} else {
		log.Print("Login: [already-logged-in]")
		ctx.String(http.StatusAlreadyReported, "you are already logged in")
	}

}

func Who(c *gin.Context) {
	session := sessions.Default(c)

	fmt.Printf("\nTest Session : %+v\n", session)

	user := session.Get("department_user")

	if user == nil {
		log.Print("who? NONE : ", user)
		c.String(http.StatusOK, "no one is logged in")
		return
	}

	log.Print("who? : ", user)

	c.JSON(http.StatusOK, user)
}

func LogOut(c *gin.Context) {
	session := sessions.Default(c)

	departmentUser := session.Get("department_user")
	adminUser := session.Get("admin_user")

	if departmentUser == nil && adminUser == nil {
		c.String(http.StatusForbidden, "you're not logged in")
		return
	}

	session.Delete("department_user")
	session.Delete("admin_user")

	if err_save_session := session.Save(); err_save_session != nil {
		c.String(http.StatusInternalServerError, "logout failed")
		return
	}

	c.String(http.StatusOK, "logout success")
}
