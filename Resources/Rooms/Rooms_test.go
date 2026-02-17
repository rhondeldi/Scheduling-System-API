package Rooms_test

import (
	"math/rand"
	"testing"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
)

func Test_Rooms_timeSlotClassCount_methods(t *testing.T) {
	room := &Rooms.Room{
		Capacity: uint16(Rooms.MAX_ROOM_CAPACITY),
	}
	array := [Const.N_WEEKLY_SCHOOL_DAYS][Const.N_DAILY_TIME_SLOTS]uint8{}

	for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
		for time_slot := 0; time_slot < Const.N_DAILY_TIME_SLOTS; time_slot++ {
			array[day][time_slot] = uint8(rand.Intn(16))
			room.SetTimeSlotClassCount(day, time_slot, array[day][time_slot])
		}
	}

	for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
		for time_slot := 0; time_slot < Const.N_DAILY_TIME_SLOTS; time_slot++ {
			if room.GetTimeSlotClassCount(day, time_slot) != array[day][time_slot] {
				t.Errorf("set then get method result to wrong value at : [%d][%d]", day, time_slot)
			}
		}
	}

	for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
		for time_slot := 0; time_slot < Const.N_DAILY_TIME_SLOTS; time_slot++ {
			num_of_inc := 1 + rand.Intn(2)
			for i := 0; i < num_of_inc; i++ {
				if array[day][time_slot] < 15 {
					array[day][time_slot]++
				}

				if room.GetTimeSlotClassCount(day, time_slot) < 15 {
					room.IncTimeSlotClassCount(day, time_slot)
				}
			}
		}
	}

	for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
		for time_slot := 0; time_slot < Const.N_DAILY_TIME_SLOTS; time_slot++ {
			if room.GetTimeSlotClassCount(day, time_slot) != array[day][time_slot] {
				t.Errorf("increment result to wrong value at : [%d][%d]", day, time_slot)
			}
		}
	}
}

func Test_Rooms_timeSlotClassCount_methods2(t *testing.T) {
	for range 100000 {

		room := &Rooms.Room{
			Capacity: uint16(Rooms.MAX_ROOM_CAPACITY),
		}

		array := [Const.N_WEEKLY_SCHOOL_DAYS][Const.N_DAILY_TIME_SLOTS]int{}

		for day := range Const.N_WEEKLY_SCHOOL_DAYS {
			for time_slot := range Const.N_DAILY_TIME_SLOTS {
				array[day][time_slot] = rand.Intn(16)

				for range int(array[day][time_slot]) {
					room.IncTimeSlotClassCount(day, time_slot)
				}
			}
		}

		for day := range Const.N_WEEKLY_SCHOOL_DAYS {
			for time_slot := range Const.N_DAILY_TIME_SLOTS {
				if int(room.GetTimeSlotClassCount(day, time_slot)) != array[day][time_slot] {
					t.Errorf("increment result, wrong value at : [%d][%d]", day, time_slot)
				}
			}
		}

		for day := range Const.N_WEEKLY_SCHOOL_DAYS {
			for time_slot := range Const.N_DAILY_TIME_SLOTS {

				if array[day][time_slot] > 0 {
					rn := rand.Intn(array[day][time_slot])
					array[day][time_slot] -= rn

					for range int(rn) {
						room.DecTimeSlotClassCount(day, time_slot)
					}
				}
			}
		}

		for day := range Const.N_WEEKLY_SCHOOL_DAYS {
			for time_slot := range Const.N_DAILY_TIME_SLOTS {
				if int(room.GetTimeSlotClassCount(day, time_slot)) != array[day][time_slot] {
					t.Errorf("decrement result, wrong value at : [%d][%d]", day, time_slot)
				}
			}
		}
	}
}
