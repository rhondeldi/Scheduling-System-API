package RoutesV1

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"go.mongodb.org/mongo-driver/mongo"
)

/*
Any "error" will return a `nil` university schedule.

retrieves university schedule from cache or persistence, can return empty university schedule if there is no university schedule generated yet.

example usage inside a gin route:

	// setting departments_to_validate to nil will validate all departments.
	university_schedules, is_success := ObtainUniversitySchedule(ctx, nil, selected_semester)
	if !is_success {
		return
	}
*/
func ObtainUniversitySchedule(ctx *gin.Context, departments_to_validate map[uint16]bool, semester int) (Schedule.UniTimeTables, bool) {
	var university_schedules Schedule.UniTimeTables = nil

	cached_university_schedule, has_cache, err_get_cache := RouteGlobals.GetCachedUniversitySchedule(semester)

	if err_get_cache != nil {
		log.Println("ObtainUniversitySchedule: [cache-error] , caused by ", err_get_cache.Error())
		ctx.String(http.StatusBadRequest, err_get_cache.Error())
		return nil, false
	}

	rooms, err_read_all_rooms := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllRooms()

	if err_read_all_rooms != nil {
		log.Print("ObtainUniversitySchedule: [read-rooms-error] caused by ", err_read_all_rooms)
		ctx.String(http.StatusInternalServerError, "we can not retrieve the rooms information right now")
		return nil, false
	}

	curriculums, err_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_curriculums != nil {
		log.Print("ObtainUniversitySchedule: [read-curriculums-error] caused by ", err_curriculums)
		ctx.String(http.StatusInternalServerError, "we can not retrieve the curriculums information right now")
		return nil, false
	}

	if has_cache {
		log.Println("ObtainUniversitySchedule: [retrieved-from-cache] retrieving university schedule from cache.")
		university_schedules = cached_university_schedule
	} else {
		log.Println("ObtainUniversitySchedule: [cache-not-found] loading from persistence")
		read_university_schedules, err_load_schedules := RouteGlobals.SchedulePersistence.LoadService.LoadSchedules(semester)

		if err_load_schedules != nil {
			log.Println("ObtainUniversitySchedule: [persistence-failed] , caused by ", err_load_schedules)

			if errors.Is(err_load_schedules, os.ErrNotExist) || mongo.ErrNoDocuments == err_load_schedules {
				log.Printf(
					"ObtainUniversitySchedule: [no-schedule] schedule for %s is not created yet, creating an empty schedule instead",
					Curriculum.SEMESTER_INDEX_NAME[semester],
				)

				university_schedules = GeneticAlgorithm.NewEmptyIndividual(curriculums, semester)
			} else {
				ctx.String(http.StatusInternalServerError, "we failed to read that schedule")
				return nil, false
			}
		} else {
			university_schedules = read_university_schedules
		}
	}

	if university_schedules.IsEmpty() {
		return university_schedules, true
	}

	if errs := university_schedules.VerticalValidation(rooms); len(errs) > 0 {
		log.Println("ObtainUniversitySchedule: invalid schedule detected, vertical overlap, caused by:")

		for _, e := range errs {
			fmt.Println(e.Error())
		}

		fmt.Print("\n\n")

		ctx.String(http.StatusConflict, "server detected an invalid schedule with vertically overlapping data")
		return nil, false
	}

	errs_horizontal_validation := GeneticAlgorithm.HorizontalValidation(
		university_schedules, curriculums, departments_to_validate, semester,
	)

	if len(errs_horizontal_validation) > 0 {
		log.Println("ObtainUniversitySchedule: invalid schedule detected, horizontal overlap, caused by:")

		for _, e := range errs_horizontal_validation {
			fmt.Println(e.Error())
		}

		fmt.Print("\n\n")

		ctx.String(http.StatusConflict, "server detected an invalid schedule with wrong horizontal data allocations")
		return nil, false
	}

	log.Println("ObtainUniversitySchedule: schedule found")
	return university_schedules, true
}

/*
Any "error" will return a `nil` university schedule.

retrieves university schedule from cache or persistence, can return empty university schedule if there is no university schedule generated yet.

example usage inside a gin route:

	// setting departments_to_validate to nil will validate all departments.
	university_schedules, is_success := ObtainUniversityScheduleNoHorizontalValidation(ctx, selected_semester)
	if !is_success {
		return
	}
*/
func ObtainUniversityScheduleNoHorizontalValidation(ctx *gin.Context, semester int) (Schedule.UniTimeTables, bool) {
	var university_schedules Schedule.UniTimeTables = nil

	cached_university_schedule, has_cache, err_get_cache := RouteGlobals.GetCachedUniversitySchedule(semester)

	if err_get_cache != nil {
		log.Println("ObtainUniversityScheduleNoHorizontalValidation: [cache-error] , caused by ", err_get_cache.Error())
		ctx.String(http.StatusBadRequest, err_get_cache.Error())
		return nil, false
	}

	rooms, err_read_all_rooms := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllRooms()

	if err_read_all_rooms != nil {
		log.Print("ObtainUniversityScheduleNoHorizontalValidation: [read-rooms-error] caused by ", err_read_all_rooms)
		ctx.String(http.StatusInternalServerError, "we can not retrieve the rooms information right now")
		return nil, false
	}

	curriculums, err_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_curriculums != nil {
		log.Print("ObtainUniversityScheduleNoHorizontalValidation: [read-curriculum-error] caused by ", err_curriculums)
		ctx.String(http.StatusInternalServerError, "we can not retrieve the curriculums information right now")
		return nil, false
	}

	if has_cache {
		log.Println("ObtainUniversityScheduleNoHorizontalValidation: [retrieved-from-cache] retrieving university schedule from cache.")
		university_schedules = cached_university_schedule
	} else {
		log.Println("ObtainUniversityScheduleNoHorizontalValidation: [cache-not-found] loading from persistence")
		read_university_schedules, err_load_schedules := RouteGlobals.SchedulePersistence.LoadService.LoadSchedules(semester)

		if err_load_schedules != nil {
			log.Println("ObtainUniversityScheduleNoHorizontalValidation: [persistence-failed] , caused by ", err_load_schedules)

			if errors.Is(err_load_schedules, os.ErrNotExist) || mongo.ErrNoDocuments == err_load_schedules {
				log.Printf(
					"ObtainUniversityScheduleNoHorizontalValidation: [no-schedule] schedule for %s is not created yet, creating an empty schedule instead",
					Curriculum.SEMESTER_INDEX_NAME[semester],
				)

				university_schedules = GeneticAlgorithm.NewEmptyIndividual(curriculums, semester)
			} else {
				ctx.String(http.StatusInternalServerError, "we failed to read that schedule")
				return nil, false
			}
		} else {
			university_schedules = read_university_schedules
		}
	}

	if university_schedules.IsEmpty() {
		return university_schedules, true
	}

	for _, err_vertical_validation := range university_schedules.VerticalValidation(rooms) {
		if err_vertical_validation != nil {
			log.Println("ObtainUniversityScheduleNoHorizontalValidation: invalid schedule detected - vertical overlap")
			ctx.String(http.StatusConflict, "server detected an invalid schedule with vertically overlapping data")
			return nil, false
		}
	}

	log.Println("ObtainUniversityScheduleNoHorizontalValidation: schedule found")
	return university_schedules, true
}

/*
retrieves university schedule from cache or persistence, can return empty university schedule if there is no university schedule generated yet.

this obtain function can return non-nil university schedules despite having a vertical or horizontal data overlaps

example usage inside a gin route:

	// setting departments_to_validate to nil will validate all departments.
	university_schedules, err_obtain_sched_no_ctx := ObtainUniversityScheduleNoContext(nil, selected_semester)
	if err_obtain_sched_no_ctx != nil {
		// handle error
		return
	}
*/
func ObtainUniversityScheduleNoContext(departments_to_validate map[uint16]bool, semester int) (Schedule.UniTimeTables, error) {
	var university_schedules Schedule.UniTimeTables = nil

	cached_university_schedule, has_cache, err_get_cache := RouteGlobals.GetCachedUniversitySchedule(semester)

	if err_get_cache != nil {
		log.Println("ObtainUniversityScheduleNoContext: [cache-error] , caused by ", err_get_cache.Error())
		return nil, err_get_cache
	}

	rooms, err_read_all_rooms := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllRooms()

	if err_read_all_rooms != nil {
		log.Print("ObtainUniversityScheduleNoContext: [read-rooms-error] caused by ", err_read_all_rooms)
		return nil, err_read_all_rooms
	}

	curriculums, err_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_curriculums != nil {
		log.Print("ObtainUniversityScheduleNoContext: [read-curriculums-error] caused by ", err_curriculums)
		return nil, err_curriculums
	}

	if has_cache {
		log.Println("ObtainUniversityScheduleNoContext: [retrieved-from-cache] retrieving university schedule from cache.")
		university_schedules = cached_university_schedule
	} else {
		log.Println("ObtainUniversityScheduleNoContext: [cache-not-found] loading from persistence")
		read_university_schedules, err_load_schedules := RouteGlobals.SchedulePersistence.LoadService.LoadSchedules(semester)

		if err_load_schedules != nil {
			log.Println("ObtainUniversityScheduleNoContext: [persistence-failed] , caused by ", err_load_schedules)

			if errors.Is(err_load_schedules, os.ErrNotExist) || mongo.ErrNoDocuments == err_load_schedules {
				log.Printf(
					"ObtainUniversityScheduleNoContext: [no-schedule] schedule for %s is not created yet, creating an empty schedule instead",
					Curriculum.SEMESTER_INDEX_NAME[semester],
				)

				university_schedules = GeneticAlgorithm.NewEmptyIndividual(curriculums, semester)
			} else {
				return nil, fmt.Errorf("failed to read the university schedule for the %s", Curriculum.SEMESTER_INDEX_NAME[semester])
			}
		} else {
			university_schedules = read_university_schedules
		}
	}

	if university_schedules.IsEmpty() {
		return university_schedules, nil
	}

	for _, err_vertical_validation := range university_schedules.VerticalValidation(rooms) {
		if err_vertical_validation != nil {
			log.Println("ObtainUniversityScheduleNoContext: invalid schedule detected, vertical overlaps")
			return university_schedules, errors.New("server detected an invalid schedule with vertically overlapping data")
		}
	}

	errs_horizontal_validation := GeneticAlgorithm.HorizontalValidation(
		university_schedules, curriculums, departments_to_validate, semester,
	)

	for _, err_horizontal_validation := range errs_horizontal_validation {
		if err_horizontal_validation != nil {
			log.Println("ObtainUniversityScheduleNoContext: invalid schedule detected, horizontal overlaps")
			return university_schedules, errors.New("server detected an invalid schedule with horizontally overlapping data")
		}
	}

	log.Println("ObtainUniversityScheduleNoContext: schedule found")
	return university_schedules, nil
}

/*
NO "Horizontal" validation.

retrieves university schedule from cache or persistence, can return empty university schedule if there is no university schedule generated yet.

this obtain function can return non-nil university schedules despite having a vertical data overlaps

example usage inside a gin route:

	university_schedules, err_obtain_sched_no_ctx := ObtainUniversityScheduleNoContextNoHorizontalValidation(selected_semester)
	if err_obtain_sched_no_ctx != nil {
		// handle error
		return
	}
*/
func ObtainUniversityScheduleNoContextNoHorizontalValidation(semester int) (Schedule.UniTimeTables, error) {
	var university_schedules Schedule.UniTimeTables = nil

	cached_university_schedule, has_cache, err_get_cache := RouteGlobals.GetCachedUniversitySchedule(semester)

	if err_get_cache != nil {
		log.Println("ObtainUniversityScheduleNoContextNoHorizontalValidation: [cache-error] , caused by ", err_get_cache.Error())
		return nil, err_get_cache
	}

	rooms, err_read_all_rooms := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllRooms()

	if err_read_all_rooms != nil {
		log.Print("ObtainUniversityScheduleNoContextNoHorizontalValidation: [read-rooms-error] caused by ", err_read_all_rooms)
		return nil, err_read_all_rooms
	}

	if has_cache {
		log.Println("ObtainUniversityScheduleNoContextNoHorizontalValidation: [retrieved-from-cache] retrieving university schedule from cache.")
		university_schedules = cached_university_schedule
	} else {
		log.Println("ObtainUniversityScheduleNoContextNoHorizontalValidation: [cache-not-found] loading from persistence")
		read_university_schedules, err_load_schedules := RouteGlobals.SchedulePersistence.LoadService.LoadSchedules(semester)

		if err_load_schedules != nil {
			log.Println("ObtainUniversityScheduleNoContextNoHorizontalValidation: [persistence-failed] , caused by ", err_load_schedules)

			if errors.Is(err_load_schedules, os.ErrNotExist) || mongo.ErrNoDocuments == err_load_schedules {
				log.Printf(
					"ObtainUniversityScheduleNoContextNoHorizontalValidation: [no-schedule] schedule for %s is not created yet, creating an empty schedule instead",
					Curriculum.SEMESTER_INDEX_NAME[semester],
				)

				curriculums, err_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

				if err_curriculums != nil {
					log.Print("ObtainUniversityScheduleNoContextNoHorizontalValidation: [read-curriculums-error] caused by ", err_curriculums)
					return nil, err_curriculums
				}

				university_schedules = GeneticAlgorithm.NewEmptyIndividual(curriculums, semester)
			} else {
				return nil, fmt.Errorf("failed to read the university schedule for the %s", Curriculum.SEMESTER_INDEX_NAME[semester])
			}
		} else {
			university_schedules = read_university_schedules
		}
	}

	if university_schedules.IsEmpty() {
		return university_schedules, nil
	}

	for _, err_vertical_validation := range university_schedules.VerticalValidation(rooms) {
		if err_vertical_validation != nil {
			log.Println("ObtainUniversityScheduleNoContextNoHorizontalValidation: invalid schedule detected, vertical overlaps")
			return university_schedules, errors.New("server detected an invalid schedule with vertically overlapping data")
		}
	}

	log.Println("ObtainUniversityScheduleNoContextNoHorizontalValidation: schedule found")
	return university_schedules, nil
}

/*
no gin "ctx", no "Vertical" and "Horizontal" validation at all.

retrieves university schedule from cache or persistence, can return empty university schedule if there is no university schedule generated yet.

example usage inside a gin route:

	university_schedules, err_obtain_sched_no_ctx := ObtainUniversityScheduleNoValidation(selected_semester)
	if err_obtain_sched_no_ctx != nil {
		// handle error
		return
	}
*/
func ObtainUniversityScheduleNoValidation(semester int) (Schedule.UniTimeTables, error) {
	var university_schedules Schedule.UniTimeTables = nil

	cached_university_schedule, has_cache, err_get_cache := RouteGlobals.GetCachedUniversitySchedule(semester)

	if err_get_cache != nil {
		log.Println("ObtainUniversityScheduleNoValidation: [cache-error] , caused by ", err_get_cache.Error())
		return nil, err_get_cache
	}

	if has_cache {
		log.Println("ObtainUniversityScheduleNoValidation: [retrieved-from-cache] retrieving university schedule from cache.")
		university_schedules = cached_university_schedule
	} else {
		log.Println("ObtainUniversityScheduleNoValidation: [cache-not-found] loading from persistence")
		read_university_schedules, err_load_schedules := RouteGlobals.SchedulePersistence.LoadService.LoadSchedules(semester)

		if err_load_schedules != nil {
			log.Println("ObtainUniversityScheduleNoValidation: [persistence-failed] , caused by ", err_load_schedules)

			if errors.Is(err_load_schedules, os.ErrNotExist) || mongo.ErrNoDocuments == err_load_schedules {
				log.Printf(
					"ObtainUniversityScheduleNoValidation: [no-schedule] schedule for %s is not created yet, creating an empty schedule instead",
					Curriculum.SEMESTER_INDEX_NAME[semester],
				)

				curriculums, err_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

				if err_curriculums != nil {
					log.Print("ObtainUniversityScheduleNoValidation: [read-curriculums-error] caused by ", err_curriculums)
					return nil, err_curriculums
				}

				university_schedules = GeneticAlgorithm.NewEmptyIndividual(curriculums, semester)
			} else {
				return nil, fmt.Errorf("failed to read the university schedule for the %s", Curriculum.SEMESTER_INDEX_NAME[semester])
			}
		} else {
			university_schedules = read_university_schedules
		}
	}

	return university_schedules, nil
}
