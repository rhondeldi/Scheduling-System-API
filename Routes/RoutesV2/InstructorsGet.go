package RoutesV2

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Routes/RoutesV1"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

type InstructorTableItem struct {
	InstructorID  uint16 `json:"InstructorID"`
	DepartmentID  uint16 `json:"DepartmentID"`
	FirstName     string `json:"FirstName"`
	MiddleInitial string `json:"MiddleInitial"`
	LastName      string `json:"LastName"`
}

type InstructorTablePage struct {
	Instructors      []InstructorTableItem `json:"Instructors"`
	TotalInstructors int                   `json:"TotalInstructors"`
}

/*
GET:

	"/instructors?
		department_id=D&
		page_size=[N>0]&
		page[0-N>=1]&
		firstname_match=<string>&
		initial_match=<string>&
		lastname_match=<string>
	"
*/
func GetDepartmentInstructors(ctx *gin.Context) {
	department_id, is_valid_department_id_param := RoutesV1.IsValidParameterDepartmentID(ctx)
	if !is_valid_department_id_param {
		return
	}

	page_size, is_valid_page_size_param := RoutesV1.IsValidPageSize(ctx)
	if !is_valid_page_size_param {
		return
	}

	page, is_valid_page_param := RoutesV1.IsValidPage(ctx)
	if !is_valid_page_param {
		return
	}

	firstname_match := ctx.Query("firstname_match")
	initial_match := ctx.Query("initial_match")
	lastname_match := ctx.Query("lastname_match")

	all_instructors, err_read_all_instructors := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllInstructors()

	if err_read_all_instructors != nil {
		log.Println(err_read_all_instructors)
		ctx.String(http.StatusInternalServerError, "we are unable to retrieve the instructors right now")
		return
	}

	department_instructors_page := make([]InstructorTableItem, 0)

	total_instructors := 0

	for _, instructor := range all_instructors {
		if int(instructor.DepartmentID) != department_id {
			continue
		}

		if firstname_match != "" && !Utils.HasSubString(instructor.FirstName, firstname_match) {
			continue
		}

		if initial_match != "" && !Utils.HasSubString(instructor.MiddleInitial, initial_match) {
			continue
		}

		if lastname_match != "" && !Utils.HasSubString(instructor.LastName, lastname_match) {
			continue
		}

		total_instructors++

		if (total_instructors - 1) < (page_size * page) {
			continue
		}

		if len(department_instructors_page) < page_size {
			department_instructors_page = append(department_instructors_page, InstructorTableItem{
				InstructorID:  instructor.InstructorID,
				DepartmentID:  instructor.DepartmentID,
				FirstName:     instructor.FirstName,
				MiddleInitial: instructor.MiddleInitial,
				LastName:      instructor.LastName,
			})
		}
	}

	instructor_table_page := &InstructorTablePage{
		Instructors:      department_instructors_page,
		TotalInstructors: total_instructors,
	}

	ctx.JSON(http.StatusOK, instructor_table_page)
}

/*
GET:

	"/instructor_resources?instructor_id=[N>0]"
*/
func GetInstructorResource(ctx *gin.Context) {
	instructor_id, is_valid_instructor_id_param := RoutesV1.IsValidInstructorID(ctx)

	if !is_valid_instructor_id_param {
		return
	}

	selected_instructor_base, err_read_instructor := RouteGlobals.ResourcesPersistence.ReaderService.ReadInstructor(uint16(instructor_id))

	if err_read_instructor != nil {
		log.Printf("error in GetInstructorResource > ReadInstructor: %s", err_read_instructor.Error())
		ctx.String(http.StatusInternalServerError, "we are unable to retrieve the instructors right now")
		return
	}

	if selected_instructor_base == nil {
		ctx.String(http.StatusNotFound, "that instructor does not exist")
		return
	}

	all_curriculums, err_read_all_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_read_all_curriculums != nil {
		ctx.String(http.StatusInternalServerError, "we are unable to get the curriculums needed to generate the instructor timeslot availability")
		return
	}

	response_body := make(map[string]any, 0)
	response_body["base_time_slots"] = selected_instructor_base.Time.Stringify()

	semesters_time_slots := make([][]string, 0)
	semesters_sub_assign := make([][]InstructorSubjectAssignmentInfo, 0)
	semesters_async_assign := make([][]Schedule.AsyncScheduleRecord, 0)
	semesters_hour_totals := make([]InstructorSemesterHourTotals, 0)

	for semester_idx := range Curriculum.SUPPORTED_SEMESTERS {
		university_schedules, has_obtained := RoutesV1.ObtainUniversityScheduleNoHorizontalValidation(ctx, semester_idx)

		if !has_obtained {
			return
		}

		err_set_cache := RouteGlobals.SetCachedUniversitySchedule(semester_idx, university_schedules)

		if err_set_cache != nil {
			log.Println(err_set_cache.Error())
		}

		instructor_time_allocation, sub_assign, err_get_instructor_time_allocation := get_instructor_time_allocation(
			*selected_instructor_base,
			university_schedules, all_curriculums,
			semester_idx,
		)

		if err_get_instructor_time_allocation != nil {
			ctx.String(http.StatusInternalServerError, fmt.Sprintf("we are unable to recreated the instructor time allocation for the %s", Curriculum.SEMESTER_INDEX_NAME[semester_idx]))
			return
		}

		semesters_time_slots = append(semesters_time_slots, instructor_time_allocation.Stringify())
		semesters_sub_assign = append(semesters_sub_assign, sub_assign)

		async_assign, async_hours, err_async_assign := get_instructor_async_assignments(selected_instructor_base.InstructorID, semester_idx)
		if err_async_assign != nil {
			ctx.String(http.StatusInternalServerError, "we are unable to retrieve instructor asynchronous schedule information right now")
			return
		}
		semesters_async_assign = append(semesters_async_assign, async_assign)

		sync_hours := 0.0
		for _, item := range sub_assign {
			sync_hours += (float64(item.SubjectTimeSlots) / float64(Const.N_HOUR_TIME_SLOTS))
		}

		semesters_hour_totals = append(semesters_hour_totals, InstructorSemesterHourTotals{
			SyncHours:  sync_hours,
			AsyncHours: async_hours,
			TotalHours: sync_hours + async_hours,
		})
	}

	response_body["semesters_time_slots"] = semesters_time_slots
	response_body["semesters_sub_assign"] = semesters_sub_assign
	response_body["semesters_async_assign"] = semesters_async_assign
	response_body["semesters_hour_totals"] = semesters_hour_totals

	ctx.JSON(http.StatusOK, response_body)
}

type InstructorSubjectAssignmentInfo struct {
	SubjectCode      string `json:"SubjectCode"`
	CourseSection    string `json:"CourseSection"`
	RoomName         string `json:"RoomName"`
	DayIdx           uint8  `json:"DayIdx"`
	TimeSlotIdx      uint8  `json:"TimeSlotIdx"`
	SubjectTimeSlots uint8  `json:"SubjectTimeSlots"`
}

type InstructorSemesterHourTotals struct {
	SyncHours  float64 `json:"SyncHours"`
	AsyncHours float64 `json:"AsyncHours"`
	TotalHours float64 `json:"TotalHours"`
}

func get_instructor_time_allocation(base_instructor Instructors.Instructor, university_schedules Schedule.UniTimeTables, all_curriculums []Curriculum.Curriculum, selected_semester int) (Instructors.InstructorTimeSlotBitMap, []InstructorSubjectAssignmentInfo, error) {

	sub_id_to_subject_code := make(map[uint16]string)

	{ // subjects
		subjects, err_read_all_subjects := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllSubjects()

		if err_read_all_subjects != nil {
			return base_instructor.Time, nil, errors.New("we can not retrieve the subjects information right now")
		}

		for _, subject := range subjects {
			sub_id_to_subject_code[subject.ID] = subject.Code
		}
	}

	room_id_to_room_name := make(map[uint16]string)

	{ // rooms
		rooms, err_read_all_rooms := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllRooms()

		if err_read_all_rooms != nil {
			return base_instructor.Time, nil, errors.New("we can not retrieve the rooms information right now")
		}

		if base_instructor.DepartmentID != 0 {
			// if not general instructor, only include the general rooms and the department rooms
			for _, room := range rooms {
				if room.DepartmentID == base_instructor.DepartmentID || room.DepartmentID == 0 {
					room_id_to_room_name[room.RoomID] = room.Name
				}
			}
		} else {
			// if general instructor, include all rooms
			for _, room := range rooms {
				room_id_to_room_name[room.RoomID] = room.Name
			}
		}
	}

	////////////////

	sub_assign_info := make([]InstructorSubjectAssignmentInfo, 0)

	GeneticAlgorithm.IterateSectionsWeekSchedule(university_schedules, all_curriculums, selected_semester, nil, nil, func(indicies GeneticAlgorithm.IterIndices, values GeneticAlgorithm.IterValues) GeneticAlgorithm.IterReturnType {

		for day := range Const.N_WEEKLY_SCHOOL_DAYS {
			for time_slot := 0; time_slot < Const.N_DAILY_TIME_SLOTS; time_slot++ {

				subject_id := university_schedules[indicies.Usi][day].GetTimeSlot(time_slot).GetSubjectID()
				instructor_id := university_schedules[indicies.Usi][day].GetTimeSlot(time_slot).GetInstructorID()

				if subject_id != 0 && instructor_id == base_instructor.InstructorID {
					if base_instructor.InstructorID == 0 {
						log.Panic("there should be an instructor allocation here, why there is none?")
					}

					base_instructor.Time.SetAvailability(false, day, time_slot)

					room_id := university_schedules[indicies.Usi][day].GetTimeSlot(time_slot).GetRoomID()

					new_sub_assignment := InstructorSubjectAssignmentInfo{
						SubjectCode:      sub_id_to_subject_code[subject_id],
						CourseSection:    fmt.Sprintf("%s-%d%s", values.Curriculum.CurriculumCode, indicies.YearLevel+1, Curriculum.SECTION[indicies.Section]),
						RoomName:         room_id_to_room_name[room_id],
						DayIdx:           uint8(day),
						TimeSlotIdx:      uint8(time_slot),
						SubjectTimeSlots: 1,
					}

					for forward_time_slot := time_slot + 1; forward_time_slot < Const.N_DAILY_TIME_SLOTS; forward_time_slot++ {
						forward_slot := university_schedules[indicies.Usi][day].GetTimeSlot(forward_time_slot)

						if forward_slot.GetSubjectID() == subject_id && forward_slot.GetInstructorID() == instructor_id && forward_slot.GetRoomID() == room_id {
							new_sub_assignment.SubjectTimeSlots++
							base_instructor.Time.SetAvailability(false, day, forward_time_slot)
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
			} // ------------- end of time_slot loop -------------
		} // ------------- end of day loop -------------

		return GeneticAlgorithm.IterProceed
	})

	return base_instructor.Time, sub_assign_info, nil
}

func get_instructor_async_assignments(instructor_id uint16, semester int) ([]Schedule.AsyncScheduleRecord, float64, error) {
	departments, err_read_departments := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllDepartments()
	if err_read_departments != nil {
		return nil, 0, err_read_departments
	}

	all_curriculums, err_read_all_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()
	if err_read_all_curriculums != nil {
		return nil, 0, err_read_all_curriculums
	}

	filtered_records := make([]Schedule.AsyncScheduleRecord, 0)
	total_async_hours := 0.0

	for _, department := range departments {
		records, err_read_records := RouteGlobals.ResourcesPersistence.ReaderService.ReadAsyncScheduleRecords(department.DepartmentID, semester)
		if err_read_records != nil {
			return nil, 0, err_read_records
		}

		records = RoutesV1.BackfillAsyncRecordCourseSection(records, all_curriculums)

		for _, record := range records {
			if record.InstructorID != instructor_id {
				continue
			}

			filtered_records = append(filtered_records, record)
			total_async_hours += record.AsyncHours
		}
	}

	return filtered_records, total_async_hours, nil
}

/*
GET:

	"/instructor_basic?instructor_id=[N>0]"
*/
func GetInstructorBasic(ctx *gin.Context) {
	instructor_id, is_valid_instructor_id_param := RoutesV1.IsValidInstructorID(ctx)
	if !is_valid_instructor_id_param {
		return
	}

	all_instructors, err_read_all_instructors := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllInstructors()

	if err_read_all_instructors != nil {
		log.Println(err_read_all_instructors)
		ctx.String(http.StatusInternalServerError, "we are unable to retrieve the instructors right now")
		return
	}

	var selected_instructor_base *Instructors.Instructor

	for _, instructor := range all_instructors {
		if instructor.InstructorID == uint16(instructor_id) {
			selected_instructor_base = &instructor
			break
		}
	}

	if selected_instructor_base == nil {
		ctx.String(http.StatusNotFound, "that instructor does not exist")
		return
	}

	ctx.JSON(http.StatusOK, selected_instructor_base)
}
