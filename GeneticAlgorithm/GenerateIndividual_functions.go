package GeneticAlgorithm

import (
	"errors"
	"fmt"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Departments"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/StorageResources"
)

func GenerateMapDeptIdToRoomTypeToRooms(rooms []Rooms.Room) map[uint16]map[uint16][]Rooms.Room {
	department_id_to_room_type_to_rooms := make(map[uint16]map[uint16][]Rooms.Room)

	for _, room := range rooms {
		_, has_department_id := department_id_to_room_type_to_rooms[room.DepartmentID]

		if !has_department_id {
			department_id_to_room_type_to_rooms[room.DepartmentID] = make(map[uint16][]Rooms.Room)
		}

		_, has_room_type := department_id_to_room_type_to_rooms[room.DepartmentID][room.RoomType]

		if !has_room_type {
			department_id_to_room_type_to_rooms[room.DepartmentID][room.RoomType] = make([]Rooms.Room, 1, 8)
			department_id_to_room_type_to_rooms[room.DepartmentID][room.RoomType][0] = room
		} else {
			department_id_to_room_type_to_rooms[room.DepartmentID][room.RoomType] = append(
				department_id_to_room_type_to_rooms[room.DepartmentID][room.RoomType], room,
			)
		}
	}

	return department_id_to_room_type_to_rooms
}

func GenerateMapDeptIdToInstructors(instructors []Instructors.Instructor) map[uint16][]Instructors.Instructor {
	department_id_to_instructors := make(map[uint16][]Instructors.Instructor)

	for _, instructor := range instructors {
		_, has_department_id := department_id_to_instructors[instructor.DepartmentID]

		if !has_department_id {
			department_id_to_instructors[instructor.DepartmentID] = make([]Instructors.Instructor, 1, 8)
			department_id_to_instructors[instructor.DepartmentID][0] = instructor
		} else {
			department_id_to_instructors[instructor.DepartmentID] = append(
				department_id_to_instructors[instructor.DepartmentID], instructor,
			)
		}
	}

	return department_id_to_instructors
}

func GenerateMapDeptIdToDepartment(departments []Departments.Department) map[uint16]Departments.Department {
	department_id_to_department := make(map[uint16]Departments.Department)

	for _, department := range departments {
		department_id_to_department[department.DepartmentID] = department
	}

	return department_id_to_department
}

////////////////////////////////////////////////////////////////////////////////////
//              CHECK IF THERE IS ENOUGH INSTRUCTORS FOR THE SCHEDULES
////////////////////////////////////////////////////////////////////////////////////

// The minimum recommended difference between the total available room hours in a department
// and the total lecture hours (or the total laboratory hours) for all subjects in the department.
// This ensures that schedules can be generated with minimal risk of resource shortages.
//
// Tested Values :
//
// * MIN_SUBJECT_ROOM_HOUR_BUFFER = 200, MIN_SUBJECT_INSTRUCTOR_HOUR_BUFFER = 300
//
// : 1024 generations => 65% - 68% valid schedules.
const MIN_SUBJECT_ROOM_HOUR_BUFFER int = 160

// The minimum recommended difference between the total available instructor hours in a department
// and the total lecture and laboratory hours combined for all subjects in the department.
// This ensures that schedules can be generated with minimal risk of resource shortages.
const MIN_SUBJECT_INSTRUCTOR_HOUR_BUFFER int = 264

type Totals struct {
	DepartmentID uint16
	SectionCount int

	LecRoomHours int
	LabRoomHours int
	GymRoomHours int

	InstructorHours float64
	InstructorCount int

	SubjectLecHours int
	SubjectLabHours int
	SubjectGymHours int

	RoomCapacity int
	RoomLabCount int
	RoomLecCount int

	Semester        int
	DepartmentName  string
	Courses         int
	LecSubjectCount int
	LabSubjectCount int
}

type MissingResources struct {
	InstructorTimeSlot int
	RoomLecTimeSlot    int
	RoomLabTimeSlot    int
	RoomGymTimeSlot    int
}

func EstimateResourceAvailability(persistence *StorageResources.Persistence, selected_semester, department_id int) (*MissingResources, error) {

	if department_id == 0 {
		return nil, errors.New("estimating general department is not allowed")
	}

	if department_id < 0 {
		return nil, errors.New("invalid department ID")
	}

	departments, err_department := persistence.ReaderService.ReadAllDepartments()

	if err_department != nil {
		return nil, err_department
	}

	// don't include general department
	departments = departments[1:]

	department_id_to_department := GenerateMapDeptIdToDepartment(departments)

	curriculums, err_curriculum := persistence.ReaderService.ReadAllCurriculum()

	if err_curriculum != nil {
		return nil, err_curriculum
	}

	instructors, err_instructors := persistence.ReaderService.ReadAllInstructors()

	if err_instructors != nil {
		return nil, err_instructors
	}

	rooms, err_rooms := persistence.ReaderService.ReadAllRooms()

	if err_rooms != nil {
		return nil, err_rooms
	}

	total_lec_room_time_slots := 0
	total_lab_room_time_slots := 0
	total_gym_room_time_slots := 0

	for _, room := range rooms {

		is_department_room := room.DepartmentID == uint16(department_id)
		is_general_room := room.DepartmentID == 0

		if !(is_department_room || is_general_room) {
			continue
		}

		total_room_time_slot := int(room.Capacity) * Const.N_WEEKLY_SCHOOL_DAYS * Const.N_DAILY_TIME_SLOTS

		if is_general_room && len(departments) > 1 {
			total_room_time_slot = total_room_time_slot / (len(departments) - 1)
		}

		switch room.RoomType {
		case Rooms.ROOM_TYPE_LEC:
			total_lec_room_time_slots += total_room_time_slot
		case Rooms.ROOM_TYPE_LAB:
			total_lab_room_time_slots += total_room_time_slot
		case Rooms.ROOM_TYPE_GYM:
			total_gym_room_time_slots += total_room_time_slot
		default:
			return nil, fmt.Errorf(
				"detected an invalid room type in the rooms of %s",
				department_id_to_department[room.DepartmentID].Name,
			)
		}
	}

	total_instructor_time_slots := 0

	for _, instructor := range instructors {

		is_department_instructor := instructor.DepartmentID == uint16(department_id)
		is_general_instructor := instructor.DepartmentID == 0

		if !(is_department_instructor || is_general_instructor) {
			continue
		}

		current_instructor_time_slots := 0

		for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
			for time_slot := 0; time_slot < Const.N_DAILY_TIME_SLOTS; time_slot++ {
				if instructor.Time.GetAvailability(day, time_slot) {
					current_instructor_time_slots++
				}
			}
		}

		if is_general_instructor && len(departments) > 1 {
			total_instructor_time_slots += current_instructor_time_slots / (len(departments) - 1)
		} else {
			total_instructor_time_slots += current_instructor_time_slots
		}
	}

	total_sub_lec_time_slots := 0
	total_sub_lab_time_slots := 0
	total_sub_gym_time_slots := 0

	IterateSectionsWeekSchedule(nil, curriculums, selected_semester, nil, func(indicies IterIndices, values IterValues) IterReturnType {
		if values.Curriculum.DepartmentID != uint16(department_id) {
			return IterProceed
		}

		for _, subject := range values.Semester.Subjects {
			lecSlotsPerSection := subject.SlotsToAssignByClassType(0)
			labSlotsPerSection := subject.SlotsToAssignByClassType(1)

			if subject.IsGymType() {
				total_sub_gym_time_slots += ((lecSlotsPerSection + labSlotsPerSection) * values.Semester.Sections)
			} else {
				total_sub_lec_time_slots += (lecSlotsPerSection * values.Semester.Sections)
				total_sub_lab_time_slots += (labSlotsPerSection * values.Semester.Sections)
			}
		}

		return IterProceed
	}, nil)

	missing_resources := &MissingResources{}

	// calculate total subject time slots + buffer

	total_subject_time_slots := total_sub_lec_time_slots + total_sub_lab_time_slots + total_sub_gym_time_slots

	// estimate missing instructor time slots

	total_subject_time_slots_with_buffer := (total_subject_time_slots + (MIN_SUBJECT_INSTRUCTOR_HOUR_BUFFER * Const.N_HOUR_TIME_SLOTS))

	if total_instructor_time_slots < total_subject_time_slots_with_buffer {
		missing_resources.InstructorTimeSlot = total_subject_time_slots_with_buffer - total_instructor_time_slots
	}

	// estimate missing lec rooms time slots

	total_sub_lec_time_slots_with_buffer := (total_sub_lec_time_slots + (MIN_SUBJECT_ROOM_HOUR_BUFFER * Const.N_HOUR_TIME_SLOTS))

	if total_lec_room_time_slots < total_sub_lec_time_slots_with_buffer {
		missing_resources.RoomLecTimeSlot = total_sub_lec_time_slots_with_buffer - total_lec_room_time_slots
	}

	// estimate missing lab rooms time slots

	total_sub_lab_time_slots_with_buffer := (total_sub_lab_time_slots + (MIN_SUBJECT_ROOM_HOUR_BUFFER * Const.N_HOUR_TIME_SLOTS))

	if total_lab_room_time_slots < total_sub_lab_time_slots_with_buffer {
		missing_resources.RoomLabTimeSlot = total_sub_lab_time_slots_with_buffer - total_lab_room_time_slots
	}

	// estimate missing gym rooms time slots

	total_sub_gym_time_slots_with_buffer := (total_sub_gym_time_slots + (MIN_SUBJECT_ROOM_HOUR_BUFFER * Const.N_HOUR_TIME_SLOTS))

	if total_gym_room_time_slots < total_sub_gym_time_slots_with_buffer {
		missing_resources.RoomGymTimeSlot = total_sub_gym_time_slots_with_buffer - total_gym_room_time_slots
	}

	fmt.Printf("=========== DEPARTMENT : %s =========== \n", department_id_to_department[uint16(department_id)].Name)
	fmt.Printf("total            SUBJECT   time slots = %d\n", total_subject_time_slots)
	fmt.Printf("total    Lecture SUBJECT   time slots = %d\n", total_sub_lec_time_slots)
	fmt.Printf("total Laboratory SUBJECT   time slots = %d\n", total_sub_lab_time_slots)
	fmt.Printf("total        Gym SUBJECT   time slots = %d\n", total_sub_gym_time_slots)
	fmt.Printf("total         INSTRUCTOR   time slots = %d\n", total_instructor_time_slots)
	fmt.Printf("total       Lecture ROOM   time slots = %d\n", total_lec_room_time_slots)
	fmt.Printf("total           Lab ROOM   time slots = %d\n", total_lab_room_time_slots)
	fmt.Printf("total           Gym ROOM   time slots = %d\n", total_gym_room_time_slots)

	return missing_resources, nil
}

func GenerateMapInstructorIdToInstructor(persistence *StorageResources.Persistence) (map[uint16]*Instructors.Instructor, error) {
	instructor_id_to_instructor := make(map[uint16]*Instructors.Instructor)

	instructors, err := persistence.ReaderService.ReadAllInstructors()

	if err != nil {
		return nil, err
	}

	for i := range instructors {
		instructor_id_to_instructor[instructors[i].InstructorID] = &instructors[i]
	}

	return instructor_id_to_instructor, nil
}
