package RoutesV2

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/Auth"
	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Routes/RoutesV1"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

/*
POST:

	"/subject_move?department_id=[N>0]&semester=[0-N>=1]&curriculum_id=[N>0]&year_level_idx=[0-N>=1]&section_idx=[0-N>=1]"
*/
func PostSubjectTimeSlotMove(ctx *gin.Context) {

	if is_success := Auth.IsAuthSuccess(ctx); !is_success {
		return
	}

	if RouteGlobals.IsGeneratingSchedule.Load() {
		log.Print("PostSubjectTimeSlotMove: [busy] you or other department(s) are still generating a schedule, please wait until the process is finished")
		ctx.String(http.StatusForbidden, "we're unable to move the subject right now, you or other department(s) are still generating a schedule, please wait a little while until those process are done")
		return
	}

	RouteGlobals.ReindexUniSchedMutex.Lock()
	defer RouteGlobals.ReindexUniSchedMutex.Unlock()

	var subject RoutesV1.SubjectAssignmentInfo

	if err := ctx.BindJSON(&subject); err != nil {
		ctx.String(http.StatusBadRequest, "we are unable to properly read the moved subject")
		return
	}

	Utils.PrettyPrint(subject)

	// parse curriculum id

	curriculum_id, is_valid_curriculum_id_param := RoutesV1.IsValidCurriculumID(ctx)

	if !is_valid_curriculum_id_param {
		return
	}

	// pasrse year level index

	param_year_level_idx, is_valid_year_level_idx_param := RoutesV1.IsValidIndex(ctx, "year_level_idx")

	if !is_valid_year_level_idx_param {
		return
	}

	// parse section index

	param_section_idx, is_valid_section_idx_param := RoutesV1.IsValidIndex(ctx, "section_idx")

	if !is_valid_section_idx_param {
		return
	}

	// parse semester parameter

	selected_semester, is_valid_semester_param := RoutesV1.IsValidParameterSemesterIndex(ctx)

	if !is_valid_semester_param {
		return
	}

	// parse department_id parameter

	department_id, is_valid_department_id_param := RoutesV1.IsValidParameterDepartmentID(ctx)

	if !is_valid_department_id_param {
		return
	}

	if is_allowed := Auth.IsDepartmentAllowed(ctx, uint16(department_id)); !is_allowed {
		return
	}

	// get all curriculums

	all_curriculums, err_read_all_curriculum := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_read_all_curriculum != nil {
		ctx.String(http.StatusInternalServerError, "unable to read curriculums for that department")
		return
	}

	// load university schedules

	university_schedules, has_obtained := RoutesV1.ObtainUniversityScheduleNoHorizontalValidation(ctx, selected_semester)

	if !has_obtained {
		return
	}

	// cache the found university schedule for the semester

	err_set_cache := RouteGlobals.SetCachedUniversitySchedule(selected_semester, university_schedules)

	if err_set_cache != nil {
		log.Println(err_set_cache.Error())
	}

	// parse schedule_idx parameter

	schedule_idx := -1

	GeneticAlgorithm.IterateSectionsWeekSchedule(university_schedules, all_curriculums, selected_semester, nil, nil,
		func(indicies GeneticAlgorithm.IterIndices, values GeneticAlgorithm.IterValues) GeneticAlgorithm.IterReturnType {

			curriculum := values.Curriculum
			year_level_idx := indicies.YearLevel
			section_idx := indicies.Section

			if curriculum_id == int(curriculum.CurriculumID) && year_level_idx == param_year_level_idx && section_idx == param_section_idx {
				schedule_idx = indicies.Usi
				return GeneticAlgorithm.IterBreakCurriculumLoop
			}

			return GeneticAlgorithm.IterProceed
		},
	)

	if schedule_idx < 0 {
		log.Print("PostSubjectTimeSlotMove: [section-schedule-not-found] unable to find the schedule for the given parameters")
		ctx.String(http.StatusNotFound, "we're unable to find that section, please make sure the parameters are correct?")
		return
	}

	// populate subject available time slot moves

	for day := range Const.N_WEEKLY_SCHOOL_DAYS {
		for time_slot := range Const.N_DAILY_TIME_SLOTS {

			subject_id := university_schedules[schedule_idx][day][time_slot].GetSubjectID()
			instructor_id := university_schedules[schedule_idx][day][time_slot].GetInstructorID()
			room_id := university_schedules[schedule_idx][day][time_slot].GetRoomID()

			is_same_time_slot := (subject_id == subject.SubjectID && instructor_id == subject.InstructorID && room_id == subject.RoomID)

			if is_same_time_slot {
				university_schedules[schedule_idx][day][time_slot].SetSubjectID(0)
				university_schedules[schedule_idx][day][time_slot].SetInstructorID(0)
				university_schedules[schedule_idx][day][time_slot].SetRoomID(0)
			}
		}
	}

	for i := range subject.SubjectTimeSlots {
		university_schedules[schedule_idx][subject.DayIdx][subject.TimeSlotIdx+i].SetSubjectID(subject.SubjectID)
		university_schedules[schedule_idx][subject.DayIdx][subject.TimeSlotIdx+i].SetInstructorID(subject.InstructorID)
		university_schedules[schedule_idx][subject.DayIdx][subject.TimeSlotIdx+i].SetRoomID(subject.RoomID)
	}

	if err := RouteGlobals.SchedulePersistence.SaveService.SaveSchedules(university_schedules, selected_semester); err != nil {
		log.Print("PostSubjectTimeSlotMove: [save-failed] error unable to save the genetic algorithm's generated schedule, caused by :", err.Error())
		ctx.String(http.StatusInternalServerError, "we're unable to save the schedule right now")
		return
	}

	ctx.String(http.StatusOK, "subject move success")
}
