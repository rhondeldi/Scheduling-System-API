package Schedule_test

import (
	"testing"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

func Test_DayTimeTable_Availability(t *testing.T) {
	for time_slot_size := 1; time_slot_size <= (9 * Const.N_HOUR_TIME_SLOTS); time_slot_size++ {

		for start_time_slot := 0; start_time_slot < (Const.N_DAILY_TIME_SLOTS - time_slot_size + 1); start_time_slot++ {

			day_time_table := Schedule.DayTimeTable{}

			for i := 0; i < time_slot_size; i++ {
				day_time_table[(start_time_slot + i)].SetSubjectID(1)
			}

			for time_slots := 0; time_slots < (Const.N_DAILY_TIME_SLOTS - time_slot_size); time_slots++ {
				curr_start_time_slot := time_slots
				curr_end_time_slot := time_slots + time_slot_size - 1

				curr_start_time_slot_within_index := (curr_start_time_slot >= start_time_slot) && (curr_start_time_slot <= (start_time_slot + time_slot_size - 1))
				curr_end_time_slot_within_index := (curr_end_time_slot >= start_time_slot) && (curr_end_time_slot <= (start_time_slot + time_slot_size - 1))

				if curr_start_time_slot_within_index || curr_end_time_slot_within_index {
					if day_time_table.IsTimeAvailable(time_slots, time_slot_size) != false {
						t.Errorf(
							"Test_DayTimeTable_Availability: start_time_slot(%d), time_slot(%d), time_slot_size(%d) : Availability : %t should be false",
							start_time_slot, time_slots, time_slot_size, day_time_table.IsTimeAvailable(time_slots, time_slot_size),
						)
					}
				} else {
					if day_time_table.IsTimeAvailable(time_slots, time_slot_size) != true {
						t.Errorf(
							"Test_DayTimeTable_Availability: start_time_slot(%d), time_slot(%d), time_slot_size(%d) : Availability : %t should be true",
							start_time_slot, time_slots, time_slot_size, day_time_table.IsTimeAvailable(time_slots, time_slot_size),
						)
					}
				}
			}
		}
	}
}
