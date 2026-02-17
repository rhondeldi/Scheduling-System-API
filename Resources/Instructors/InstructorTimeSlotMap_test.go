package Instructors_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
)

func TestInstructorTimeSlotMapSerialization(t *testing.T) {
	for tidx := 0; tidx < 100; tidx++ {
		instructor_time := Instructors.InstructorTimeSlotBitMap{}

		for i := range instructor_time {
			instructor_time[i] = rand.Uint64()
		}

		serialized := instructor_time.Serialize()

		deserialized_time := Instructors.InstructorTimeSlotBitMap{}
		deserialized_time.Deserialize(serialized)

		if instructor_time != deserialized_time {
			t.Fatal("not equal raw and deserialized value")
		}

		for i := range instructor_time {
			if instructor_time[i] != deserialized_time[i] {
				t.Fatal("not equal raw and deserialized value in index")
			}
		}
	}
}

func TestInstructorTimeSlotMapAvailability(t *testing.T) {
	instructor_time := Instructors.InstructorTimeSlotBitMap{}

	cnt := 0
	index := 0
	for day := 0; day < 6; day++ {
		for time_slot := 0; time_slot < 24; time_slot++ {

			available := instructor_time.GetAvailability(day, time_slot)

			if cnt == 0 || cnt == 63 {
				fmt.Printf("Array Values (day = %d, time_slot = %d)[0]: %064b\n", day, time_slot, instructor_time)
			}

			if available == false {
				t.Errorf(
					"GetAvailability(day = %d, time_slot = %d) : {loop test phase 1} should not be false yet",
					day, time_slot,
				)
			}

			instructor_time.SetAvailability(false, day, time_slot)
			instructor_time.SetAvailability(false, day, time_slot)
			available = instructor_time.GetAvailability(day, time_slot)

			if cnt == 0 || cnt == 63 {
				fmt.Printf("Array Values (day = %d, time_slot = %d)[1]: %064b\n", day, time_slot, instructor_time)
			}

			if available == true {
				t.Errorf(
					"GetAvailability(day = %d, time_slot = %d) : {loop test phase 2} should be false now",
					day, time_slot,
				)
			}

			if instructor_time[index] != (uint64(1) << cnt) {
				t.Errorf(
					"[day:%d, time_slot:%d] : {loop test phase 3} (bitsetmap[%d] = %d) != ((uint64(1) << cnt) = %d)",
					day, time_slot, index, instructor_time[index], (uint64(1) << cnt),
				)
			}

			instructor_time.SetAvailability(true, day, time_slot)
			instructor_time.SetAvailability(true, day, time_slot)
			available = instructor_time.GetAvailability(day, time_slot)

			if available == false {
				t.Errorf(
					"GetAvailability(day = %d, time_slot = %d) : {loop test phase 4} should not be false again",
					day, time_slot,
				)
			}

			if instructor_time[index] != uint64(0) {
				t.Errorf(
					"[day:%d, time_slot:%d] : {loop test phase 5} (bitsetmap[%d] = %d) != uint64(0))",
					day, time_slot, index, instructor_time[index],
				)
			}

			if cnt == 0 || cnt == 63 {
				fmt.Printf("Array Values (day = %d, time_slot = %d)[2]: %064b\n\n", day, time_slot, instructor_time)
			}

			cnt++

			if cnt == 64 {
				cnt = 0
				index++
			}
		}
	}
}
