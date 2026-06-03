package GeneticAlgorithm

import (
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

const MAX_TIME_SLOT_NUDGE int = 3
const SUBJECT_TIME_SLOT_NUDGE_PROBABILITY int = 90         // %
const SUBJECT_TIME_SLOT_AND_DAY_NUDGE_PROBABILITY int = 50 // %
const SUBJECT_DAY_SWAP_PROBABILITY int = 90                // %
const DAY_SWAP_PERCENT_PROBABILITY int = 10                // %
const SECTION_WEEK_CLEAR_PERCENT_PROBABILITY int = 2       // %
const SUBJECT_ERASURE_PROBABILITY int = 7                  // %
const MUTATION_FITNESS_GUARD int = 11                      // prevent mutations of a weekly section schedule if it's fitness is already high enough, above this defined value

func ApplyRandomDaySwapTimeSlots(
	sched Schedule.UniTimeTables, encoding_resource *EncodingResource,
	all_curriculums []Curriculum.Curriculum,
	department_id uint16, selected_semester int,
) {
	rng := rand.New(rand.NewSource(time.Now().UnixMilli()))
	nstpSubjectIDs := buildNSTP1Or2SubjectIDSet(all_curriculums)

	day_swap_attempts := 0
	day_swap_success := 0
	has_attempted := true

	id_to_instructor := encoding_resource.IdToInstructor
	id_to_room := encoding_resource.IdToRoom

	for day := range Const.N_WEEKLY_SCHOOL_DAYS {

		if rng.Int31n(100) >= int32(DAY_SWAP_PERCENT_PROBABILITY) {
			has_attempted = false
			continue
		}

		day_swap := rng.Intn(int(Const.N_WEEKLY_SCHOOL_DAYS))

		if day_swap == day {
			continue
		}

		IterateSectionsWeekSchedule(sched, all_curriculums, selected_semester, nil, nil, func(indecies IterIndices, values IterValues) IterReturnType {
			if values.Curriculum.DepartmentID == department_id {
				section_subject_async_hours := buildSubjectIDToAsyncHoursMapFromSubjects(values.Semester.Subjects)
				// Avoid moving NSTP 1/2 subjects away from Saturday during day swaps.
				if (day == saturdayDayIndex() || day_swap == saturdayDayIndex()) &&
					(weekDayHasNSTP1Or2(values.WeekSched, day, nstpSubjectIDs) || weekDayHasNSTP1Or2(values.WeekSched, day_swap, nstpSubjectIDs)) {
					return IterProceed
				}

				mfit := MeasureWeekTimeTableBasicFitness(*values.WeekSched, section_subject_async_hours)

				if mfit > float64(MUTATION_FITNESS_GUARD) {
					return IterProceed
				}

				day_swap_attempts++

				usi := indecies.Usi

				is_instructor_a_available := true
				is_instructor_b_available := true

				is_room_a_available := true
				is_room_b_available := true

				for time_slot := range Const.N_DAILY_TIME_SLOTS {

					// check instructors

					instructor_id_a := sched[usi][day][time_slot].GetInstructorID()

					if instructor_id_a != 0 {
						is_instructor_a_available = is_instructor_a_available && id_to_instructor[instructor_id_a].Time.GetAvailability(day_swap, time_slot)
					}

					instructor_id_b := sched[usi][day_swap][time_slot].GetInstructorID()

					if instructor_id_b != 0 {
						is_instructor_b_available = is_instructor_b_available && id_to_instructor[instructor_id_b].Time.GetAvailability(day, time_slot)
					}

					// check rooms

					room_id_a := sched[usi][day][time_slot].GetRoomID()

					if room_id_a != 0 {
						is_room_a_available = is_room_a_available && id_to_room[room_id_a].GetTimeSlotClassCount(day_swap, time_slot) < uint8(id_to_room[room_id_a].Capacity)
					}

					room_id_b := sched[usi][day_swap][time_slot].GetRoomID()
					if room_id_b != 0 {
						is_room_b_available = is_room_b_available && id_to_room[room_id_b].GetTimeSlotClassCount(day, time_slot) < uint8(id_to_room[room_id_b].Capacity)
					}
				}

				if is_instructor_a_available && is_instructor_b_available && is_room_a_available && is_room_b_available {

					for time_slot := range Const.N_DAILY_TIME_SLOTS {

						// update instructors

						instructor_id_a := sched[usi][day][time_slot].GetInstructorID()

						if instructor_id_a > 0 {
							id_to_instructor[instructor_id_a].Time.SetAvailability(false, day_swap, time_slot)
							id_to_instructor[instructor_id_a].Time.SetAvailability(true, day, time_slot)
						}

						instructor_id_b := sched[usi][day_swap][time_slot].GetInstructorID()

						if instructor_id_b > 0 {
							id_to_instructor[instructor_id_b].Time.SetAvailability(false, day, time_slot)
							id_to_instructor[instructor_id_b].Time.SetAvailability(true, day_swap, time_slot)
						}

						// update rooms

						room_id_a := sched[usi][day][time_slot].GetRoomID()

						if room_id_a > 0 {
							id_to_room[room_id_a].IncTimeSlotClassCount(day_swap, time_slot)
							id_to_room[room_id_a].DecTimeSlotClassCount(day, time_slot)
						}

						room_id_b := sched[usi][day_swap][time_slot].GetRoomID()

						if room_id_b > 0 {
							id_to_room[room_id_b].IncTimeSlotClassCount(day, time_slot)
							id_to_room[room_id_b].DecTimeSlotClassCount(day_swap, time_slot)
						}

						// swap time slots

						sched[usi][day][time_slot], sched[usi][day_swap][time_slot] = sched[usi][day_swap][time_slot], sched[usi][day][time_slot]
					}

					day_swap_success++
				}
			}

			return IterProceed
		})
	}

	if os.Getenv("LOG_MODE") == "verbose" && has_attempted {
		log.Printf(
			"Random Mutation : [section-swap-days] %d attempts and, %d successful day swaps. (%d/%d)\n",
			day_swap_attempts, day_swap_success,
			day_swap_attempts, day_swap_success,
		)
	}
}

func ApplyRandomSubjectDaySwap(
	sched Schedule.UniTimeTables, encoding_resource *EncodingResource,
	all_curriculums []Curriculum.Curriculum,
	department_id uint16, selected_semester int,
) {
	rng := rand.New(rand.NewSource(time.Now().UnixMilli()))
	nstpSubjectIDs := buildNSTP1Or2SubjectIDSet(all_curriculums)

	successful_subject_day_swaps := 0
	total_tried_day_swaps := 0
	total_lec_and_lab_subjects := 0

	id_to_instructor := encoding_resource.IdToInstructor
	id_to_room := encoding_resource.IdToRoom

	IterateSectionsWeekSchedule(sched, all_curriculums, selected_semester, nil, nil, func(indicies IterIndices, values IterValues) IterReturnType {

		curriculum := values.Curriculum

		usi := indicies.Usi

		if curriculum.DepartmentID == department_id {
			section_subject_async_hours := buildSubjectIDToAsyncHoursMapFromSubjects(values.Semester.Subjects)

			mfit := MeasureWeekTimeTableBasicFitness(*values.WeekSched, section_subject_async_hours)

			if mfit > float64(MUTATION_FITNESS_GUARD) {
				return IterProceed
			}

			subjects_json := sched[usi].GetWeekSubjectsJSON()

			if len(subjects_json) == 0 {
				return IterProceed
			}

			rng_n := len(subjects_json)

			if rng_n <= 0 {
				return IterProceed
			}

			subject_count_to_try_day_swap := rng.Intn(rng_n) + 1

			total_lec_and_lab_subjects += len(subjects_json)
			total_tried_day_swaps += subject_count_to_try_day_swap

			rng.Shuffle(len(subjects_json), func(i, j int) {
				subjects_json[i], subjects_json[j] = subjects_json[j], subjects_json[i]
			})

			for shuffled_idx := range subject_count_to_try_day_swap {

				if rng.Int31n(100) >= int32(SUBJECT_DAY_SWAP_PROBABILITY) {
					continue
				}

				rand_subject := subjects_json[shuffled_idx]

				if nstpSubjectIDs[rand_subject.SubjectID] {
					continue
				}

				day_swap := rng.Intn(Const.N_WEEKLY_SCHOOL_DAYS)

				is_free_time_slot := true
				for i := range rand_subject.TimeSlotSize {
					swap_slot := sched[usi][day_swap].GetTimeSlot(rand_subject.StartingTimeSlot + i)

					is_time_slot_available := swap_slot.GetSubjectID() == 0
					is_instructor_available := id_to_instructor[rand_subject.InstructorID].Time.GetAvailability(day_swap, rand_subject.StartingTimeSlot+i)
					is_room_available := id_to_room[rand_subject.RoomID].GetTimeSlotClassCount(day_swap, rand_subject.StartingTimeSlot+i) < uint8(id_to_room[rand_subject.RoomID].Capacity)

					if !(is_time_slot_available && is_instructor_available && is_room_available) {
						is_free_time_slot = false
						break
					}
				}

				if !is_free_time_slot {
					continue
				}

				for i := range rand_subject.TimeSlotSize {
					old_slot := sched[usi][rand_subject.Day].GetTimeSlot(rand_subject.StartingTimeSlot + i)
					old_slot.Set(0, 0, 0)
					id_to_instructor[rand_subject.InstructorID].Time.SetAvailability(true, rand_subject.Day, rand_subject.StartingTimeSlot+i)
					id_to_room[rand_subject.RoomID].DecTimeSlotClassCount(rand_subject.Day, rand_subject.StartingTimeSlot+i)

					swap_slot := sched[usi][day_swap].GetTimeSlot(rand_subject.StartingTimeSlot + i)
					swap_slot.Set(rand_subject.SubjectID, rand_subject.InstructorID, rand_subject.RoomID)
					id_to_instructor[rand_subject.InstructorID].Time.SetAvailability(false, day_swap, rand_subject.StartingTimeSlot+i)
					id_to_room[rand_subject.RoomID].IncTimeSlotClassCount(day_swap, rand_subject.StartingTimeSlot+i)
				}

				successful_subject_day_swaps++
			}
		}

		return IterProceed
	})

	if os.Getenv("LOG_MODE") == "verbose" {
		log.Printf("Random Mutation : [subject-day-swaps] from %d lec & lab subjects, there are %d/%d successful day swaps\n", total_lec_and_lab_subjects, successful_subject_day_swaps, total_tried_day_swaps)
	}
}

func ApplyRandomSubjectTimeSlotNudge(
	sched Schedule.UniTimeTables, encoding_resource *EncodingResource,
	all_curriculums []Curriculum.Curriculum,
	department_id uint16, selected_semester int,
) {
	rng := rand.New(rand.NewSource(time.Now().UnixMilli()))

	successful_subject_time_slot_nudge := 0
	total_tried_time_slot_nudge := 0
	total_lec_and_lab_subjects := 0

	id_to_instructor := encoding_resource.IdToInstructor
	id_to_room := encoding_resource.IdToRoom

	IterateSectionsWeekSchedule(sched, all_curriculums, selected_semester, nil, nil, func(indicies IterIndices, values IterValues) IterReturnType {

		curriculum := values.Curriculum
		usi := indicies.Usi

		if curriculum.DepartmentID != department_id {
			return IterProceed
		}

		section_subject_async_hours := buildSubjectIDToAsyncHoursMapFromSubjects(values.Semester.Subjects)
		mfit := MeasureWeekTimeTableBasicFitness(*values.WeekSched, section_subject_async_hours)

		if mfit > float64(MUTATION_FITNESS_GUARD) {
			return IterProceed
		}

		subjects_json := sched[usi].GetWeekSubjectsJSON()
		total_lec_and_lab_subjects += len(subjects_json)

		if len(subjects_json) == 0 {
			return IterProceed
		}

		rng_n := len(subjects_json)

		if rng_n <= 0 {
			return IterProceed
		}

		subject_count_to_try_time_slot_nudge := rng.Intn(rng_n)

		rng.Shuffle(len(subjects_json), func(i, j int) {
			subjects_json[i], subjects_json[j] = subjects_json[j], subjects_json[i]
		})

		if subject_count_to_try_time_slot_nudge > len(subjects_json) {
			panic("WHUWAW VERY GOOOOD!")
		}

		for shuffled_idx := range subject_count_to_try_time_slot_nudge {

			if rng.Int31n(100) >= int32(SUBJECT_TIME_SLOT_NUDGE_PROBABILITY) {
				continue
			}

			total_tried_time_slot_nudge++

			rnd_subject := subjects_json[shuffled_idx]
			nudge_value := Utils.RandomInRange(-MAX_TIME_SLOT_NUDGE, MAX_TIME_SLOT_NUDGE)

			if rnd_subject.SubjectID == 0 {
				continue
			}

			if nudge_value == 0 {
				continue
			}

			is_nudge_start_idx_lt_min := (rnd_subject.StartingTimeSlot + nudge_value) < 0
			is_nudge_start_idx_gt_max := (rnd_subject.StartingTimeSlot + nudge_value) >= Const.N_DAILY_TIME_SLOTS
			is_nudge_start_idx_valid := !is_nudge_start_idx_lt_min && !is_nudge_start_idx_gt_max

			is_nudge_end_idx_lt_min := (rnd_subject.StartingTimeSlot + nudge_value + rnd_subject.TimeSlotSize - 1) < 0
			is_nudge_end_idx_gt_max := (rnd_subject.StartingTimeSlot + nudge_value + rnd_subject.TimeSlotSize - 1) >= Const.N_DAILY_TIME_SLOTS
			is_nudge_end_idx_valid := !is_nudge_end_idx_lt_min && !is_nudge_end_idx_gt_max

			if !is_nudge_start_idx_valid || !is_nudge_end_idx_valid {
				continue
			}

			is_free_time_slot := true

			for i_ts := 0; i_ts < rnd_subject.TimeSlotSize; i_ts++ {
				nudge_slot := sched[usi][rnd_subject.Day].GetTimeSlot(rnd_subject.StartingTimeSlot + nudge_value + i_ts)

				is_same_subject_block := (nudge_slot.GetSubjectID() == rnd_subject.SubjectID) &&
					(nudge_slot.GetInstructorID() == rnd_subject.InstructorID) &&
					(nudge_slot.GetRoomID() == rnd_subject.RoomID)

				if _, has_instructor := id_to_instructor[rnd_subject.InstructorID]; !has_instructor {
					log.Panicf("that instructor id does not exist id = %d ", rnd_subject.InstructorID)
				}

				if _, has_room := id_to_room[rnd_subject.RoomID]; !has_room {
					log.Panicf("that room id does not exist id = %d ", rnd_subject.RoomID)
				}

				is_empty_slot := nudge_slot.GetSubjectID() == 0
				is_instructor_available := id_to_instructor[rnd_subject.InstructorID].Time.GetAvailability(rnd_subject.Day, rnd_subject.StartingTimeSlot+nudge_value+i_ts)
				is_room_available := (id_to_room[rnd_subject.RoomID].GetTimeSlotClassCount(rnd_subject.Day, (rnd_subject.StartingTimeSlot + nudge_value + i_ts))) < uint8(id_to_room[rnd_subject.RoomID].Capacity)

				if !((is_empty_slot && is_instructor_available && is_room_available) || is_same_subject_block) {
					is_free_time_slot = false
					break
				}
			}

			if !is_free_time_slot {
				continue
			}

			for i_ts := 0; i_ts < rnd_subject.TimeSlotSize; i_ts++ {
				old_slot := sched[usi][rnd_subject.Day].GetTimeSlot(rnd_subject.StartingTimeSlot + i_ts)
				old_slot.Set(0, 0, 0)

				id_to_instructor[rnd_subject.InstructorID].Time.SetAvailability(true, rnd_subject.Day, (rnd_subject.StartingTimeSlot + i_ts))

				prev_room_val := int(id_to_room[rnd_subject.RoomID].GetTimeSlotClassCount(rnd_subject.Day, (rnd_subject.StartingTimeSlot + i_ts)))
				id_to_room[rnd_subject.RoomID].DecTimeSlotClassCount(rnd_subject.Day, (rnd_subject.StartingTimeSlot + i_ts))
				next_room_val := int(id_to_room[rnd_subject.RoomID].GetTimeSlotClassCount(rnd_subject.Day, (rnd_subject.StartingTimeSlot + i_ts)))

				if !(prev_room_val == 0 && next_room_val == 0) {
					if prev_room_val == 0 {
						if next_room_val != 0 {
							log.Panic("decrement is wrong for zero values")
						}
					}

					if next_room_val != prev_room_val-1 {
						log.Panicf("decrement is wrong for normal values : previous = %d, next = %d", prev_room_val, next_room_val)
					}
				}
			}

			for i_ts := 0; i_ts < rnd_subject.TimeSlotSize; i_ts++ {

				nudge_slot := sched[usi][rnd_subject.Day].GetTimeSlot(rnd_subject.StartingTimeSlot + nudge_value + i_ts)
				nudge_slot.Set(rnd_subject.SubjectID, rnd_subject.InstructorID, rnd_subject.RoomID)

				id_to_instructor[rnd_subject.InstructorID].Time.SetAvailability(false, rnd_subject.Day, (rnd_subject.StartingTimeSlot + nudge_value + i_ts))

				prev_room_val := int(id_to_room[rnd_subject.RoomID].GetTimeSlotClassCount(rnd_subject.Day, (rnd_subject.StartingTimeSlot + nudge_value + i_ts)))
				id_to_room[rnd_subject.RoomID].IncTimeSlotClassCount(rnd_subject.Day, (rnd_subject.StartingTimeSlot + nudge_value + i_ts))
				next_room_val := int(id_to_room[rnd_subject.RoomID].GetTimeSlotClassCount(rnd_subject.Day, (rnd_subject.StartingTimeSlot + nudge_value + i_ts)))

				if prev_room_val == 15 {
					log.Panic("increment is allowing increment above 15")
				}

				if prev_room_val == int(id_to_room[rnd_subject.RoomID].Capacity) {
					log.Panic("increment is allowing increment above maximum room capacity")
				}

				if next_room_val != prev_room_val+1 {
					log.Panicf("increment is wrong for normal values: previous = %d, next = %d", prev_room_val, next_room_val)
				}
			}

			successful_subject_time_slot_nudge++
		}

		return IterProceed
	})

	if os.Getenv("LOG_MODE") == "verbose" {
		log.Printf(
			"Random Mutation : [time-slot-nudge] from %d lec and lab subjects, there are %d/%d successful subjects nudge on different time slot\n",
			total_lec_and_lab_subjects, successful_subject_time_slot_nudge, total_tried_time_slot_nudge,
		)
	}
}

func ApplyRandomSubjectTimeSlotAndDayNudge(
	sched Schedule.UniTimeTables, encoding_resource *EncodingResource,
	all_curriculums []Curriculum.Curriculum,
	department_id uint16, selected_semester int,
) {
	rng := rand.New(rand.NewSource(time.Now().UnixMilli()))
	nstpSubjectIDs := buildNSTP1Or2SubjectIDSet(all_curriculums)

	successful_subject_nudge := 0
	total_tried_nudge := 0
	total_lec_and_lab_subjects := 0

	id_to_instructor := encoding_resource.IdToInstructor
	id_to_room := encoding_resource.IdToRoom

	IterateSectionsWeekSchedule(sched, all_curriculums, selected_semester, nil, nil, func(indicies IterIndices, values IterValues) IterReturnType {

		curriculum := values.Curriculum

		usi := indicies.Usi

		if curriculum.DepartmentID == department_id {
			section_subject_async_hours := buildSubjectIDToAsyncHoursMapFromSubjects(values.Semester.Subjects)

			mfit := MeasureWeekTimeTableBasicFitness(*values.WeekSched, section_subject_async_hours)

			if mfit > float64(MUTATION_FITNESS_GUARD) {
				return IterProceed
			}

			subjects_json := sched[usi].GetWeekSubjectsJSON()
			total_lec_and_lab_subjects += len(subjects_json)

			if len(subjects_json) == 0 {
				return IterProceed
			}

			rng_n := len(subjects_json)

			if rng_n <= 0 {
				return IterProceed
			}

			subject_count_to_try_nudge := rng.Intn(rng_n) + 1

			if subject_count_to_try_nudge <= 0 {
				return IterProceed
			}

			rng.Shuffle(len(subjects_json), func(i, j int) {
				subjects_json[i], subjects_json[j] = subjects_json[j], subjects_json[i]
			})

			for shuffled_idx := 0; shuffled_idx < subject_count_to_try_nudge; shuffled_idx++ {

				if rng.Int31n(100) >= int32(SUBJECT_TIME_SLOT_AND_DAY_NUDGE_PROBABILITY) {
					continue
				}

				total_tried_nudge++

				rnd_subject := subjects_json[shuffled_idx]

				if nstpSubjectIDs[rnd_subject.SubjectID] {
					continue
				}

				day_swap := rng.Intn(Const.N_WEEKLY_SCHOOL_DAYS)
				nudge_value := Utils.RandomInRange(-MAX_TIME_SLOT_NUDGE, MAX_TIME_SLOT_NUDGE)

				is_nudge_start_idx_lt_min := (rnd_subject.StartingTimeSlot + nudge_value) < 0
				is_nudge_start_idx_gt_max := (rnd_subject.StartingTimeSlot + nudge_value) >= Const.N_DAILY_TIME_SLOTS
				is_nudge_start_idx_valid := !is_nudge_start_idx_lt_min && !is_nudge_start_idx_gt_max

				is_nudge_end_idx_lt_min := (rnd_subject.StartingTimeSlot + nudge_value + rnd_subject.TimeSlotSize - 1) < 0
				is_nudge_end_idx_gt_max := (rnd_subject.StartingTimeSlot + nudge_value + rnd_subject.TimeSlotSize - 1) >= Const.N_DAILY_TIME_SLOTS
				is_nudge_end_idx_valid := !is_nudge_end_idx_lt_min && !is_nudge_end_idx_gt_max

				if !is_nudge_start_idx_valid || !is_nudge_end_idx_valid {
					continue
				}

				is_free_time_slot := true
				for i := 0; i < rnd_subject.TimeSlotSize; i++ {
					nudge_slot := sched[usi][day_swap].GetTimeSlot(rnd_subject.StartingTimeSlot + nudge_value + i)

					is_same_subject_block := (nudge_slot.GetSubjectID() == rnd_subject.SubjectID) &&
						(nudge_slot.GetInstructorID() == rnd_subject.InstructorID) &&
						(nudge_slot.GetRoomID() == rnd_subject.RoomID)

					if is_same_subject_block {
						continue
					}

					is_empty_slot := nudge_slot.GetSubjectID() == 0
					is_instructor_available := id_to_instructor[rnd_subject.InstructorID].Time.GetAvailability(day_swap, rnd_subject.StartingTimeSlot+nudge_value+i)
					is_room_available := id_to_room[rnd_subject.RoomID].GetTimeSlotClassCount(day_swap, rnd_subject.StartingTimeSlot+nudge_value+i) < uint8(id_to_room[rnd_subject.RoomID].Capacity)

					if !(is_empty_slot && is_instructor_available && is_room_available) {
						is_free_time_slot = false
						break
					}
				}

				if !is_free_time_slot {
					continue
				}

				for i := 0; i < rnd_subject.TimeSlotSize; i++ {
					old_slot := sched[usi][rnd_subject.Day].GetTimeSlot(rnd_subject.StartingTimeSlot + i)
					old_slot.Set(0, 0, 0)

					id_to_instructor[rnd_subject.InstructorID].Time.SetAvailability(true, rnd_subject.Day, (rnd_subject.StartingTimeSlot + i))
					id_to_room[rnd_subject.RoomID].DecTimeSlotClassCount(rnd_subject.Day, (rnd_subject.StartingTimeSlot + i))
				}

				for i := 0; i < rnd_subject.TimeSlotSize; i++ {
					nudge_slot := sched[usi][day_swap].GetTimeSlot(rnd_subject.StartingTimeSlot + nudge_value + i)
					nudge_slot.Set(rnd_subject.SubjectID, rnd_subject.InstructorID, rnd_subject.RoomID)

					id_to_instructor[rnd_subject.InstructorID].Time.SetAvailability(false, day_swap, (rnd_subject.StartingTimeSlot + nudge_value + i))
					id_to_room[rnd_subject.RoomID].IncTimeSlotClassCount(day_swap, (rnd_subject.StartingTimeSlot + nudge_value + i))
				}

				successful_subject_nudge++

			}
		}

		return IterProceed
	})

	if os.Getenv("LOG_MODE") == "verbose" {
		log.Printf(
			"Random Mutation : [day-time-slot-nudge] from %d lec and lab subjects, there are %d/%d successful subjects nudge on different day & time slot\n",
			total_lec_and_lab_subjects, successful_subject_nudge, total_tried_nudge,
		)
	}
}
