package Schedule_test

import (
	"os"
	"strings"
	"testing"

	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"github.com/mrdcvlsc/scheduling-system-backend/StorageResources"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

func Test_UniTimeTablesSerializationAndDeserialization(t *testing.T) {

	var wrote_sched Schedule.UniTimeTables

	storage_persistence := StorageResources.Persistence{ReaderService: &StorageResources.JsonReader{}}
	resource_persistence := StorageResources.Persistence{ReaderService: &StorageResources.JsonReader{}}

	////////////////////////////////////////////////////////////////////////////////////////

	departments, err_read_all_departments := resource_persistence.ReaderService.ReadAllDepartments()

	if err_read_all_departments != nil {
		t.Fatal(err_read_all_departments)
	}

	rooms, err_read_all_rooms := resource_persistence.ReaderService.ReadAllRooms()

	if err_read_all_rooms != nil {
		t.Fatal(err_read_all_rooms)
	}

	curriculums, err_read_all_curriculum := storage_persistence.ReaderService.ReadAllCurriculum()

	if err_read_all_curriculum != nil {
		t.Fatal(err_read_all_curriculum)
	}

	dept_id_to_department := GeneticAlgorithm.GenerateMapDeptIdToDepartment(departments)

	////////////////////////////////////////////////////////////////////////////////////////

	default_encoding_resource, err_read_default_encoding_resource := GeneticAlgorithm.ReadDefaultEncodingResource(&resource_persistence)

	if err_read_default_encoding_resource != nil {
		t.Fatal(err_read_default_encoding_resource)
	}

	////////////////////////////////////////////////////////////////////////////////////////

	{
		empty_university_schedule := GeneticAlgorithm.NewEmptyIndividual(curriculums, GeneticAlgorithm.TERM_1ST_SEMESTER)

		uni_sched, encoding_resource_output, err := GeneticAlgorithm.EncodeIndividualGenome(
			empty_university_schedule,
			curriculums, dept_id_to_department,
			default_encoding_resource, nil,
			GeneticAlgorithm.TERM_1ST_SEMESTER, 0,
		)

		if uni_sched == nil && err != nil {
			t.Fatalf("there was an error reading data: %v\n", err)
		}

		if uni_sched.IsEmpty() {
			t.Fatal("there was no schedule generated to be tested")
		}

		errs_slice := make([]error, 0)

		errs_slice = append(errs_slice, uni_sched.VerticalValidation(rooms)...)

		if len(errs_slice) > 0 {
			for _, e := range errs_slice {
				t.Error(e)
			}
		}

		if err == nil {
			errs_slice = append(errs_slice,
				GeneticAlgorithm.HorizontalValidation(uni_sched, curriculums, nil, GeneticAlgorithm.TERM_1ST_SEMESTER)...,
			)
		}

		if len(errs_slice) > 0 {
			for _, e := range errs_slice {
				t.Error(e)
			}
		}

		///////////////////////

		generated_encoding_resource, output_resources := GeneticAlgorithm.GenerateEncodingResourceFromUniTimeTable(
			uni_sched, curriculums, GeneticAlgorithm.TERM_1ST_SEMESTER, default_encoding_resource,
		)

		if output_resources != nil {
			t.Fatal(output_resources)
		}

		if encoding_resource_output != nil {
			if !GeneticAlgorithm.IsEqualEncodingResource(generated_encoding_resource, encoding_resource_output) {
				t.Fatal("generated encoding resource from bare university schedule is not equal to the produced encoding resource of GA")
			}
		}

		/////////////////

		serialized_data := Schedule.SerializeUniversitySchedule(uni_sched)

		if err := Utils.SaveToBinFile("tmp-university.schedule", serialized_data); err != nil {
			t.Fatalf("failed to save serialized data to binary file: %v\n", err)
		}

		wrote_sched = uni_sched
	}

	{
		read_bytes, err := Utils.ReadFromBinFile("tmp-university.schedule")

		if err != nil {
			t.Fatalf("failed to read the serialized data from the binary file: %v\n", err)
		}

		deserialize_data := Schedule.DeserializeUniversitySchedule(read_bytes)

		for section_idx, original := range deserialize_data {
			for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
				for time_slot := 0; time_slot < Const.N_DAILY_TIME_SLOTS; time_slot++ {
					if wrote_sched[section_idx][day][time_slot] != original[day][time_slot] {
						t.Errorf("Data missmatch at section index %d, time slot (day: %d, time_slot:%d)\n", section_idx, day, time_slot)
					}
				}
			}
		}
	}

	err := os.Remove("tmp-university.schedule")

	if err != nil {
		t.Fatalf("error deleting file: %v\n", err)
	}
}

func TestVerticalValidation_InstructorWithoutSubject(t *testing.T) {
	persistence := StorageResources.Persistence{ReaderService: &StorageResources.JsonReader{}}

	rooms, err_read_all_rooms := persistence.ReaderService.ReadAllRooms()

	if err_read_all_rooms != nil {
		t.Fatal(err_read_all_rooms)
	}

	uni := Schedule.NewUniTimeTables(1)
	// instructor assigned but no subject
	if err := uni[0].GetDayTimeTable(0).GetTimeSlot(0).Set(0, 42, 0); err != nil {
		t.Fatal(err)
	}
	errs := uni.VerticalValidation(rooms)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if !strings.Contains(errs[0].Error(), "an instructor was assigned, but no subject was scheduled") {
		t.Errorf("unexpected error: %v", errs[0])
	}
}

func TestVerticalValidation_RoomWithoutSubject(t *testing.T) {
	persistence := StorageResources.Persistence{ReaderService: &StorageResources.JsonReader{}}

	rooms, err := persistence.ReaderService.ReadAllRooms()
	if err != nil || len(rooms) == 0 {
		t.Skip("no rooms available for test")
	}
	roomID := rooms[0].RoomID

	uni := Schedule.NewUniTimeTables(1)
	if err := uni[0].GetDayTimeTable(0).GetTimeSlot(0).Set(0, 0, roomID); err != nil {
		t.Fatal(err)
	}
	errs := uni.VerticalValidation(rooms)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if !strings.Contains(errs[0].Error(), "a room was assigned, but no subject was scheduled") {
		t.Errorf("unexpected error: %v", errs[0])
	}
}

func TestVerticalValidation_OverlappingInstructor(t *testing.T) {
	persistence := StorageResources.Persistence{ReaderService: &StorageResources.JsonReader{}}

	rooms, err_read_all_rooms := persistence.ReaderService.ReadAllRooms()

	if err_read_all_rooms != nil {
		t.Fatal(err_read_all_rooms)
	}

	// two sections, same instructor, same slot
	uni := Schedule.NewUniTimeTables(2)
	for i := 0; i < 2; i++ {
		if err := uni[i].GetDayTimeTable(1).GetTimeSlot(2).Set(10, 99, 0); err != nil {
			t.Fatal(err)
		}
	}
	errs := uni.VerticalValidation(rooms)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if !strings.Contains(errs[0].Error(), "overlapping instructor time slot") {
		t.Errorf("unexpected error: %v", errs[0])
	}
}

func TestVerticalValidation_OverlappingRoomWithinAndExceedingCapacity(t *testing.T) {
	persistence := StorageResources.Persistence{ReaderService: &StorageResources.JsonReader{}}
	rooms, err := persistence.ReaderService.ReadAllRooms()
	if err != nil || len(rooms) == 0 {
		t.Skip("no rooms available for test")
	}
	room := rooms[0]
	cap := int(room.Capacity)

	// within capacity
	uniWithin := Schedule.NewUniTimeTables(uint(cap))
	for i := 0; i < cap; i++ {
		if err := uniWithin[i].GetDayTimeTable(2).GetTimeSlot(3).Set(10, 0, room.RoomID); err != nil {
			t.Fatal(err)
		}
	}
	errs := uniWithin.VerticalValidation(rooms)
	if len(errs) != 0 {
		t.Errorf("expected no errors within capacity, got %d", len(errs))
	}

	// exceeding capacity
	uniExceed := Schedule.NewUniTimeTables(uint(cap + 1))
	for i := 0; i < cap+1; i++ {
		if err := uniExceed[i].GetDayTimeTable(2).GetTimeSlot(3).Set(10, 0, room.RoomID); err != nil {
			t.Fatal(err)
		}
	}
	errs2 := uniExceed.VerticalValidation(rooms)
	if len(errs2) != 1 {
		t.Fatalf("expected 1 error exceeding capacity, got %d", len(errs2))
	}
	if !strings.Contains(errs2[0].Error(), "overlapping room time slot") {
		t.Errorf("unexpected error: %v", errs2[0])
	}
}

func TestVerticalValidation_MixedErrors(t *testing.T) {
	persistence := StorageResources.Persistence{ReaderService: &StorageResources.JsonReader{}}
	rooms, err := persistence.ReaderService.ReadAllRooms()
	if err != nil || len(rooms) == 0 {
		t.Skip("no rooms available for test")
	}
	roomID := rooms[0].RoomID

	uni := Schedule.NewUniTimeTables(2)
	// section 0: instructor without subject
	if err := uni[0].GetDayTimeTable(0).GetTimeSlot(0).Set(0, 8, 0); err != nil {
		t.Fatal(err)
	}
	// section 1: overlapping instructor
	if err := uni[1].GetDayTimeTable(0).GetTimeSlot(0).Set(100, 8, roomID); err != nil {
		t.Fatal(err)
	}

	errs := uni.VerticalValidation(rooms)
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(errs))
	}
	if !strings.Contains(errs[0].Error(), "an instructor was assigned, but no subject was scheduled") {
		t.Errorf("unexpected error[0]: %v", errs[0])
	}
	if !strings.Contains(errs[1].Error(), "overlapping instructor time slot") {
		t.Errorf("unexpected error[1]: %v", errs[1])
	}
}

func TestHorizontalValidation_MissingEntireSubject(t *testing.T) {
	persistence := StorageResources.Persistence{ReaderService: &StorageResources.JsonReader{}}

	curriculums, err_read_all_curriculum := persistence.ReaderService.ReadAllCurriculum()

	if err_read_all_curriculum != nil {
		t.Fatal(err_read_all_curriculum)
	}

	currs, err := persistence.ReaderService.ReadAllCurriculum()
	if err != nil {
		t.Fatal(err)
	}
	// find first active semester
	var sem Curriculum.Semester
	var deptID uint16
	for _, c := range currs {
		for _, yl := range c.YearLevels {
			if !yl.IsActive {
				continue
			}
			if len(yl.Semesters) > 0 {
				sem = yl.Semesters[0]
				deptID = c.DepartmentID
				break
			}
		}
		if sem.Name != "" {
			break
		}
	}
	if sem.Name == "" || len(sem.Subjects) == 0 {
		t.Skip("no subjects/semester to test")
	}
	total := Curriculum.GetTotalNumberOfSections(currs, 0)
	uni := Schedule.NewUniTimeTables(uint(total))

	errs := GeneticAlgorithm.HorizontalValidation(uni, curriculums, map[uint16]bool{deptID: true}, 0)

	if len(errs) == 0 {
		t.Fatalf("expected missing‐subject errors, got none")
	}
	foundDetected, foundNotAssigned := false, false
	for _, e := range errs {
		msg := e.Error()
		if strings.Contains(msg, "detected") {
			foundDetected = true
		}
		if strings.Contains(msg, "was not assigned") {
			foundNotAssigned = true
		}
	}
	if !foundDetected || !foundNotAssigned {
		t.Errorf("expected both 'detected' and 'was not assigned' errors, got: %v", errs)
	}
}

func TestHorizontalValidation_TimeSlotAllocationBounds(t *testing.T) {
	persistence := StorageResources.Persistence{ReaderService: &StorageResources.JsonReader{}}
	curriculums, err_read_all_curriculum := persistence.ReaderService.ReadAllCurriculum()

	if err_read_all_curriculum != nil {
		t.Fatal(err_read_all_curriculum)
	}

	currs, err := persistence.ReaderService.ReadAllCurriculum()
	if err != nil {
		t.Fatal(err)
	}
	// pick first subject of first active semester
	var subj Curriculum.Subject
	var deptID uint16
	for _, c := range currs {
		for _, yl := range c.YearLevels {
			if !yl.IsActive {
				continue
			}
			for _, sem := range yl.Semesters {
				if len(sem.Subjects) > 0 {
					subj = sem.Subjects[0]
					deptID = c.DepartmentID
					break
				}
			}
		}
		if subj.ID != 0 {
			break
		}
	}
	if subj.ID == 0 {
		t.Skip("no subject to test")
	}
	slotsRequired := int((subj.LecHours + subj.LabHours) * uint8(Const.N_HOUR_TIME_SLOTS))
	total := Curriculum.GetTotalNumberOfSections(currs, 0)
	uniFew := Schedule.NewUniTimeTables(uint(total))
	// assign fewer slots for section 0
	for i := 0; i < slotsRequired-1 && i < Const.N_DAILY_TIME_SLOTS; i++ {
		uniFew[0].GetDayTimeTable(0).GetTimeSlot(i).Set(subj.ID, 0, 0)
	}

	errsFew := GeneticAlgorithm.HorizontalValidation(uniFew, curriculums, map[uint16]bool{deptID: true}, 0)

	if len(errsFew) == 0 {
		t.Errorf("expected missing time‐slot allocation error, got none")
	}
	// assign extra slots
	uniMany := Schedule.NewUniTimeTables(uint(total))
	for i := 0; i < slotsRequired+1 && i < Const.N_DAILY_TIME_SLOTS; i++ {
		uniMany[0].GetDayTimeTable(0).GetTimeSlot(i).Set(subj.ID, 0, 0)
	}

	errsMany := GeneticAlgorithm.HorizontalValidation(uniMany, curriculums, map[uint16]bool{deptID: true}, 0)

	if len(errsMany) == 0 {
		t.Errorf("expected extra time‐slot allocation error, got none")
	}
}

func TestHorizontalValidation_DepartmentFilterReducesErrors(t *testing.T) {
	persistence := StorageResources.Persistence{ReaderService: &StorageResources.JsonReader{}}

	curriculums, err_read_all_curriculum := persistence.ReaderService.ReadAllCurriculum()

	if err_read_all_curriculum != nil {
		t.Fatal(err_read_all_curriculum)
	}

	currs, err := persistence.ReaderService.ReadAllCurriculum()
	if err != nil {
		t.Fatal(err)
	}
	if len(currs) < 2 {
		t.Skip("not enough departments to test filter")
	}
	// build full schedule with no assignments
	total := Curriculum.GetTotalNumberOfSections(currs, 0)
	uni := Schedule.NewUniTimeTables(uint(total))

	errsNoFilter := GeneticAlgorithm.HorizontalValidation(uni, curriculums, nil, 0)

	// filter only first department

	filter := map[uint16]bool{currs[0].DepartmentID: true}

	errsFilter := GeneticAlgorithm.HorizontalValidation(uni, curriculums, filter, 0)

	if len(errsFilter) > len(errsNoFilter) {
		t.Errorf("expected filter to reduce or equal errors, got %d > %d", len(errsFilter), len(errsNoFilter))
	}
}
