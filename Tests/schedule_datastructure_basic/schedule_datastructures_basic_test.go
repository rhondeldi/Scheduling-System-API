package schedule_datastructures_basic_test

import (
	"testing"

	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

func TestScheduleTypes1(t *testing.T) {

	universitySchedules := Schedule.NewUniTimeTables(1)

	universitySchedules[0][0][1].Set(222, 333, 444)

	testSectionSchedule := universitySchedules.GetWeekTimeTable(0)

	testDaySchedule := testSectionSchedule.GetDayTimeTable(0)

	testTimeSlot := testDaySchedule.GetTimeSlot(1)

	if universitySchedules[0][0][1].GetInstructorID() != 333 {
		t.Error("uniSectionSchedules[0][0][1].GetInstructorID() != 333")
	}

	if universitySchedules[0][0][1].GetRoomID() != 444 {
		t.Error("uniSectionSchedules[0][0][1].GetRoomID() != 444")
	}

	if universitySchedules[0][0][1].GetSubjectID() != 222 {
		t.Error("uniSectionSchedules[0][0][1].GetSubjectID() != 222")
	}

	testTimeSlot.SetInstructorID(23423)
	testTimeSlot.SetRoomID(12122)
	testTimeSlot.SetSubjectID(9898)

	if universitySchedules[0][0][1].GetInstructorID() != 23423 {
		t.Error("uniSectionSchedules[0][0][1].GetInstructorID() != 23423")
	}

	if universitySchedules[0][0][1].GetRoomID() != 12122 {
		t.Error("uniSectionSchedules[0][0][1].GetRoomID() != 12122")
	}

	if universitySchedules[0][0][1].GetSubjectID() != 9898 {
		t.Error("uniSectionSchedules[0][0][1].GetSubjectID() != 9898")
	}

	if universitySchedules[0][0][1].GetInstructorID() != 23423 {
		t.Error("uniSectionSchedules[0][0][1].GetInstructorID() != 23423")
	}

	if universitySchedules[0][0][1].GetRoomID() != 12122 {
		t.Error("uniSectionSchedules[0][0][1].GetRoomID() != 12122")
	}

	if universitySchedules[0][0][1].GetSubjectID() != 9898 {
		t.Error("uniSectionSchedules[0][0][1].GetSubjectID() != 9898")
	}
}

func TestScheduleTypes1_5(t *testing.T) {

	universitySchedules := Schedule.NewUniTimeTables(1)

	universitySchedules[0][0][1].SetSubjectID(222)
	universitySchedules[0][0][1].SetInstructorID(333)
	universitySchedules[0][0][1].SetRoomID(444)

	testSectionSchedule := universitySchedules.GetWeekTimeTable(0)

	testDaySchedule := testSectionSchedule.GetDayTimeTable(0)

	testTimeSlot := testDaySchedule.GetTimeSlot(1)

	if universitySchedules[0][0][1].GetInstructorID() != 333 {
		t.Error("uniSectionSchedules[0][0][1].GetInstructorID() != 333")
	}

	if universitySchedules[0][0][1].GetRoomID() != 444 {
		t.Error("uniSectionSchedules[0][0][1].GetRoomID() != 444")
	}

	if universitySchedules[0][0][1].GetSubjectID() != 222 {
		t.Error("uniSectionSchedules[0][0][1].GetSubjectID() != 222")
	}

	testTimeSlot.SetInstructorID(23423)
	testTimeSlot.SetRoomID(12122)
	testTimeSlot.SetSubjectID(9898)

	if universitySchedules[0][0][1].GetInstructorID() != 23423 {
		t.Error("uniSectionSchedules[0][0][1].GetInstructorID() != 23423")
	}

	if universitySchedules[0][0][1].GetRoomID() != 12122 {
		t.Error("uniSectionSchedules[0][0][1].GetRoomID() != 12122")
	}

	if universitySchedules[0][0][1].GetSubjectID() != 9898 {
		t.Error("uniSectionSchedules[0][0][1].GetSubjectID() != 9898")
	}

	if universitySchedules[0][0][1].GetInstructorID() != 23423 {
		t.Error("uniSectionSchedules[0][0][1].GetInstructorID() != 23423")
	}

	if universitySchedules[0][0][1].GetRoomID() != 12122 {
		t.Error("uniSectionSchedules[0][0][1].GetRoomID() != 12122")
	}

	if universitySchedules[0][0][1].GetSubjectID() != 9898 {
		t.Error("uniSectionSchedules[0][0][1].GetSubjectID() != 9898")
	}
}

func TestScheduleTypes2(t *testing.T) {

	universitySchedules := Schedule.NewUniTimeTables(1)

	universitySchedules.GetWeekTimeTable(0).GetDayTimeTable(0).GetTimeSlot(1).Set(222, 333, 444)

	testSectionSchedule := universitySchedules.GetWeekTimeTable(0)

	testDaySchedule := testSectionSchedule.GetDayTimeTable(0)

	testTimeSlot := testDaySchedule.GetTimeSlot(1)

	if universitySchedules.GetWeekTimeTable(0).GetDayTimeTable(0).GetTimeSlot(1).GetInstructorID() != 333 {
		t.Error("uniSectionSchedules[0][0][1].GetInstructorID() != 333")
	}

	if universitySchedules.GetWeekTimeTable(0).GetDayTimeTable(0).GetTimeSlot(1).GetRoomID() != 444 {
		t.Error("uniSectionSchedules[0][0][1].GetRoomID() != 444")
	}

	if universitySchedules.GetWeekTimeTable(0).GetDayTimeTable(0).GetTimeSlot(1).GetSubjectID() != 222 {
		t.Error("uniSectionSchedules[0][0][1].GetSubjectID() != 222")
	}

	testTimeSlot.SetInstructorID(23423)
	testTimeSlot.SetRoomID(12122)
	testTimeSlot.SetSubjectID(9898)

	if universitySchedules.GetWeekTimeTable(0).GetDayTimeTable(0).GetTimeSlot(1).GetInstructorID() != 23423 {
		t.Error("uniSectionSchedules[0][0][1].GetInstructorID() != 23423")
	}

	if universitySchedules.GetWeekTimeTable(0).GetDayTimeTable(0).GetTimeSlot(1).GetRoomID() != 12122 {
		t.Error("uniSectionSchedules[0][0][1].GetRoomID() != 12122")
	}

	if universitySchedules.GetWeekTimeTable(0).GetDayTimeTable(0).GetTimeSlot(1).GetSubjectID() != 9898 {
		t.Error("uniSectionSchedules[0][0][1].GetSubjectID() != 9898")
	}

	if universitySchedules.GetWeekTimeTable(0).GetDayTimeTable(0).GetTimeSlot(1).GetInstructorID() != 23423 {
		t.Error("uniSectionSchedules[0][0][1].GetInstructorID() != 23423")
	}

	if universitySchedules.GetWeekTimeTable(0).GetDayTimeTable(0).GetTimeSlot(1).GetRoomID() != 12122 {
		t.Error("uniSectionSchedules[0][0][1].GetRoomID() != 12122")
	}

	if universitySchedules.GetWeekTimeTable(0).GetDayTimeTable(0).GetTimeSlot(1).GetSubjectID() != 9898 {
		t.Error("uniSectionSchedules[0][0][1].GetSubjectID() != 9898")
	}
}
