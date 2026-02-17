package RoutesV1

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/RouteGlobals"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

type RoomTablePage struct {
	Rooms      []Rooms.Room `json:"Rooms"`
	TotalRooms int          `json:"TotalRooms"`
}

/*
GET:

	"/rooms?department_id=D&page_size=[N>0]&page[0-N>0]&name_match=[string]"
*/
func GetDepartmentRooms(ctx *gin.Context) {
	department_id, is_valid_department_id_param := IsValidParameterDepartmentID(ctx)
	if !is_valid_department_id_param {
		return
	}

	page_size, is_valid_page_size_param := IsValidPageSize(ctx)
	if !is_valid_page_size_param {
		return
	}

	page, is_valid_page_param := IsValidPage(ctx)
	if !is_valid_page_param {
		return
	}

	name_parameter := ctx.Query("name_match")

	all_rooms, err_read_all_rooms := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllRooms()

	if err_read_all_rooms != nil {
		ctx.String(http.StatusInternalServerError, "we're currently unable to get the rooms")
		return
	}

	department_rooms_page := make([]Rooms.Room, 0)
	total_department_rooms := 0

	for _, room := range all_rooms {
		if room.DepartmentID != uint16(department_id) {
			continue
		}

		if len(name_parameter) > 0 {
			if !Utils.HasSubString(room.Name, name_parameter) {
				continue
			}
		}

		total_department_rooms++

		if (total_department_rooms - 1) < (page_size * page) {
			continue
		}

		if len(department_rooms_page) < page_size {
			department_rooms_page = append(department_rooms_page, room)
		}
	}

	room_table_page := &RoomTablePage{
		Rooms:      department_rooms_page,
		TotalRooms: total_department_rooms,
	}

	ctx.JSON(http.StatusOK, room_table_page)
}

type RoomSubjectAssignmentInfo struct {
	SubjectCode      string `json:"SubjectCode"`
	CourseSection    string `json:"CourseSection"`
	InstructorName   string `json:"InstructorName"`
	DayIdx           uint8  `json:"DayIdx"`
	TimeSlotIdx      uint8  `json:"TimeSlotIdx"`
	SubjectTimeSlots uint8  `json:"SubjectTimeSlots"`
}

/*
GET:

	"/room_allocation?room_id=[N>0]"
*/
func GetRoomSubjectAssignment(ctx *gin.Context) {
	room_id, is_valid_room_id := IsValidRoomID(ctx)
	if !is_valid_room_id {
		return
	}

	target_room, err_read_room := RouteGlobals.ResourcesPersistence.ReaderService.ReadRoom(uint16(room_id))

	if err_read_room != nil {
		log.Printf("unable to read the room id %d, caused by : %s", room_id, err_read_room.Error())
		ctx.String(http.StatusInternalServerError, "we're unable to read that room right now, please try again later")
		return
	}

	if target_room.Capacity != 1 {
		ctx.String(http.StatusConflict, "The frontend does not support visualizing a week timetable schedule for any room whose section capacity is greater than 1")
		return
	}

	all_curriculums, err_read_all_curriculums := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllCurriculum()

	if err_read_all_curriculums != nil {
		ctx.String(http.StatusInternalServerError, "we are unable to get the curriculums needed to generate the instructor timeslot availability")
		return
	}

	semesters_sub_assign := make([][]RoomSubjectAssignmentInfo, 0)

	for semester_idx := range Curriculum.SUPPORTED_SEMESTERS {
		university_schedules, has_obtained := ObtainUniversityScheduleNoHorizontalValidation(ctx, semester_idx)

		if !has_obtained {
			return
		}

		err_set_cache := RouteGlobals.SetCachedUniversitySchedule(semester_idx, university_schedules)

		if err_set_cache != nil {
			log.Println(err_set_cache.Error())
		}

		room_allocation, err_get_room_allocation := get_room_time_allocation(*target_room, university_schedules, all_curriculums, semester_idx)

		if err_get_room_allocation != nil {
			ctx.String(http.StatusInternalServerError, fmt.Sprintf("we are unable to read the room time allocation for the %s", Curriculum.SEMESTER_INDEX_NAME[semester_idx]))
			return
		}

		semesters_sub_assign = append(semesters_sub_assign, room_allocation)
	}

	ctx.JSON(http.StatusOK, semesters_sub_assign)
}

func get_room_time_allocation(target_room Rooms.Room, university_schedules Schedule.UniTimeTables, all_curriculums []Curriculum.Curriculum, selected_semester int) ([]RoomSubjectAssignmentInfo, error) {

	sub_id_to_subject_code := make(map[uint16]string)

	{ // subjects
		subjects, err_read_all_subjects := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllSubjects()

		if err_read_all_subjects != nil {
			return nil, errors.New("we can not retrieve the subjects information right now")
		}

		for _, subject := range subjects {
			sub_id_to_subject_code[subject.ID] = subject.Code
		}
	}

	instructor_id_to_instructor_name := make(map[uint16]string)

	{ // instructors
		instructors, err_read_all_instructors := RouteGlobals.ResourcesPersistence.ReaderService.ReadAllInstructors()

		if err_read_all_instructors != nil {
			return nil, errors.New("we can not retrieve the instructors information right now")
		}

		if target_room.DepartmentID != 0 {
			// if not general room, only include the general instructors and the department instructors
			for _, instructor := range instructors {
				if instructor.DepartmentID == target_room.DepartmentID || instructor.DepartmentID == 0 {
					instructor_id_to_instructor_name[instructor.InstructorID] = instructor.LastName
				}
			}
		} else {
			// if general room, include all instructors
			for _, instructor := range instructors {
				instructor_id_to_instructor_name[instructor.InstructorID] = instructor.LastName
			}
		}
	}

	////////////////

	sub_assign_info := make([]RoomSubjectAssignmentInfo, 0)

	GeneticAlgorithm.IterateSectionsWeekSchedule(university_schedules, all_curriculums, selected_semester, nil, nil, func(indicies GeneticAlgorithm.IterIndices, values GeneticAlgorithm.IterValues) GeneticAlgorithm.IterReturnType {

		for day := range Const.N_WEEKLY_SCHOOL_DAYS {
			for time_slot := 0; time_slot < Const.N_DAILY_TIME_SLOTS; time_slot++ {

				subject_id := university_schedules[indicies.Usi][day].GetTimeSlot(time_slot).GetSubjectID()
				room_id := university_schedules[indicies.Usi][day].GetTimeSlot(time_slot).GetRoomID()

				if subject_id != 0 && room_id == target_room.RoomID {
					if target_room.RoomID == 0 {
						log.Panic("there should be a room allocation here, why there is none?")
					}

					instructor_id := university_schedules[indicies.Usi][day].GetTimeSlot(time_slot).GetInstructorID()

					new_sub_assignment := RoomSubjectAssignmentInfo{
						SubjectCode:      sub_id_to_subject_code[subject_id],
						CourseSection:    fmt.Sprintf("%s-%d%s", values.Curriculum.CurriculumCode, indicies.YearLevel+1, Curriculum.SECTION[indicies.Section]),
						InstructorName:   instructor_id_to_instructor_name[instructor_id],
						DayIdx:           uint8(day),
						TimeSlotIdx:      uint8(time_slot),
						SubjectTimeSlots: 1,
					}

					for forward_time_slot := time_slot + 1; forward_time_slot < Const.N_DAILY_TIME_SLOTS; forward_time_slot++ {
						forward_slot := university_schedules[indicies.Usi][day].GetTimeSlot(forward_time_slot)

						if forward_slot.GetSubjectID() == subject_id && forward_slot.GetInstructorID() == instructor_id && forward_slot.GetRoomID() == room_id {
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
			} // ------------- end of time_slot loop -------------
		} // ------------- end of day loop -------------

		return GeneticAlgorithm.IterProceed
	})

	return sub_assign_info, nil
}
