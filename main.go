package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Routes/RoutesV1"
	"github.com/mrdcvlsc/scheduling-system-backend/Routes/RoutesV2"
	"github.com/mrdcvlsc/scheduling-system-backend/StorageResources"
	"github.com/mrdcvlsc/scheduling-system-backend/StorageSchedule"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

func main() {
	fmt.Println("Starting backend service")

	env_err := godotenv.Load()
	if env_err != nil {
		fmt.Println("Warning: .env file not loaded")
	}

	if sessionInitErr := Auth.InitSessionStore(); sessionInitErr != nil {
		panic(fmt.Sprintf("failed to initialize session store: %s", sessionInitErr.Error()))
	}

	switch os.Getenv("USE_DATABASE") {

	case "MongoDB":

		//////////////////////////////////////////////////////////////////////////
		//                         MongoDB Persistence
		//////////////////////////////////////////////////////////////////////////

		mongo_client := StorageResources.NewMongodbClient()
		defer StorageResources.CloseMongodbClient(mongo_client)

		RouteGlobals.ResourcesPersistence = &StorageResources.Persistence{
			ReaderService: &StorageResources.MongodbReader{
				Mongo: &StorageResources.MongoDB{Client: mongo_client},
			},

			WriterService: &StorageResources.MongodbWriter{
				Mongo: &StorageResources.MongoDB{Client: mongo_client},
			},
		}

		RouteGlobals.SchedulePersistence = &StorageSchedule.Persistence{
			LoadService: &StorageSchedule.MongodbReader{
				Mongo: &StorageSchedule.MongoDB{Client: mongo_client},
			},

			SaveService: &StorageSchedule.MongodbWriter{
				Mongo: &StorageSchedule.MongoDB{Client: mongo_client},
			},
		}

	default:

		//////////////////////////////////////////////////////////////////////////
		//                          JSON Persistence
		//////////////////////////////////////////////////////////////////////////

		RouteGlobals.ResourcesPersistence = &StorageResources.Persistence{
			ReaderService: &StorageResources.JsonReader{},
			WriterService: &StorageResources.JsonWriter{},
		}

		RouteGlobals.SchedulePersistence = &StorageSchedule.Persistence{
			LoadService: &StorageSchedule.JsonReader{},
			SaveService: &StorageSchedule.JsonWriter{},
		}
	}

	//////////////////////////////////////////////////////////////////////////
	//                         Initialize Globals
	//////////////////////////////////////////////////////////////////////////

	RouteGlobals.InitializeCachedUniversitySchedule()
	RouteGlobals.InitDeptSchedGenQueue()

	//////////////////////////////////////////////////////////////////////////

	use_secure_cookie := false
	same_site := http.SameSiteNoneMode

	if (os.Getenv("GIN_MODE") == "release") && (os.Getenv("DEV_MODE") == "local_release") {
		panic("those two modes are not allowed to be together")
	}

	cookie_expiration := 60 * 20 // dev 20 mins

	if os.Getenv("GIN_MODE") == "release" {
		use_secure_cookie = true
		same_site = http.SameSiteStrictMode
		cookie_expiration = 60 * 60 * 12 // 12 hours - cookie expiration
	} else if os.Getenv("DEV_MODE") == "local_release" {
		use_secure_cookie = false
		same_site = http.SameSiteLaxMode
		cookie_expiration = 60 * 60 // 1 hour - cookie expiration for local release
	}

	Auth.SessionStore.Options(sessions.Options{
		MaxAge:   cookie_expiration,
		Secure:   use_secure_cookie,
		HttpOnly: true,
		SameSite: same_site,
	})

	//////////////////////////////////////////////////////////////////////////

	router := gin.Default()

	// maximum memory limit for multipart form file uploads
	router.MaxMultipartMemory = 5 << 20 // 5 MiB

	router.Use(static.Serve("/", static.LocalFile("./dist", true)))
	router.Use(sessions.Sessions("session_id", Auth.SessionStore))

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	//////////////////////////////////////////////////////////////////////////
	//                              API-v1
	//////////////////////////////////////////////////////////////////////////

	v1 := router.Group("/v1")
	v2 := router.Group("/v2")

	v1.GET("/const", RoutesV1.GetConst)

	// ============= auth =============

	router.POST("/auth_admin_login", Auth.AdminLogin)
	router.POST("/auth_gasss_login", Auth.Login)
	v1.POST("/department_add", RoutesV1.PostDepartment)

	router.DELETE("/auth_gasss_logout", Auth.LogOut)
	router.GET("/admin_who", Auth.AdminWho)
	router.GET("/who", Auth.Who)

	// ============= department routes and handlers =============

	v1.GET("/all_departments", RoutesV1.GetAllDepartments)
	v1.GET("/departments", RoutesV1.GetDepartmentsPaginated)
	v1.GET("/department_data", RoutesV1.GetCurriculumsDataInDepartment)
	v1.PATCH("/department_update", RoutesV1.PatchDepartment)
	v1.DELETE("/department_remove", RoutesV1.DeleteDepartment)

	// ============= instructor routes and handlers =============

	v1.POST("/instructor_add", RoutesV1.PostInstructor)
	v1.PATCH("/instructor_update", RoutesV1.PatchInstructor)
	v1.DELETE("/instructor_remove", RoutesV1.DeleteInstructor)

	v2.GET("instructor_basic", RoutesV2.GetInstructorBasic)
	v2.GET("instructors", RoutesV2.GetDepartmentInstructors)
	v2.GET("instructor_resources", RoutesV2.GetInstructorResource)

	// ============= room routes and handlers =============

	v1.GET("/rooms", RoutesV1.GetDepartmentRooms)
	v1.GET("/room_allocation", RoutesV1.GetRoomSubjectAssignment)
	v1.POST("/room_add", RoutesV1.PostRoom)
	v1.PATCH("/room_update", RoutesV1.PatchRoom)
	v1.DELETE("/room_remove", RoutesV1.DeleteRoom)

	// ============= subject routes and handlers =============

	v1.GET("/subjects", RoutesV1.GetSubjects)
	v1.POST("/subject_add", RoutesV1.PostSubject)
	v1.PATCH("/subject_update", RoutesV1.PatchSubject)
	v1.DELETE("/subject_remove", RoutesV1.DeleteSubject)

	// ============= curriculum routes and handlers =============

	v1.GET("/curriculum_list", RoutesV1.GetDepartmentCurriculumList)
	v1.GET("/curriculum_load", RoutesV1.GetCurriculum)
	v1.POST("/curriculum_add", RoutesV1.PostCurriculum)
	v1.PATCH("/curriculum_update", RoutesV1.PatchCurriculum)
	v1.DELETE("/curriculum_remove", RoutesV1.DeleteCurriculum)

	// ============= schedule routes and handlers =============

	v2.GET("/estimate_resources", RoutesV2.GetEstimates)

	v1.GET("/gen_status", RoutesV1.GetGenStatus)
	v1.GET("/dept_gen_result", RoutesV1.GetDeptartmentGenerationResult)

	v1.GET("/university_schedule", RoutesV1.GetUniversitySchedule)
	v1.POST("/university_schedule", RoutesV1.PostUniversitySchedule)

	v1.GET("/class_schedule", RoutesV1.GetClassSchedule)
	v2.GET("/class_json_schedule", RoutesV2.GetJsonClassSchedule)
	v2.DELETE("/clear_class_schedule", RoutesV2.DeleteClearClassSchedule)
	v1.DELETE("/clear_department_schedules", RoutesV1.DeleteClearDepartmentSchedule)
	v1.GET("/delete_all_generated_university_schedules_for_all_semester_a_complete_reset", RoutesV1.DeleteAllUniversitySchedules)
	v1.POST("/generate_schedule", RoutesV1.RequestGenerateSchedule)

	v2.POST("/available_subject_moves", RoutesV2.GetSubjectAvailableTimeSlotMoves)
	v2.POST("/subject_move", RoutesV2.PostSubjectTimeSlotMove)

	v2.GET("/validate_schedules", RoutesV2.GetValidateSchedules)

	if os.Getenv("GIN_MODE") != "release" {
		v1.GET("/generate_schedule", RoutesV1.RequestGenerateSchedule) // for dev only
	}

	// ============= survery routes and handlers =============

	v2.POST("/add_schedule_preference", RoutesV2.PostWeekTimeTableSurvery)

	//////////////////////////////////////////////////////////////////////////

	if os.Getenv("GIN_MODE") != "release" {
		router.GET("/cpu", func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{"cpu": runtime.NumCPU()})
		})

		router.GET("/die", func(ctx *gin.Context) {
			ctx.String(http.StatusOK, "bye-bye")
			os.Exit(0)
		})
	}

	switch os.Getenv("USE_DATABASE") {
	case "MongoDB":
		router.GET("/using_database_persistence", func(ctx *gin.Context) { ctx.String(http.StatusAccepted, "m") })
	default:
		router.GET("/using_database_persistence", func(ctx *gin.Context) { ctx.String(http.StatusAccepted, "j") })
	}

	////////////////////////////////////////////////////////////////////////

	Utils.DisplayOutboundIP(os.Getenv("PORT"))
	router.Run(fmt.Sprintf(":%s", os.Getenv("PORT")))

}