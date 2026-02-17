package StorageResources_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Departments"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/StorageResources"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

func TestJsonFilePersistence_Update(t *testing.T) {
	TestPersistence.ReaderService = &StorageResources.JsonReader{}
	TestPersistence.WriterService = &StorageResources.JsonWriter{}

	instructors, err := TestPersistence.ReaderService.ReadAllInstructors()

	if err != nil {
		t.Error(err)
	}

	empty_strings := Utils.CheckForEmptyStrings(instructors, "instructors")

	if len(empty_strings) > 0 {
		for _, err := range empty_strings {
			t.Error("empty string :", err)
			fmt.Println()
		}
	}

	backup_instructor := instructors[2]

	instructors[2].FirstName = "Carcharodon"
	instructors[2].LastName = "Astra"

	updated_instructor := instructors[2]

	if backup_instructor == instructors[2] {
		t.Error("that should not be equal anymore")
	}

	err_create := TestPersistence.WriterService.CreateInstructor(instructors[2])

	if err_create == nil {
		t.Error("there should be an error there")
	}

	err_update := TestPersistence.WriterService.UpdateInstructor(instructors[2])

	if err_update != nil {
		t.Error(err_update)
	}

	new_instructor := Instructors.Instructor{
		DepartmentID:  1,
		FirstName:     "Mephiston",
		MiddleInitial: "E",
		LastName:      "Calistarius",
		Time:          [3]uint64{1, 3},
	}

	err_create_2 := TestPersistence.WriterService.CreateInstructor(new_instructor)

	if err_create_2 != nil {
		t.Error(err_create_2)
	}

	instructors, err = TestPersistence.ReaderService.ReadAllInstructors()

	if err != nil {
		t.Error(err)
	}

	has_found_new_instructor := false
	has_updated := false

	for _, instructor := range instructors {
		if instructor.FirstName == new_instructor.FirstName {
			new_instructor.InstructorID = instructor.InstructorID
			if instructor == new_instructor {
				has_found_new_instructor = true
			}
		}

		if updated_instructor == instructor {
			has_updated = true
		}
	}

	if !has_found_new_instructor {
		t.Error("New instructor created earlier not found")
	}

	if !has_updated {
		t.Error("there is no updated instructor")
	}

	fmt.Println("\n\nTotal Number of Instructors : ", len(instructors))
}

func TestJsonFilePersistence_InstructorsCRU(t *testing.T) {
	TestPersistence.ReaderService = &StorageResources.JsonReader{}
	TestPersistence.WriterService = &StorageResources.JsonWriter{}

	// read 1

	all_instructors1, err_read_all_instructors1 := TestPersistence.ReaderService.ReadAllInstructors()

	if err_read_all_instructors1 != nil {
		t.Error(err_read_all_instructors1)
	}

	// fmt.Print("============================================ INSTRUCTOR CRU BEFORE ============================================\n\n")

	// Utils.PrettyPrint(all_instructors1)

	// update

	all_instructors1[2].FirstName = "UpdatedFirstName"
	all_instructors1[2].MiddleInitial = "UpdatedMiddleInitial"
	all_instructors1[2].LastName = "UpdatedLastName"
	all_instructors1[2].DepartmentID = 7

	for i := range all_instructors1[2].Time {
		all_instructors1[2].Time[i] = 143 + uint64(i)
	}

	err_update_instructor := TestPersistence.WriterService.UpdateInstructor(all_instructors1[2])

	if err_update_instructor != nil {
		t.Error(err_update_instructor)
	}

	// create

	new_instructor := Instructors.Instructor{
		DepartmentID:  6,
		FirstName:     "NewFirstName",
		MiddleInitial: "NewMiddleInitial",
		LastName:      "NewLastName",
		Time:          Instructors.InstructorTimeSlotBitMap{},
	}

	for i := range new_instructor.Time {
		new_instructor.Time[i] = 72 + uint64(i)
	}

	err_create_instructor := TestPersistence.WriterService.CreateInstructor(new_instructor)

	if err_create_instructor != nil {
		t.Error(err_create_instructor)
	}

	// read 2

	all_instructors2, err_read_all_instructors2 := TestPersistence.ReaderService.ReadAllInstructors()

	if err_read_all_instructors2 != nil {
		t.Error(err_read_all_instructors1)
	}

	// fmt.Print("============================================ INSTRUCTOR CRU AFTER ============================================\n\n")

	// Utils.PrettyPrint(all_instructors2)

	// fmt.Print("============================================ INSTRUCTOR CRU END ============================================\n\n")

	// read length test

	if len(all_instructors1) == len(all_instructors2) {
		t.Error("the length of instructors from before and after should not be equal")
	}

	// test update

	if all_instructors2[2].FirstName != "UpdatedFirstName" {
		t.Error("wrong `all_instructors2[2].FirstName`")
	}

	if all_instructors2[2].MiddleInitial != "UpdatedMiddleInitial" {
		t.Error("wrong `all_instructors2[2].MiddleInitial`")
	}

	if all_instructors2[2].LastName != "UpdatedLastName" {
		t.Error("wrong `all_instructors2[2].LastName`")
	}

	if all_instructors2[2].DepartmentID != 7 {
		t.Error("wrong `all_instructors2[2].DepartmentID`")
	}

	for i := range all_instructors2[2].Time {
		if all_instructors2[2].Time[i] != (143 + uint64(i)) {
			t.Error("wrong `all_instructors2[2].Time[i]`")
		}
	}

	// test create

	created_new_instructor := all_instructors2[len(all_instructors1)]

	if created_new_instructor.InstructorID != all_instructors1[len(all_instructors1)-1].InstructorID+1 {
		t.Error("wrong new created instructor ID")
	}

	if created_new_instructor.DepartmentID != 6 {
		t.Error("wrong `created_new_instructor.DepartmentID`")
	}
	if created_new_instructor.FirstName != "NewFirstName" {
		t.Error("wrong `created_new_instructor.FirstName`")
	}
	if created_new_instructor.MiddleInitial != "NewMiddleInitial" {
		t.Error("wrong `created_new_instructor.MiddleInitial`")
	}
	if created_new_instructor.LastName != "NewLastName" {
		t.Error("wrong `created_new_instructor.LastName`")
	}

	for i := range new_instructor.Time {
		if created_new_instructor.Time[i] != 72+uint64(i) {
			t.Error("wrong `created_new_instructor.Time[i]`")
		}
	}
}

func TestJsonFilePersistence_DepartmentCRU(t *testing.T) {
	TestPersistence.ReaderService = &StorageResources.JsonReader{}
	TestPersistence.WriterService = &StorageResources.JsonWriter{}

	// read 1

	all_departments1, err_read_all_departments1 := TestPersistence.ReaderService.ReadAllDepartments()

	if err_read_all_departments1 != nil {
		t.Error(err_read_all_departments1)
	}

	// fmt.Print("============================================ DEPARTMENT CRU BEFORE ============================================\n\n")

	// Utils.PrettyPrint(all_departments1)

	// update

	all_departments1[2].Code = "DTEST"
	all_departments1[2].Name = "Department of Testing"

	err_update_department := TestPersistence.WriterService.UpdateDepartment(all_departments1[2])

	if err_update_department != nil {
		t.Error(err_update_department)
	}

	// create

	new_department := Departments.Department{
		Code: "DONTEST",
		Name: "Department of New Testing",
	}

	err_create_department := TestPersistence.WriterService.CreateDepartment(new_department)

	if err_create_department != nil {
		t.Error(err_create_department)
	}

	// read 2

	all_departments2, err_read_all_departments2 := TestPersistence.ReaderService.ReadAllDepartments()

	if err_read_all_departments2 != nil {
		t.Error(err_read_all_departments1)
	}

	// fmt.Print("============================================ DEPARTMENT CRU AFTER ============================================\n\n")

	// Utils.PrettyPrint(all_departments2)

	// fmt.Print("============================================ DEPARTMENT CRU END ============================================\n\n")

	// read length test

	if len(all_departments1) == len(all_departments2) {
		t.Error("the length of instructors from before and after should not be equal")
	}

	// test update

	if all_departments2[2].Code != "DTEST" {
		t.Error("wrong `all_departments2[2].Code`")
	}

	if all_departments2[2].Name != "Department of Testing" {
		t.Error("wrong `all_departments2[2].Name`")
	}

	// test create

	created_new_department := all_departments2[len(all_departments1)]

	if created_new_department.Code != "DONTEST" {
		t.Error("wrong `created_new_department.Code`")
	}

	if created_new_department.Name != "Department of New Testing" {
		t.Error("wrong `created_new_department.Name`")
	}
}

func TestJsonFilePersistence_RoomCRU(t *testing.T) {
	TestPersistence.ReaderService = &StorageResources.JsonReader{}
	TestPersistence.WriterService = &StorageResources.JsonWriter{}

	// read 1

	all_rooms1, err_read_all_rooms1 := TestPersistence.ReaderService.ReadAllRooms()

	if err_read_all_rooms1 != nil {
		t.Error(err_read_all_rooms1)
	}

	// fmt.Print("============================================ ROOM CRU BEFORE ============================================\n\n")

	// Utils.PrettyPrint(all_rooms1)

	// update

	all_rooms1[2].Name = "ADVANCE 77"
	all_rooms1[2].DepartmentID = 7
	all_rooms1[2].Capacity = 7
	all_rooms1[2].RoomType = 1

	err_update_room := TestPersistence.WriterService.UpdateRoom(all_rooms1[2])

	if err_update_room != nil {
		t.Error(err_update_room)
	}

	// create

	new_room := Rooms.Room{
		Name:         "NEW RM 360",
		DepartmentID: 98,
		Capacity:     14,
		RoomType:     0,
	}

	err_create_room := TestPersistence.WriterService.CreateRoom(new_room)

	if err_create_room != nil {
		t.Error(err_create_room)
	}

	// read 2

	all_rooms2, err_read_all_rooms2 := TestPersistence.ReaderService.ReadAllRooms()

	if err_read_all_rooms2 != nil {
		t.Error(err_read_all_rooms1)
	}

	// fmt.Print("============================================ ROOM CRU AFTER ============================================\n\n")

	// Utils.PrettyPrint(all_rooms2)

	// fmt.Print("============================================ ROOM CRU END ============================================\n\n")

	// read length test

	if len(all_rooms1) == len(all_rooms2) {
		t.Error("the length of instructors from before and after should not be equal")
	}

	// test update

	if all_rooms2[2].Name != "ADVANCE 77" {
		t.Error("wrong `all_rooms2[2].Name`")
	}
	if all_rooms2[2].DepartmentID != 7 {
		t.Error("wrong `all_rooms2[2].DepartmentID`")
	}
	if all_rooms2[2].Capacity != 7 {
		t.Error("wrong `all_rooms2[2].Capacity`")
	}
	if all_rooms2[2].RoomType != 1 {
		t.Error("wrong `all_rooms2[2].RoomType`")
	}

	// test create

	created_new_room := all_rooms2[len(all_rooms1)]

	if created_new_room.Name != "NEW RM 360" {
		t.Error("wrong `created_new_room.Name`")
	}
	if created_new_room.DepartmentID != 98 {
		t.Error("wrong `created_new_room.DepartmentID`")
	}
	if created_new_room.Capacity != 14 {
		t.Error("wrong `created_new_room.Capacity`")
	}
	if created_new_room.RoomType != 0 {
		t.Error("wrong `created_new_room.RoomType`")
	}
}

func TestJsonFilePersistence_SubjectCRU(t *testing.T) {
	TestPersistence.ReaderService = &StorageResources.JsonReader{}
	TestPersistence.WriterService = &StorageResources.JsonWriter{}

	// read 1

	all_subjects1, err_read_all_subjects1 := TestPersistence.ReaderService.ReadAllSubjects()

	if err_read_all_subjects1 != nil {
		t.Error(err_read_all_subjects1)
	}

	// fmt.Print("============================================ SUBJECT CRU BEFORE ============================================\n\n")

	// Utils.PrettyPrint(all_subjects1)

	// update

	all_subjects1[2].Code = "PDT 1000"
	all_subjects1[2].Name = "Subject Update 1000"
	all_subjects1[2].LecHours = 1
	all_subjects1[2].LabHours = 2
	all_subjects1[2].BitFlags = ^uint16(0)

	err_update_subject := TestPersistence.WriterService.UpdateSubject(all_subjects1[2])

	if err_update_subject != nil {
		t.Error(err_update_subject)
	}

	// create

	new_subject := Curriculum.Subject{
		Code:     "NW 2000",
		Name:     "New Subject 2000",
		LecHours: 4,
		LabHours: 1,
		BitFlags: 14,
	}

	err_create_subject := TestPersistence.WriterService.CreateSubject(new_subject)

	if err_create_subject != nil {
		t.Error(err_create_subject)
	}

	// read 2

	all_subjects2, err_read_all_subjects2 := TestPersistence.ReaderService.ReadAllSubjects()

	if err_read_all_subjects2 != nil {
		t.Error(err_read_all_subjects1)
	}

	// fmt.Print("============================================ SUBJECT CRU AFTER ============================================\n\n")

	// Utils.PrettyPrint(all_subjects2)

	// fmt.Print("============================================ SUBJECT CRU END ============================================\n\n")

	// read length test

	if len(all_subjects1) == len(all_subjects2) {
		t.Error("the length of instructors from before and after should not be equal")
	}

	// test update

	if all_subjects2[2].Code != "PDT 1000" {
		t.Error("wrong `all_subjects2[2].Code`")
	}

	if all_subjects2[2].Name != "Subject Update 1000" {
		t.Error("wrong `all_subjects2[2].Name`")
	}

	if all_subjects2[2].LecHours != 1 {
		t.Error("wrong `all_subjects2[2].LecHours`")
	}

	if all_subjects2[2].LabHours != 2 {
		t.Error("wrong `all_subjects2[2].LabHours`")
	}

	if all_subjects2[2].BitFlags != ^uint16(0) {
		t.Error("wrong `all_subjects2[2].BitFlags`")
	}

	// test create

	created_new_subject := all_subjects2[len(all_subjects1)]

	if created_new_subject.Code != "NW 2000" {
		t.Error("wrong `created_new_subject.Code`")
	}
	if created_new_subject.Name != "New Subject 2000" {
		t.Error("wrong `created_new_subject.Name`")
	}
	if created_new_subject.LecHours != 4 {
		t.Error("wrong `created_new_subject.LecHours`")
	}
	if created_new_subject.LabHours != 1 {
		t.Error("wrong `created_new_subject.LabHours`")
	}
	if created_new_subject.BitFlags != 14 {
		t.Error("wrong `created_new_subject.BitFlags`")
	}
}

func TestJsonFilePersistence_CurriculumCRU(t *testing.T) {
	TestPersistence.ReaderService = &StorageResources.JsonReader{}
	TestPersistence.WriterService = &StorageResources.JsonWriter{}

	// read 1

	all_curriculums1, err_read_all_curriculums1 := TestPersistence.ReaderService.ReadAllCurriculum()

	if err_read_all_curriculums1 != nil {
		t.Fatal(err_read_all_curriculums1)
	}

	// fmt.Print("============================================ CURRICULUMS CRU BEFORE ============================================\n\n")

	all_curriculums1[2].CurriculumCode = "BS EMT"
	all_curriculums1[2].CurriculumName = "Bachelor of Science in Electro-Mechanical Technology"
	all_curriculums1[2].DepartmentID = 5
	all_curriculums1[2].YearLevels[0].Semesters[0].Sections = 8

	// this subject will not auto generate an ID in the last comparison test, so we need to
	// manually assign the current correct subject ID for the associated subject Code.
	all_curriculums1[2].YearLevels[0].Semesters[0].Subjects = []Curriculum.Subject{{
		ID:                    239,
		Code:                  "ITEC 50",
		Name:                  "Web Systems and Technologies",
		LecHours:              3,
		LabHours:              1,
		BitFlags:              0,
		DesignatedInstructors: []uint16{76},
	}}

	err_update_curriculum := TestPersistence.WriterService.UpdateCurriculum(all_curriculums1[2])

	if err_update_curriculum != nil {
		t.Error(err_update_curriculum)
	}

	// create

	new_curriculum := Curriculum.Curriculum{
		CurriculumCode: "BS Aerospace",
		CurriculumName: "Bachelor of Aerospace Engineering",
		DepartmentID:   5,
		YearLevels: []Curriculum.YearLevel{{
			Name:     "1st Year",
			IsActive: true,
			Semesters: []Curriculum.Semester{{
				Name:     "1st Semester",
				Sections: 4,
				Subjects: []Curriculum.Subject{{
					ID:                    225,
					Code:                  "ITEC 55",
					Name:                  "Platform Technologies",
					LecHours:              3,
					LabHours:              5,
					BitFlags:              0,
					DesignatedInstructors: []uint16{76, 80, 86},
				}},
			}},
		}},
	}

	id, err_create_curriculum := TestPersistence.WriterService.CreateCurriculum(new_curriculum)

	if err_create_curriculum != nil {
		t.Error(err_create_curriculum)
	}

	if id == 0 {
		t.Error("new curriculums can't have an ID of 0 and has no error during creation")
	}

	// read 2

	all_curriculum2, err_read_all_curriculum2 := TestPersistence.ReaderService.ReadAllCurriculum()

	if err_read_all_curriculum2 != nil {
		t.Error(err_read_all_curriculums1)
	}

	// fmt.Print("============================================ CURRICULUMS CRU AFTER ============================================\n\n")

	// Utils.PrettyPrint(all_curriculum2)

	// fmt.Print("============================================ CURRICULUMS CRU END ============================================\n\n")

	// read length test

	if len(all_curriculums1) == len(all_curriculum2) {
		t.Error("the length of all_curriculums from before and after should not be equal")
	}

	// test update

	if all_curriculum2[2].CurriculumCode != all_curriculums1[2].CurriculumCode {
		t.Errorf("wrong `all_curriculum2[2].CurriculumCode : (%s)`", all_curriculum2[2].CurriculumCode)
	}

	if all_curriculum2[2].CurriculumName != all_curriculums1[2].CurriculumName {
		t.Errorf("wrong `all_curriculum2[2].CurriculumName : %s`", all_curriculum2[2].CurriculumName)
	}

	if all_curriculum2[2].DepartmentID != all_curriculums1[2].DepartmentID {
		t.Errorf("wrong `all_curriculum2[2].DepartmentID : %d`", all_curriculum2[2].DepartmentID)
	}

	if all_curriculum2[2].CurriculumID != all_curriculums1[2].CurriculumID {
		t.Errorf("wrong `all_curriculum2[2].CurriculumID : %d`", all_curriculum2[2].CurriculumID)
	}

	if !reflect.DeepEqual(all_curriculum2[2].YearLevels, all_curriculums1[2].YearLevels) {
		t.Error("wrong `all_curriculum2[2].YearLevels...`")
		fmt.Printf("================================= all_curriculum2[2].YearLevels ==================================\n")
		Utils.PrettyPrint(all_curriculum2[2].YearLevels)
		fmt.Printf("================================= all_curriculums1[2].YearLevels =================================\n")
		Utils.PrettyPrint(all_curriculums1[2].YearLevels)
		fmt.Printf("==================================================================================================\n")
	}

	// test create

	created_new_curriculum := all_curriculum2[len(all_curriculums1)]

	if reflect.DeepEqual(created_new_curriculum, new_curriculum) {
		t.Error("wrong `reflect.DeepEqual(created_new_curriculum, new_curriculum)`")
	}
}
