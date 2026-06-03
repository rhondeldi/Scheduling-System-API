package GeneticAlgorithm

import (
	"fmt"
	"log"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

func ValidateEncodingResource(
	sched Schedule.UniTimeTables, encoding_resource *EncodingResource,
	curriculums []Curriculum.Curriculum, selected_semester int,
) error {

	room_id_to_room := make(map[uint16]*Rooms.Room)

	for out_key, out_v := range encoding_resource.DeptIdToRoomtypeToRooms {
		for in_key, in_v := range out_v {
			for room_idx, room := range in_v {
				room_id_to_room[room.RoomID] = &encoding_resource.DeptIdToRoomtypeToRooms[out_key][in_key][room_idx]
			}
		}
	}

	instructor_id_to_instructor := make(map[uint16]*Instructors.Instructor)

	for k, v := range encoding_resource.DeptIdToInstructors {
		for instructor_idx, instructor := range v {
			instructor_id_to_instructor[instructor.InstructorID] = &encoding_resource.DeptIdToInstructors[k][instructor_idx]
		}
	}

	encoding_resource.IdToRoom = room_id_to_room
	encoding_resource.IdToInstructor = instructor_id_to_instructor

	var err_return error = nil

	for day := range Const.N_WEEKLY_SCHOOL_DAYS {
		for time_slot := range Const.N_DAILY_TIME_SLOTS {

			room_id_to_count := make(map[uint16]int)

			IterateSectionsWeekSchedule(sched, curriculums, selected_semester, nil, nil,
				func(indicies IterIndices, values IterValues) IterReturnType {

					id_subject := sched[indicies.Usi][day][time_slot].GetSubjectID()
					id_instructor := sched[indicies.Usi][day][time_slot].GetInstructorID()
					id_room := sched[indicies.Usi][day][time_slot].GetRoomID()

					// resource dangling cases

					if id_subject == 0 && id_instructor > 0 {
						err_return = fmt.Errorf(
							"ValidateEncodingResource: dangling instructor - [%d] %s %s %s in %s %s section %s => usi(%d), day(%d), timeslot(%d)",
							id_instructor,
							encoding_resource.IdToInstructor[id_instructor].FirstName,
							encoding_resource.IdToInstructor[id_instructor].MiddleInitial,
							encoding_resource.IdToInstructor[id_instructor].LastName,
							values.Curriculum.CurriculumCode, values.Semester.Name, Curriculum.SECTION[indicies.Section],
							indicies.Usi, day, time_slot,
						)

						return IterBreakCurriculumLoop
					}

					if id_subject == 0 && id_room > 0 {
						err_return = fmt.Errorf(
							"ValidateEncodingResource: dangling room - [%d] %s in %s %s section %s => usi(%d), day(%d), timeslot(%d)",
							id_room,
							encoding_resource.IdToRoom[id_room].Name,
							values.Curriculum.CurriculumCode, values.Semester.Name, Curriculum.SECTION[indicies.Section],
							indicies.Usi, day, time_slot,
						)

						return IterBreakCurriculumLoop
					}

					// subject dangling cases

					if id_subject > 0 && id_instructor == 0 {
						err_return = fmt.Errorf(
							"ValidateEncodingResource: missing instructor in %s %s section %s => usi(%d), day(%d), timeslot(%d)",
							values.Curriculum.CurriculumCode, values.Semester.Name, Curriculum.SECTION[indicies.Section],
							indicies.Usi, day, time_slot,
						)

						return IterBreakCurriculumLoop
					}

					if id_subject > 0 && id_room == 0 {
						err_return = fmt.Errorf(
							"ValidateEncodingResource: missing room in %s %s section %s => usi(%d), day(%d), timeslot(%d)",
							values.Curriculum.CurriculumCode, values.Semester.Name, Curriculum.SECTION[indicies.Section],
							indicies.Usi, day, time_slot,
						)

						return IterBreakCurriculumLoop
					}

					if id_subject == 0 {
						return IterProceed
					}

					// check instructor encoding resource correctness

					if encoding_resource == nil {
						log.Panic("ValidateEncodingResource: encoding_resource is nil")
					}

					if encoding_resource.IdToInstructor == nil {
						log.Panic("ValidateEncodingResource: encoding_resource.IdToInstructor is nil")
					}

					if _, has_instructor_id := encoding_resource.IdToInstructor[id_instructor]; !has_instructor_id {
						err_return = fmt.Errorf(
							"ValidateEncodingResource: the instructor id = %d detected in university schedule is not found in the encoding resource",
							id_instructor,
						)

						return IterBreakCurriculumLoop
					}

					is_instructor_available := encoding_resource.IdToInstructor[id_instructor].Time.GetAvailability(day, time_slot)

					if is_instructor_available {
						err_return = fmt.Errorf(
							"ValidateEncodingResource: (%d|%s %s %s) in %s %s section %s should not be available in this time slot => usi(%d), day(%d), timeslot(%d)",
							id_instructor,
							encoding_resource.IdToInstructor[id_instructor].FirstName,
							encoding_resource.IdToInstructor[id_instructor].MiddleInitial,
							encoding_resource.IdToInstructor[id_instructor].LastName,
							values.Curriculum.CurriculumCode, values.Semester.Name, Curriculum.SECTION[indicies.Section],
							indicies.Usi, day, time_slot,
						)

						return IterBreakCurriculumLoop
					}

					// check room encoding resource correctness

					if _, has_room_id := encoding_resource.IdToRoom[id_room]; !has_room_id {
						err_return = fmt.Errorf(
							"ValidateEncodingResource: the room id = %d detected in university schedule is not found in the encoding resource",
							id_room,
						)

						return IterBreakCurriculumLoop
					}

					if _, has_id := room_id_to_count[id_room]; !has_id {
						room_id_to_count[id_room] = 0
					}

					room_id_to_count[id_room]++

					for id_room, allocation_count := range room_id_to_count {
						encoding_allocation_count := encoding_resource.IdToRoom[id_room].GetTimeSlotClassCount(day, time_slot)
						if encoding_allocation_count < uint8(allocation_count) {
							err_return = fmt.Errorf(
								"ValidateEncodingResource: [usi:%d] wrong room allocation of [%d]-%s in day(%d), timeslot(%d), encoding has %d, validation detected %d",
								indicies.Usi,
								encoding_resource.IdToRoom[id_room].RoomID,
								encoding_resource.IdToRoom[id_room].Name,
								day, time_slot, encoding_allocation_count, allocation_count,
							)

							return IterBreakCurriculumLoop
						}
					}

					return IterProceed
				},
			)

			for id_room, allocation_count := range room_id_to_count {
				encoding_allocation_count := encoding_resource.IdToRoom[id_room].GetTimeSlotClassCount(day, time_slot)
				if encoding_allocation_count != uint8(allocation_count) {
					return fmt.Errorf(
						"ValidateEncodingResource: wrong room allocation of (%d|%s) in day(%d), timeslot(%d), encoding has %d, validation detected %d",
						encoding_resource.IdToRoom[id_room].RoomID,
						encoding_resource.IdToRoom[id_room].Name,
						day, time_slot, encoding_allocation_count, allocation_count,
					)
				}
			}
		}
	}

	return err_return
}

/*
validate assigned subjects to every section schedules in the whole university.

to validate whole university schedules, set department to encode to nil:

	department_to_validate = nil
*/
func HorizontalValidation(
	university_sched Schedule.UniTimeTables,
	curriculums []Curriculum.Curriculum,
	department_to_validate map[uint16]bool, selected_semester int,
) []error {

	errs_slice := make([]error, 0, 16)

	/////////////////////////////////////////////////////////////////////////////////
	//                            HORIZONTAL CHECKS
	/////////////////////////////////////////////////////////////////////////////////

	total_university_sections := Curriculum.GetTotalNumberOfSections(curriculums, selected_semester)

	if total_university_sections != len(university_sched) {
		errs_slice = append(errs_slice, fmt.Errorf(
			"read total university sections (%d) in persistence did not match the university schedule instance (%d)",
			total_university_sections, len(university_sched),
		))
	}

	IterateSectionsWeekSchedule(university_sched, curriculums, selected_semester, nil, nil, func(indicies IterIndices, values IterValues) IterReturnType {

		curriculum := values.Curriculum
		year_level := values.YearLevel
		semester := values.Semester

		section_idx := indicies.Section
		usi := indicies.Usi

		if department_to_validate != nil {
			is_to_validate := department_to_validate[curriculum.DepartmentID]

			if !is_to_validate {
				return IterProceed // skip section that does not need horizontal validation
			}
		}

		is_subject_id_to_is_in_curriculum := make(map[uint16]bool)
		stray_subject_ids := make([]uint16, 0)

		for _, section_subject := range values.Semester.Subjects {
			is_subject_id_to_is_in_curriculum[section_subject.ID] = true
		}

		subject_id_to_instructor_id := make(map[uint16]uint16)

		subject_id_to_time_slot_count := make(map[uint16]int)

		for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
			for time_slot := 0; time_slot < Const.N_DAILY_TIME_SLOTS; time_slot++ {
				subject_id := university_sched[usi][day][time_slot].GetSubjectID()

				if subject_id == 0 {
					if university_sched[usi][day][time_slot].GetInstructorID() > 0 {
						errs_slice = append(
							errs_slice,
							fmt.Errorf(
								"dangling instructor id detected in %s %s %s %s [usi:%d] - day(%d), timeslot(%d)",
								curriculum.CurriculumCode, year_level.Name, semester.Name,
								Curriculum.SEMESTER_INDEX_NAME[indicies.Section],
								indicies.Usi, day, time_slot,
							),
						)

						return IterBreakCurriculumLoop
					}
					if university_sched[usi][day][time_slot].GetRoomID() > 0 {
						errs_slice = append(
							errs_slice,
							fmt.Errorf(
								"dangling room id detected in %s %s %s %s [usi:%d] - day(%d), timeslot(%d)",
								curriculum.CurriculumCode, year_level.Name, semester.Name,
								Curriculum.SEMESTER_INDEX_NAME[indicies.Section],
								indicies.Usi, day, time_slot,
							),
						)

						return IterBreakCurriculumLoop
					}
					continue
				}

				if !is_subject_id_to_is_in_curriculum[subject_id] {
					stray_subject_ids = append(stray_subject_ids, subject_id)
				}

				_, has_subjsubject_id := subject_id_to_time_slot_count[subject_id]

				if !has_subjsubject_id {
					subject_id_to_time_slot_count[subject_id] = 1
				} else {
					subject_id_to_time_slot_count[subject_id]++
				}

				// check if instructors are the same for same subject ids

				if instructor_id, has_subject_id := subject_id_to_instructor_id[subject_id]; has_subject_id {
					if instructor_id != university_sched[usi][day][time_slot].GetInstructorID() {
						errs_slice = append(
							errs_slice,
							fmt.Errorf(
								"different instructor id detected in %s, %s, %s, section %s, [usi:%d] - day(%d), timeslot(%d), subject id %d, instructor ids: %d & %d",
								curriculum.CurriculumCode, year_level.Name, semester.Name,
								Curriculum.SECTION[indicies.Section],
								indicies.Usi, day, time_slot,
								subject_id, instructor_id, university_sched[usi][day][time_slot].GetInstructorID(),
							),
						)

						return IterBreakCurriculumLoop
					}
				} else {
					subject_id_to_instructor_id[subject_id] = university_sched[usi][day][time_slot].GetInstructorID()
				}
			}
		}

		// ── EMPTY-SECTION GUARD ────────────────────────────────────────────────
		//
		// A section that has no placed subjects AND no stray subjects is treated
		// as a deliberate skip — typically because GA_SKIP_CURRICULA marked the
		// curriculum as unschedulable, or because it belongs to another program
		// in the same department whose data is over-capacity (e.g. BSPsych
		// within DAS when 38 lab-room time slots are short).
		//
		// Without this guard, HorizontalValidation reports every curriculum
		// subject as "missing" on every empty section, which causes the GA's
		// mutation phase to revert EVERY individual EVERY generation: the
		// repaired schedule re-encodes BSPsych as empty (correct), HV reports
		// 7 missing subjects per BSPsych section (wrong), the GA reverts the
		// mutation, and progress halts.  The dangling-id checks above still
		// run, so we never let a half-cleared section sneak through — only
		// genuinely empty ones pass.
		//
		// The guard is the third piece of the three-part skip chain:
		//   1. RunGeneticAlgorithm writes IsSchedIdxToSubIdToSkip (3 sites)
		//   2. EncodeIndividualGenome leaves those subjects unplaced
		//   3. HorizontalValidation tolerates the resulting empty section ← here
		if len(subject_id_to_time_slot_count) == 0 && len(stray_subject_ids) == 0 {
			return IterProceed
		}

		if len(semester.Subjects) != len(subject_id_to_time_slot_count) {
			errs_slice = append(errs_slice, fmt.Errorf(
				"detected %d missing subject(s) in %s, %s, %s, section %s (usi:%d)",
				len(semester.Subjects)-len(subject_id_to_time_slot_count),
				curriculum.CurriculumCode,
				semester.Name,
				year_level.Name,
				Curriculum.SECTION[section_idx],
				usi,
			))
		}

		for _, subject := range semester.Subjects {
			_, has_subject_id := subject_id_to_time_slot_count[subject.ID]
			expected_slots := uint8(subject.SlotsToAssign())

			if !has_subject_id {
				errs_slice = append(errs_slice, fmt.Errorf(
					"the subject %s was not assigned to %s, %s, %s, section %s (usi:%d)",
					subject.Code, curriculum.CurriculumCode, year_level.Name, semester.Name, Curriculum.SECTION[section_idx], usi,
				))
			} else if expected_slots > uint8(subject_id_to_time_slot_count[subject.ID]) {
				errs_slice = append(errs_slice, fmt.Errorf(
					"the subject %s has missing time slot allocations, expecting %d, but only found %d in %s, %s, %s, section %s (usi:%d)",
					subject.Code,
					expected_slots, uint8(subject_id_to_time_slot_count[subject.ID]),
					curriculum.CurriculumCode, year_level.Name, semester.Name, Curriculum.SECTION[section_idx], usi,
				))
			} else if expected_slots < uint8(subject_id_to_time_slot_count[subject.ID]) {
				errs_slice = append(errs_slice, fmt.Errorf(
					"the subject %s has extra time slot allocations, expecting only %d, but found %d in %s, %s, %s, section %s (usi:%d)",
					subject.Code,
					expected_slots, uint8(subject_id_to_time_slot_count[subject.ID]),
					curriculum.CurriculumCode, year_level.Name, semester.Name, Curriculum.SECTION[section_idx], usi,
				))
			}
		}

		if len(stray_subject_ids) > 0 {
			errs_slice = append(errs_slice, fmt.Errorf(
				"the schedule in %s, %s, %s, section %s (usi:%d) has stray subject ids: [%v]",
				curriculum.CurriculumCode,
				year_level.Name,
				semester.Name,
				Curriculum.SECTION[section_idx],
				usi, stray_subject_ids,
			))
		}

		// ── SOFT inter-subject gap check (see gap_constraint.go) ───────────────
		//
		// The 1-2 hour gap between subjects is a soft preference, not a hard
		// requirement like room conflicts or missing subjects. Violations are
		// only logged here as a warning and are deliberately NOT appended to
		// errs_slice, so they never cause RunGeneticAlgorithm to return an
		// error. The fitness function is what nudges the GA toward better-gapped
		// schedules over generations.
		if gapConfig.Enabled {
			gap_violations := CountGapViolations(
				university_sched[usi], gapConfig.MinGapSlots, gapConfig.MaxGapSlots,
			)
			if gap_violations > 0 {
				log.Printf(
					"gap constraint: %s %s %s section %s (usi:%d) has %d gap violation(s)",
					curriculum.CurriculumCode, year_level.Name, semester.Name,
					Curriculum.SECTION[section_idx], usi, gap_violations,
				)
			}
		}

		return IterProceed
	})

	return errs_slice
}

// ValidateNoUnexpectedEmptySections errors when a target department has a fully
// empty section, unless that section was explicitly marked as allowable to skip.
func ValidateNoUnexpectedEmptySections(
	university_sched Schedule.UniTimeTables,
	curriculums []Curriculum.Curriculum,
	department_to_validate map[uint16]bool,
	selected_semester int,
	allowed_empty map[uint16]map[uint16]bool,
) error {
	var validationErr error

	isAllowedEmpty := func(usi int) bool {
		if allowed_empty == nil {
			return false
		}
		_, ok := allowed_empty[uint16(usi)]
		return ok
	}

	IterateSectionsWeekSchedule(university_sched, curriculums, selected_semester, nil, nil,
		func(indicies IterIndices, values IterValues) IterReturnType {
			if values.WeekSched == nil || values.Curriculum == nil || values.Semester == nil || values.YearLevel == nil {
				return IterProceed
			}
			if department_to_validate != nil {
				if !department_to_validate[values.Curriculum.DepartmentID] {
					return IterProceed
				}
			}

			hasSubject := false
			for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
				for time_slot := 0; time_slot < Const.N_DAILY_TIME_SLOTS; time_slot++ {
					if values.WeekSched[day][time_slot].GetSubjectID() != 0 {
						hasSubject = true
						break
					}
				}
				if hasSubject {
					break
				}
			}

			if !hasSubject && !isAllowedEmpty(indicies.Usi) {
				validationErr = fmt.Errorf(
					"unexpected empty section detected in %s %s %s section %s (usi:%d)",
					values.Curriculum.CurriculumCode,
					values.Semester.Name,
					values.YearLevel.Name,
					Curriculum.SECTION[indicies.Section],
					indicies.Usi,
				)
				return IterBreakCurriculumLoop
			}

			return IterProceed
		},
	)

	return validationErr
}
