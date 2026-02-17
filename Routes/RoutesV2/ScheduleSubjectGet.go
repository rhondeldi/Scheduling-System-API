package RoutesV2

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Routes/RoutesV1"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

type TimeSlotMoveAvailability [Const.N_WEEKLY_SCHOOL_DAYS][Const.N_DAILY_TIME_SLOTS]bool

/*
POST:

	"/available_subject_moves?department_id=[N>0]&semester=[0-N>=1]&curriculum_id=[N>0]&year_level_idx=[0-N>=1]&section_idx=[0-N>=1]"
*/
func GetSubjectAvailableTimeSlotMoves(ctx *gin.Context) {

	if RouteGlobals.IsGeneratingSchedule.Load() {
		log.Print("v2.GetSubjectAvailableTimeSlotMoves: [busy] you or other department(s) are still generating a schedule, please wait until the process is finished")
		ctx.String(http.StatusForbidden, "we're unable to get the available time slots right now, you or other department(s) are still generating a schedule, please wait a little while until those process are done")
		return
	}

	var subject RoutesV1.SubjectAssignmentInfo

	if err := ctx.BindJSON(&subject); err != nil {
		ctx.String(http.StatusBadRequest, "we are unable to properly read the prefered configured week time table")
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

	// generate encoding resource

	default_empty_encoding_resource, err_read_default_encoding := GeneticAlgorithm.ReadDefaultEncodingResource(RouteGlobals.ResourcesPersistence)

	if err_read_default_encoding != nil {
		log.Print("GetSubjectAvailableTimeSlotMoves: we're unable to retrieve the default encoding resource for the requested semester, caused by ", err_read_default_encoding)
		ctx.String(http.StatusInternalServerError, "we're unable to retrieve the default encoding resource required")
		return
	}

	encoding_resource, err_gen_encoding_resource := GeneticAlgorithm.GenerateEncodingResourceFromUniTimeTable(
		university_schedules, all_curriculums, selected_semester, default_empty_encoding_resource,
	)

	if err_gen_encoding_resource != nil {
		ctx.String(http.StatusInternalServerError, "we're unable to generate encoding resource")
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
		ctx.String(http.StatusNotFound, "we're unable to find that section, please make sure the parameters are correct?")
		return
	}

	section_week_uni_sched := university_schedules[schedule_idx]
	subject_available_time_slot_moves := TimeSlotMoveAvailability{}

	// departments to include

	departments_to_include := [2]uint16{
		0,                     // general department
		uint16(department_id), // selected department
	}

	// generate instructor id to instructor map (includes general instructors and department instructors)

	instructor_id_to_instructor := make(map[uint16]*Instructors.Instructor)

	for _, department := range departments_to_include {
		for _, instructor := range encoding_resource.DeptIdToInstructors[department] {
			if instructor.InstructorID == 0 {
				continue
			}

			instructor_id_to_instructor[instructor.InstructorID] = &instructor
		}
	}

	// generate room id to room map (includes general rooms and department rooms)

	room_id_to_room := make(map[uint16]*Rooms.Room)

	for _, department := range departments_to_include {
		for room_type := range Rooms.ROOM_TYPE_NAMES {
			room_type_to_rooms := encoding_resource.DeptIdToRoomtypeToRooms[department]
			for _, room := range room_type_to_rooms[uint16(room_type)] {
				if room.RoomID == 0 {
					continue
				}

				room_id_to_room[room.RoomID] = &room
			}
		}
	}

	// populate subject available time slot moves

	for day := range Const.N_WEEKLY_SCHOOL_DAYS {
		for time_slot := range Const.N_DAILY_TIME_SLOTS {

			subject_id := section_week_uni_sched[day][time_slot].GetSubjectID()
			instructor_id := section_week_uni_sched[day][time_slot].GetInstructorID()
			room_id := section_week_uni_sched[day][time_slot].GetRoomID()

			subject_available_time_slot_moves[day][time_slot] = true

			is_same_time_slot := (subject_id == subject.SubjectID && instructor_id == subject.InstructorID && room_id == subject.RoomID)

			if is_same_time_slot {
				continue
			}

			is_free_time_slot := subject_id == 0

			is_available_instructor := instructor_id_to_instructor[subject.InstructorID].Time.GetAvailability(day, time_slot)

			is_available_room := room_id_to_room[subject.RoomID].GetTimeSlotClassCount(day, time_slot) < uint8(room_id_to_room[subject.RoomID].Capacity)

			if !is_free_time_slot || !is_available_instructor || !is_available_room {
				subject_available_time_slot_moves[day][time_slot] = false
			}
		}
	}

	// edit university schedules
	ctx.JSON(http.StatusOK, subject_available_time_slot_moves)
}
