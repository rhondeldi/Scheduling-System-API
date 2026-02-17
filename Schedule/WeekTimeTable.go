package Schedule

import (
	"fmt"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
)

// type for weekly schedule of a class / section.
//
// this is just an array of `Day` types.
type WeekTimeTable [Const.N_WEEKLY_SCHOOL_DAYS]DayTimeTable

func (week *WeekTimeTable) GetDayTimeTable(day_idx int) *DayTimeTable {
	if day_idx < 0 || day_idx >= Const.N_WEEKLY_SCHOOL_DAYS {
		panic(fmt.Sprintf(
			"GetDaySchedule(day_idx = %d | min:max = 0:%d): error index out of bounds",
			day_idx, (Const.N_WEEKLY_SCHOOL_DAYS - 1),
		))
	}

	return &week[day_idx]
}

func (week *WeekTimeTable) GetWeekSubjectsJSON() []TimeSlotSubjectJSON {
	week_subjects_json := make([]TimeSlotSubjectJSON, 0)

	for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
		for time_slot := 0; time_slot < Const.N_DAILY_TIME_SLOTS; time_slot++ {
			slot := week[day].GetTimeSlot(time_slot)

			subject_id := slot.GetSubjectID()

			if subject_id == 0 {
				continue
			}

			subject_json := TimeSlotSubjectJSON{
				SubjectID:        subject_id,
				InstructorID:     slot.GetInstructorID(),
				RoomID:           slot.GetRoomID(),
				Day:              day,
				StartingTimeSlot: time_slot,
				TimeSlotSize:     1,
			}

			for forward_time_slot := time_slot + 1; forward_time_slot < Const.N_DAILY_TIME_SLOTS; forward_time_slot++ {
				forward_slot := week[day].GetTimeSlot(forward_time_slot)

				if forward_slot.GetSubjectID() == slot.GetSubjectID() && forward_slot.GetInstructorID() == slot.GetInstructorID() && forward_slot.GetRoomID() == slot.GetRoomID() {
					subject_json.TimeSlotSize++
				} else {
					time_slot = forward_time_slot - 1
					break
				}

				if forward_time_slot == (Const.N_DAILY_TIME_SLOTS - 1) {
					time_slot = 9999
					break
				}
			}

			week_subjects_json = append(week_subjects_json, subject_json)
		}
	}

	return week_subjects_json
}

type TimeSlotSubjectJSON struct {
	SubjectID        uint16
	InstructorID     uint16
	RoomID           uint16
	Day              int
	StartingTimeSlot int
	TimeSlotSize     int
}
