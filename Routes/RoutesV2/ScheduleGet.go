package RoutesV2

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Routes/RoutesV1"
)

/*
GET:

	"/class_json_schedule?department_id=[N>0]&semester=[0-N>=1]&curriculum_id=[N>0]&year_level_idx=[0-N>=1]&section_idx=[0-N>=1]"
*/
func GetJsonClassSchedule(ctx *gin.Context) {

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

	// get all curriculums

	all_curriculums, err_read_all_curriculum := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_read_all_curriculum != nil {
		ctx.String(http.StatusInternalServerError, "Unable to load curriculums for the selected department.")
		return
	}

	total_number_of_sections := 0

	for _, curriculum := range all_curriculums {
		total_number_of_sections += curriculum.GetTotalSectionsBySemester(selected_semester)
	}

	log.Printf("GetJsonClassSchedule: [%s] total number of sections of all read curriculums is %d",
		Curriculum.SEMESTER_INDEX_NAME[selected_semester], total_number_of_sections,
	)

	log.Printf("GetJsonClassSchedule: [%s] read university schedule has a total length of..... %d",
		Curriculum.SEMESTER_INDEX_NAME[selected_semester], len(university_schedules),
	)

	// parse schedule_idx parameter

	schedule_idx := -1

	GeneticAlgorithm.IterateSectionsWeekSchedule(university_schedules, all_curriculums, selected_semester, nil, nil,
		func(indicies GeneticAlgorithm.IterIndices, values GeneticAlgorithm.IterValues) GeneticAlgorithm.IterReturnType {
			if curriculum_id == int(values.Curriculum.CurriculumID) && indicies.YearLevel == param_year_level_idx && indicies.Section == param_section_idx {
				schedule_idx = indicies.Usi
				return GeneticAlgorithm.IterBreakCurriculumLoop
			}

			return GeneticAlgorithm.IterProceed
		},
	)

	// check if section index was found

	if schedule_idx < 0 {
		log.Print("unable to find the section")
		ctx.String(http.StatusInternalServerError, "unable to find that section, the curriculum might have been edited, please refresh the page")
		return
	}

	// extract selected schedule

	selected_class_schedule := university_schedules[schedule_idx:(schedule_idx + 1)]

	department_to_measure := make(map[uint16]bool)
	department_to_measure[uint16(department_id)] = true

	log.Printf("university measured fitness : %f", GeneticAlgorithm.MeasureUniSchedBasicFitness(
		university_schedules, all_curriculums, nil, selected_semester,
	))

	log.Printf("department measured fitness : %f", GeneticAlgorithm.MeasureUniSchedBasicFitness(
		university_schedules, all_curriculums, department_to_measure, selected_semester,
	))

	log.Printf("     class measured fitness : %f", GeneticAlgorithm.MeasureWeekTimeTableBasicFitness(
		selected_class_schedule[0],
	))

	// process schedule information

	sub_id_to_subject_code := make(map[uint16]string)

	{ // subjects
		subjects, err_read_all_subjects := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllSubjects()

		if err_read_all_subjects != nil {
			ctx.String(http.StatusInternalServerError, "Unable to retrieve subject data at this time. Please try again later.")
			return
		}

		for _, subject := range subjects {
			sub_id_to_subject_code[subject.ID] = subject.Code
		}
	}

	instructor_id_to_instructor_name := make(map[uint16]string)

	{ // instructors
		instructors, err_read_all_instructors := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllInstructors()

		if err_read_all_instructors != nil {
			ctx.String(http.StatusInternalServerError, "Unable to retrieve instructor data at this time. Please try again later.")
			return
		}

		for _, instructor := range instructors {
			if instructor.DepartmentID == uint16(department_id) || instructor.DepartmentID == 0 {
				instructor_id_to_instructor_name[instructor.InstructorID] = instructor.LastName
			}
		}
	}

	room_id_to_room_name := make(map[uint16]string)

	{ // rooms
		rooms, err_read_all_rooms := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllRooms()

		if err_read_all_rooms != nil {
			ctx.String(http.StatusInternalServerError, "Unable to retrieve room data at this time. Please try again later.")
			return
		}

		for _, room := range rooms {
			if room.DepartmentID == uint16(department_id) || room.DepartmentID == 0 {
				room_id_to_room_name[room.RoomID] = room.Name
			}
		}
	}

	sub_assign_info := make([]RoutesV1.SubjectAssignmentInfo, 0)

	for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
		for time_slot := 0; time_slot < Const.N_DAILY_TIME_SLOTS; time_slot++ {
			slot := selected_class_schedule[0][day].GetTimeSlot(time_slot)

			if slot.GetSubjectID() == 0 {
				continue
			}

			new_sub_assignment := RoutesV1.SubjectAssignmentInfo{
				SubjectID:          slot.GetSubjectID(),
				InstructorID:       slot.GetInstructorID(),
				RoomID:             slot.GetRoomID(),
				SubjectCode:        sub_id_to_subject_code[slot.GetSubjectID()],
				InstructorLastName: instructor_id_to_instructor_name[slot.GetInstructorID()],
				RoomName:           room_id_to_room_name[slot.GetRoomID()],
				DayIdx:             uint8(day),
				TimeSlotIdx:        uint8(time_slot),
				SubjectTimeSlots:   1,
			}

			for forward_time_slot := time_slot + 1; forward_time_slot < Const.N_DAILY_TIME_SLOTS; forward_time_slot++ {
				forward_slot := selected_class_schedule[0][day].GetTimeSlot(forward_time_slot)

				if forward_slot.GetSubjectID() == slot.GetSubjectID() && forward_slot.GetInstructorID() == slot.GetInstructorID() && forward_slot.GetRoomID() == slot.GetRoomID() {
					new_sub_assignment.SubjectTimeSlots++
				} else {
					time_slot = forward_time_slot - 1
					break
				}

				if forward_time_slot == (Const.N_DAILY_TIME_SLOTS - 1) {
					time_slot = 9999
					break
				}
			}

			sub_assign_info = append(sub_assign_info, new_sub_assignment)
		}
	}

	ctx.JSON(http.StatusOK, sub_assign_info)
}

/*
GET:

	"/validate_schedules?department_id=[N>0]&semester=[0-N>=1]"
*/
func GetValidateSchedules(ctx *gin.Context) {

	// parse parameters

	selected_semester, is_valid_semester_param := RoutesV1.IsValidParameterSemesterIndex(ctx)

	if !is_valid_semester_param {
		return
	}

	department_id, is_valid_department_id_param := RoutesV1.IsValidParameterDepartmentID(ctx)

	if !is_valid_department_id_param {
		return
	}

	department_to_horizontal_validate := make(map[uint16]bool)
	department_to_horizontal_validate[uint16(department_id)] = true

	// load university schedules

	university_schedules, err_obtain := RoutesV1.ObtainUniversityScheduleNoValidation(selected_semester)

	if err_obtain != nil {
		log.Print("GetClassScheduleValidate - obtain error:", err_obtain)
		ctx.String(http.StatusInternalServerError, err_obtain.Error())
		return
	}

	// cache the found university schedule for the semester

	err_set_cache := RouteGlobals.SetCachedUniversitySchedule(selected_semester, university_schedules)

	if err_set_cache != nil {
		log.Println("GetClassScheduleValidate - cache error:", err_set_cache.Error())
	}

	// extract selected schedule

	validation_results := make([]any, 0)

	if university_schedules.IsEmpty() {
		validation_results = append(validation_results, "all university schedules are empty")
		ctx.JSON(http.StatusNotFound, validation_results)
		return
	}

	rooms, err_read_all_rooms := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllRooms()

	if err_read_all_rooms != nil {
		log.Print("GetValidateSchedules [error-rooms-read]: ", err_read_all_rooms)
		ctx.String(http.StatusInternalServerError, "Unable to retrieve room data at this time. Please try again later.")
		return
	}

	curriculums, err_read_all_curriculum := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_read_all_curriculum != nil {
		log.Print("GetValidateSchedules [error-curriculums-read]: ", err_read_all_curriculum)
		ctx.String(http.StatusInternalServerError, "Unable to load curriculum information at this time. Please refresh and try again later.")
		return
	}

	total_number_of_sections := Curriculum.GetTotalNumberOfSections(curriculums, selected_semester)

	if total_number_of_sections != len(university_schedules) {
		log.Print("GetValidateSchedules: curriculum sections vs schedule sections mismatch (emergency delete all schedule fix with GET: /v1/delete_all_generated_university_schedules_for_all_semester_a_complete_reset) : ", err_read_all_curriculum)
		ctx.String(http.StatusInternalServerError, "Indexing error, curriculum section count and schedule section count mismatch. Please contact the developers")
		return
	}

	default_empty_encoding_resource, err_read_default_encoding_resource := GeneticAlgorithm.ReadDefaultEncodingResource(RouteGlobals.ResourcesPersistence)

	if err_read_default_encoding_resource != nil {
		log.Print("GetValidateSchedules [error-encoding-resource-read]: ", err_read_default_encoding_resource)
		ctx.String(http.StatusInternalServerError, "Unable to read the default encoding resource right now. Please try again later.")
		return
	}

	encoding_resource, err_gen_encoding_resource := GeneticAlgorithm.GenerateEncodingResourceFromUniTimeTable(
		university_schedules, curriculums, selected_semester, default_empty_encoding_resource,
	)

	if err_gen_encoding_resource != nil {
		log.Print("GetValidateSchedules [error-encoding-resource-generation]: ", err_gen_encoding_resource)
		ctx.String(http.StatusInternalServerError, "Unable to generate the university's encoding resource right now. Please try again later.")
		return
	}

	err_encoding_resource_validation := GeneticAlgorithm.ValidateEncodingResource(university_schedules, encoding_resource, curriculums, selected_semester)

	if err_encoding_resource_validation != nil {
		log.Print("GetValidateSchedules [error-encoding-resource-validation]: ", err_encoding_resource_validation)
		ctx.String(http.StatusInternalServerError, "An unknown error occured, please try again later. If this still persist, report to the devs")
		return
	}

	errs_vertical_validation := university_schedules.VerticalValidation(rooms)

	for _, err_vertical_validation := range errs_vertical_validation {
		if err_vertical_validation != nil {
			validation_results = append(validation_results, err_vertical_validation.Error())
		}
	}

	if len(errs_vertical_validation) > 0 {
		ctx.JSON(http.StatusConflict, validation_results)
		return
	}

	errs_horizontal_validation := GeneticAlgorithm.HorizontalValidation(
		university_schedules, curriculums, department_to_horizontal_validate, selected_semester,
	)

	for _, err_horizontal_validation := range errs_horizontal_validation {
		if err_horizontal_validation != nil {
			validation_results = append(validation_results, err_horizontal_validation.Error())
		}
	}

	if len(errs_horizontal_validation) > 0 {
		ctx.JSON(http.StatusConflict, validation_results)
		return
	}

	ctx.JSON(http.StatusOK, validation_results)
}

/*
GET:

	"/estimate_resources?department_id=[N>0]&semester=[0-N>=1]"
*/
func GetEstimates(ctx *gin.Context) {
	selected_semester, is_valid_semester_param := RoutesV1.IsValidParameterSemesterIndex(ctx)

	if !is_valid_semester_param {
		return
	}

	department_id, is_valid_department_id_param := RoutesV1.IsValidParameterDepartmentID(ctx)

	if !is_valid_department_id_param {
		return
	}

	missing_resources, err := GeneticAlgorithm.EstimateResourceAvailability(RouteGlobals.ResourcesPersistence, selected_semester, department_id)

	if err != nil {
		log.Print("GetEstimates: [error-estimation]")
		ctx.String(
			http.StatusInternalServerError,
			"Resource estimation error: %s. Please contact support.",
			err.Error(),
		)
		return
	}

	log.Printf("Missing Resources :\n\n%+v\n\n", missing_resources)

	missing_resources_msg := ""

	// problems

	if missing_resources.InstructorTimeSlot > 0 {

		if len(missing_resources_msg) > 0 {
			missing_resources_msg += ", "
		}

		missing_resources_msg += fmt.Sprintf(
			"there are %d missing instructor(s) availability hours",
			int(float64(missing_resources.InstructorTimeSlot)/Const.N_HOUR_TIME_SLOTS),
		)
	}

	if missing_resources.RoomLecTimeSlot > 0 {

		if len(missing_resources_msg) > 0 {
			missing_resources_msg += ", "
		}

		missing_resources_msg += fmt.Sprintf(
			"there are %d missing LEC room(s) availability hours",
			int(float64(missing_resources.RoomLecTimeSlot)/Const.N_HOUR_TIME_SLOTS),
		)
	}

	if missing_resources.RoomLabTimeSlot > 0 {

		if len(missing_resources_msg) > 0 {
			missing_resources_msg += ", "
		}

		missing_resources_msg += fmt.Sprintf(
			"there are %d missing LAB room(s) availability hours",
			int(float64(missing_resources.RoomLabTimeSlot)/Const.N_HOUR_TIME_SLOTS),
		)
	}

	if missing_resources.RoomGymTimeSlot > 0 {

		if len(missing_resources_msg) > 0 {
			missing_resources_msg += ", "
		}

		missing_resources_msg += fmt.Sprintf(
			"there are %d missing GYM room(s) availability hours",
			int(float64(missing_resources.RoomGymTimeSlot)/Const.N_HOUR_TIME_SLOTS),
		)
	}

	// possible solutions

	if len(missing_resources_msg) > 0 {
		missing_resources_msg +=
			", we would recommend the department the following options to fix this limited resource problem; " +
				"reduce assigned subject(s) contact hours in the curriculums of the department, " +
				"reduce the number of sections in the curriculums of the department"
	}

	if missing_resources.InstructorTimeSlot > 0 {

		if len(missing_resources_msg) > 0 {
			missing_resources_msg += ", "
		}

		missing_resources_msg += fmt.Sprintf(
			`add more instructor(s) with %d hour(s) of duty,
			 or enable more available time slots for the existing instructor(s)`,
			int(float64(missing_resources.InstructorTimeSlot)/Const.N_HOUR_TIME_SLOTS),
		)
	}

	if missing_resources.RoomLecTimeSlot > 0 {

		if len(missing_resources_msg) > 0 {
			missing_resources_msg += ", "
		}

		missing_resources_msg += "add more LECTURE room(s), or increase one or more LECTURE room's class/section capacity"

	}

	if missing_resources.RoomLabTimeSlot > 0 {

		if len(missing_resources_msg) > 0 {
			missing_resources_msg += ", "
		}

		missing_resources_msg += "add more LAB room(s), or increase one or more LAB room's class/section capacity"
	}

	if missing_resources.RoomGymTimeSlot > 0 {

		if len(missing_resources_msg) > 0 {
			missing_resources_msg += ", "
		}

		missing_resources_msg += "add more GYM room(s), or increase one or more GYM room's class/section capacity"
	}

	if len(missing_resources_msg) > 0 {
		missing_resources_msg = "the system estimated that, " + missing_resources_msg
		ctx.String(http.StatusOK, missing_resources_msg)
		return
	}

	ctx.String(http.StatusOK, "the system estimated that there are enough resources for this department semester")
}
