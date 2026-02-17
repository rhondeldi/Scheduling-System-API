package Schedule_test

import (
	"reflect"
	"testing"

	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

func TestGetWeekSubjectsJSON(t *testing.T) {
	week := Schedule.WeekTimeTable{}

	week[0][0].Set(1, 101, 201)
	week[0][1].Set(1, 101, 201)

	week[0][3].Set(2, 102, 202)
	week[0][4].Set(2, 102, 202)
	week[0][5].Set(2, 102, 202)

	week[0][7].Set(3, 103, 203)

	week[1][0].Set(4, 104, 204)
	week[1][1].Set(4, 104, 204)

	week[1][3].Set(5, 105, 205)
	week[1][4].Set(5, 105, 205)

	week[5][23].Set(6, 107, 20)

	expected := []Schedule.TimeSlotSubjectJSON{
		{SubjectID: 1, InstructorID: 101, RoomID: 201, Day: 0, StartingTimeSlot: 0, TimeSlotSize: 2},
		{SubjectID: 2, InstructorID: 102, RoomID: 202, Day: 0, StartingTimeSlot: 3, TimeSlotSize: 3},
		{SubjectID: 3, InstructorID: 103, RoomID: 203, Day: 0, StartingTimeSlot: 7, TimeSlotSize: 1},
		{SubjectID: 4, InstructorID: 104, RoomID: 204, Day: 1, StartingTimeSlot: 0, TimeSlotSize: 2},
		{SubjectID: 5, InstructorID: 105, RoomID: 205, Day: 1, StartingTimeSlot: 3, TimeSlotSize: 2},
		{SubjectID: 6, InstructorID: 107, RoomID: 20, Day: 5, StartingTimeSlot: 23, TimeSlotSize: 1},
	}

	result := week.GetWeekSubjectsJSON()

	if !reflect.DeepEqual(expected, result) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}
