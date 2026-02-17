package GeneticAlgorithm

import (
	"fmt"
	"log"
	"math/rand"
	"slices"
	"sort"
	"time"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Departments"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

const (
	TERM_1ST_SEMESTER int = 0 // `selected_semester` option.
	TERM_2ND_SEMESTER int = 1 // `selected_semester` option.
	TERM_MIDYEAR      int = 2 // `selected_semester` option.
)

const (
	DIST_FRONT_COMPRESSED int = 0
	DIST_BACK_COMPRESSED  int = 1
	DIST_FRONT_LOOSE      int = 2
	DIST_BACK_LOOSE       int = 3
)

/*
Generate individual university schedules.

Parameter prefix meanings for map and list type objects:

	ro_* - read only inside the function, original values are not modified.

	rc_* - read then copy inside the function, original values are not modified, the copied object is the one that the function modifies inside.

Different return types:

	(nil, nil, error) // -> resources copy failed.
	(UniTimeTables, nil, error) // -> not enough resources.
	(UniTimeTables, EncodingResource, nil) // -> successfully generated valid university schedules.

to generate whole university schedules, set department to encode to nil:

	department_to_encode = nil
*/
func EncodeIndividualGenome(
	rc_university_schedules Schedule.UniTimeTables,
	rc_curriculums []Curriculum.Curriculum,
	ro_dept_id_to_department map[uint16]Departments.Department,
	rc_encoding_resource *EncodingResource,
	ro_department_to_encode map[uint16]bool,
	selected_semester, force_distribution_type int,
) (Schedule.UniTimeTables, *EncodingResource, error) {

	// defer debug_log(rc_encoding_resource, ro_dept_id_to_department)
	rng := rand.New(rand.NewSource(time.Now().UnixMilli()))

	////////////////////////////////////////////////////////////////////////////////////////

	curriculums := make([]Curriculum.Curriculum, len(rc_curriculums))
	copied_curriculums := copy(curriculums, rc_curriculums)

	if copied_curriculums != len(rc_curriculums) {
		return nil, nil, fmt.Errorf("error encode individual genome, slice elements copied %d, internal curriculum copy operation failed in generate new individual function", copied_curriculums)
	}

	////////////////////////////////////////////////////////////////////////////////////////

	university_schedules := make(Schedule.UniTimeTables, len(rc_university_schedules))

	copied_week_time_table := copy(university_schedules, rc_university_schedules)

	if copied_week_time_table != len(rc_university_schedules) {
		return nil, nil, fmt.Errorf("error encode individual genome, slice elements copied %d, internal university schedule copy operation failed in generate new individual function", copied_week_time_table)
	}

	////////////////////////////////////////////////////////////////////////////////////////

	encoding_resource, err_make_copy := rc_encoding_resource.MakeCopy()

	if err_make_copy != nil {
		return nil, nil, fmt.Errorf("error encode individual genome, caused by: %s", err_make_copy.Error())
	}

	////////////////////////////////////////////////////////////////////////////////////////

	var room_type_to_rooms map[uint16][]Rooms.Room
	var instructors []Instructors.Instructor

	room_type_to_general_rooms := encoding_resource.DeptIdToRoomtypeToRooms[0]

	is_to_return := false
	var return_uni_time_table Schedule.UniTimeTables
	var return_encoding_resource *EncodingResource
	var return_error error

	successful_generated_section_schedules := 0

	IterateSectionsWeekSchedule(university_schedules, curriculums, selected_semester,

		func(indicies IterIndices, values IterValues) IterReturnType {
			curriculum := values.Curriculum

			room_type_to_rooms = encoding_resource.DeptIdToRoomtypeToRooms[curriculum.DepartmentID]
			instructors = encoding_resource.DeptIdToInstructors[curriculum.DepartmentID]

			return IterProceed
		},
		nil,
		func(indicies IterIndices, values IterValues) IterReturnType {

			usi := indicies.Usi

			curriculum := values.Curriculum
			semester := values.Semester
			year_level := values.YearLevel
			section_idx := indicies.Section

			if ro_department_to_encode != nil {
				is_to_encode := ro_department_to_encode[curriculum.DepartmentID]

				if !is_to_encode {
					return IterProceed
				}
			}

			week_time_table := university_schedules[usi]

			/////////////////////////////////////////////////////////////////////////////////////////////////////////
			//                                    SHUFFLE ROOMS AND SUBJECT
			/////////////////////////////////////////////////////////////////////////////////////////////////////////

			// shuffle the rooms
			for _, rooms := range room_type_to_rooms {
				rng.Shuffle(len(rooms), func(i, j int) {
					rooms[i], rooms[j] = rooms[j], rooms[i]
				})
			}

			// shuffle the subjects
			rng.Shuffle(len(semester.Subjects), func(i, j int) {
				semester.Subjects[i], semester.Subjects[j] = semester.Subjects[j], semester.Subjects[i]
			})

			/////////////////////////////////////////////////////////////////////////////////////////////////////////

			// TODO: the map below was added for debugging purposes only, remove when the code becomes stable and optimized.

			subject_recorder := make(map[uint16]Curriculum.Subject)

			subj_assign_fail_possible_reason := make(map[string]int)

			subj_assign_fail_possible_reason["not-enough-time-slots"] = 0
			subj_assign_fail_possible_reason["not-enough-instructors"] = 0
			subj_assign_fail_possible_reason["not-enough-rooms"] = 0

			for _, subject := range semester.Subjects {

				non_final_sched_idx := uint16(usi)

				if _, has_sched_idx := encoding_resource.IsSchedIdxToSubIdToSkip[non_final_sched_idx]; has_sched_idx {
					_, has_sub_id := encoding_resource.IsSchedIdxToSubIdToSkip[non_final_sched_idx][subject.ID]
					if has_sub_id {
						if encoding_resource.IsSchedIdxToSubIdToSkip[non_final_sched_idx][subject.ID] {
							subject_recorder[subject.ID] = subject
							continue // skip since subject was already assigned
						}
					}
				}

				var selected_instructor *Instructors.Instructor

				/////////////////////////////////////////////////////////////////////////////////////////////////////////
				//               POPULATE SPECIALIZED INSTRUCTOR LIST FOR THE SUBJECT IF THEY EXIST
				/////////////////////////////////////////////////////////////////////////////////////////////////////////

				specialized_instructors := make([]*Instructors.Instructor, 0)

				if len(subject.DesignatedInstructors) > 0 {
					instructor_id_to_instructor := make(map[uint16]*Instructors.Instructor)

					for i := range instructors {
						instructor_id_to_instructor[instructors[i].InstructorID] = &instructors[i]
					}

					for i := range encoding_resource.DeptIdToInstructors[0] {
						instructor_id_to_instructor[encoding_resource.DeptIdToInstructors[0][i].InstructorID] = &encoding_resource.DeptIdToInstructors[0][i]
					}

					for _, specialized_id := range subject.DesignatedInstructors {
						if _, has_id := instructor_id_to_instructor[specialized_id]; has_id {
							specialized_instructors = append(specialized_instructors, instructor_id_to_instructor[specialized_id])
						}
					}

					if len(specialized_instructors) == 0 {
						is_to_return = true

						return_uni_time_table = nil
						return_encoding_resource = nil
						return_error = fmt.Errorf(
							"error encode individual genome, the specialized instructor(s) added in %s, %s, %s, section %s, subject %s, are not found in the department instructors and general instructors list",
							curriculum.CurriculumCode, year_level.Name, semester.Name, Curriculum.SECTION[section_idx], subject.Code,
						)

						return IterBreakCurriculumLoop
					}

					// shuffle specialized instructors
					rng.Shuffle(len(specialized_instructors), func(i, j int) {
						specialized_instructors[i], specialized_instructors[j] = specialized_instructors[j], specialized_instructors[i]
					})

					// sort the specialized_instructors based on the number of subjects they are assigned
					sort.Slice(specialized_instructors, func(i, j int) bool {
						return specialized_instructors[i].TotalTeachingHours < specialized_instructors[j].TotalTeachingHours
					})
				} else {
					// shuffle instructors
					rng.Shuffle(len(instructors), func(i, j int) {
						instructors[i], instructors[j] = instructors[j], instructors[i]
					})

					// sort the instructors based on the number of subjects they are assigned
					sort.Slice(instructors, func(i, j int) bool {
						return instructors[i].TotalTeachingHours < instructors[j].TotalTeachingHours
					})
				}

				is_double_block_subject := (subject.LecHours > 0) && (subject.LabHours > 0)

				is_prev_initial_block_success := false
				prev_initial_block_timeslot_count := -1
				prev_initial_block_day := -1
				prev_initial_block_timeslot := -1

				var prev_initial_block_instructor *Instructors.Instructor
				var prev_initial_block_room *Rooms.Room

				is_2nd_block_tried := false
				is_2nd_block_success := false

				target_instructor_idx := -1

			target_instructor_loop:
				for {
					target_instructor_idx++

					var target_instructor *Instructors.Instructor
					number_of_target_instructors := 0

					if len(subject.DesignatedInstructors) > 0 {
						number_of_target_instructors = len(subject.DesignatedInstructors)
						target_instructor = specialized_instructors[target_instructor_idx]
					} else {
						number_of_target_instructors = len(instructors)
						target_instructor = &instructors[target_instructor_idx]
					}

					// undo the allocation of the previous first subject block if the previous second block failed

					if is_double_block_subject && is_prev_initial_block_success && prev_initial_block_instructor != nil && prev_initial_block_room != nil && is_2nd_block_tried && !is_2nd_block_success {
						for time_slot_iter := range prev_initial_block_timeslot_count {
							prev_initial_block_instructor.Time.SetAvailability(true, prev_initial_block_day, (prev_initial_block_timeslot + time_slot_iter))
							prev_initial_block_room.DecTimeSlotClassCount(prev_initial_block_day, (prev_initial_block_timeslot + time_slot_iter))
							week_time_table.GetDayTimeTable(prev_initial_block_day).GetTimeSlot(prev_initial_block_timeslot+time_slot_iter).Set(0, 0, 0)
						}

						prev_initial_block_instructor.AssignedSubjects--
						prev_initial_block_instructor.TotalTeachingHours -= (float32(prev_initial_block_timeslot_count) / float32(Const.N_HOUR_TIME_SLOTS))
					}

					/////////////////////////////////////////////////////////////////////////////////////////////////////////
					//                         RANDOMIZE LECTURE AND LABORATORY ASSIGNMENT ORDER
					/////////////////////////////////////////////////////////////////////////////////////////////////////////

					// iterate over the class type of the subject lec = 0 or lab = 1
					is_subject_type_added_once := false

					rand_class_type := int(rng.Int31n(2))

					for class_type_iter := 0; class_type_iter < 2; class_type_iter++ {

						class_type := (rand_class_type + class_type_iter) % 2

						var selected_room *Rooms.Room

						var subject_hours int

						if class_type == 0 {
							subject_hours = int(subject.LecHours)
						} else {
							subject_hours = int(subject.LabHours)
						}

						if subject_hours == 0 {
							continue // skip subject class type if there is no contact hours
						}

						if is_double_block_subject && class_type_iter == 1 {
							is_2nd_block_tried = true
							is_2nd_block_success = false
						}

						if is_double_block_subject && class_type_iter == 0 {
							is_prev_initial_block_success = false
						} else {
							is_prev_initial_block_success = true
						}

						subject_total_time_slots := subject_hours * Const.N_HOUR_TIME_SLOTS

						/////////////////////////////////////////////////////////////////////////////////////////////////////////
						//                                ITERATE THROUGH THE WEEKLY TIME SLOTS
						/////////////////////////////////////////////////////////////////////////////////////////////////////////

						distribution_type := int(rng.Int31n(2))

						m := Const.N_DAILY_TIME_SLOTS - subject_total_time_slots + 1
						n := Const.N_WEEKLY_SCHOOL_DAYS
						total_iterations := n * m

						for i := range total_iterations {
							var day, time_slot int

							if distribution_type == 0 {
								day = i / m
								time_slot = i % m
							} else {
								time_slot = i / n
								day = i % n
							}

							day_sched := week_time_table.GetDayTimeTable(day)

							/////////////////////////////////////////////////////////////////////////////////////////////////////////
							//                      CHECK IF CURRENT TIME SLOT IS AVAILABLE FOR THE SUBJECT
							/////////////////////////////////////////////////////////////////////////////////////////////////////////

							is_time_slot_available := day_sched.IsTimeAvailable(time_slot, subject_total_time_slots)

							if !is_time_slot_available && i == total_iterations-1 && (target_instructor_idx == (number_of_target_instructors - 1)) {
								is_to_return = true

								return_uni_time_table = university_schedules
								return_encoding_resource = nil
								return_error = fmt.Errorf(
									"error encode individual genome, no time slot found for %s in %s for %s, %s, %s, section %s, after generating schedules for the previous %d other sections",
									subject.Code, ro_dept_id_to_department[curriculum.DepartmentID].Name,
									curriculum.CurriculumCode, semester.Name, year_level.Name, Curriculum.SECTION[section_idx], successful_generated_section_schedules,
								)

								return IterBreakCurriculumLoop
							}

							if !is_time_slot_available && (i == (total_iterations - 1)) {
								continue target_instructor_loop // to next instructor if there is no time slot left
							}

							if !is_time_slot_available {
								continue // if the current time slot is not available, go to the next time slot
							}

							/////////////////////////////////////////////////////////////////////////////////////////////////////////
							//                          FIND AVAILABLE INSTRUCTOR FOR THE TIME SLOT
							/////////////////////////////////////////////////////////////////////////////////////////////////////////

							// if the time slot is available proceed to find then assign an available instructor

							instructor_search_iteration := 0

							selected_instructor_idx := -1
							is_available_instructor := true

							if selected_instructor == nil {
								for instructor_time_slot := time_slot; instructor_time_slot < (time_slot + subject_total_time_slots); instructor_time_slot++ {
									is_available_instructor = is_available_instructor && target_instructor.Time.GetAvailability(day, instructor_time_slot)
								}
							} else {
								for instructor_time_slot := time_slot; instructor_time_slot < (time_slot + subject_total_time_slots); instructor_time_slot++ {
									is_available_instructor = is_available_instructor && selected_instructor.Time.GetAvailability(day, instructor_time_slot)
								}
							}

							instructor_search_iteration++

							if len(subject.DesignatedInstructors) > 0 {
								if (!is_available_instructor && ((target_instructor_idx == len(specialized_instructors)-1) || selected_instructor != nil)) && i == total_iterations-1 {
									is_to_return = true

									return_uni_time_table = university_schedules
									return_encoding_resource = nil
									return_error = fmt.Errorf(
										"error encode individual genome, not enough specialized_instructors (%d) in %s for %s, %s, %s, section %s, after generating schedules for the previous %d other sections",
										len(specialized_instructors), ro_dept_id_to_department[curriculum.DepartmentID].Name,
										curriculum.CurriculumCode, semester.Name, year_level.Name, Curriculum.SECTION[section_idx], successful_generated_section_schedules,
									)

									return IterBreakCurriculumLoop
								}
							} else {
								if (!is_available_instructor && ((target_instructor_idx == len(instructors)-1) || selected_instructor != nil)) && i == total_iterations-1 {
									is_to_return = true

									return_uni_time_table = university_schedules
									return_encoding_resource = nil
									return_error = fmt.Errorf(
										"error encode individual genome, not enough instructors (%d) in %s for %s, %s, %s, section %s, after generating schedules for the previous %d other sections",
										len(instructors), ro_dept_id_to_department[curriculum.DepartmentID].Name,
										curriculum.CurriculumCode, semester.Name, year_level.Name, Curriculum.SECTION[section_idx], successful_generated_section_schedules,
									)

									return IterBreakCurriculumLoop
								}
							}

							if !is_available_instructor && (i == (total_iterations - 1)) {
								continue target_instructor_loop // find another instructor if not available for the time slot
							}

							if !is_available_instructor {
								continue // find another instructor if not available for the time slot
							}

							selected_instructor_idx = target_instructor_idx

							/////////////////////////////////////////////////////////////////////////////////////////////////////////
							//                             FIND AVAILABLE ROOM FOR THE TIME SLOT
							/////////////////////////////////////////////////////////////////////////////////////////////////////////

							room_search_iteration := 0
							room_type := uint16(class_type)

							var has_available_room bool

							if subject.IsGymType() {

								/////////////////////////////////////////////////////////////////////////////////////////////////////////
								//                                       find available gym room
								/////////////////////////////////////////////////////////////////////////////////////////////////////////

								// search available gym for physical education subjects

								{
									gym := encoding_resource.DeptIdToRoomtypeToRooms[0][Rooms.ROOM_TYPE_GYM]

									for room_idx := range gym {

										has_available_room = true

										for room_time_slot := time_slot; room_time_slot < (time_slot + subject_total_time_slots); room_time_slot++ {
											has_available_room = has_available_room && gym[room_idx].GetTimeSlotClassCount(day, room_time_slot) < uint8(gym[room_idx].Capacity)
										}

										if !has_available_room {
											continue
										}

										selected_room = &gym[room_idx]
										break
									}
								}

								if !has_available_room {
									gym := encoding_resource.DeptIdToRoomtypeToRooms[curriculum.DepartmentID][Rooms.ROOM_TYPE_GYM]

									for room_idx := range gym {

										has_available_room = true

										for room_time_slot := time_slot; room_time_slot < (time_slot + subject_total_time_slots); room_time_slot++ {
											has_available_room = has_available_room && gym[room_idx].GetTimeSlotClassCount(day, room_time_slot) < uint8(gym[room_idx].Capacity)
										}

										if !has_available_room {
											continue
										}

										selected_room = &gym[room_idx]
										break
									}
								}
							} else {

								/////////////////////////////////////////////////////////////////////////////////////////////////////////
								//                                  find available department rooms
								/////////////////////////////////////////////////////////////////////////////////////////////////////////

								// search for department specific rooms that are available

								for room_idx := range room_type_to_rooms[room_type] {

									has_available_room = true

									for room_time_slot := time_slot; room_time_slot < (time_slot + subject_total_time_slots); room_time_slot++ {
										has_available_room = has_available_room && room_type_to_rooms[room_type][room_idx].GetTimeSlotClassCount(day, room_time_slot) < uint8(room_type_to_rooms[room_type][room_idx].Capacity)
									}

									if !has_available_room {
										continue
									}

									// fmt.Printf("selecting the available room[type:%d] for the time slot [d:%d, ts:%d]...\n", room_type, day, time_slot) // DEBUG PRINTS
									selected_room = &room_type_to_rooms[room_type][room_idx]
									break
								}

								// search for general rooms that are available

								if selected_room == nil || !has_available_room {
									for room_idx := range room_type_to_general_rooms[room_type] {

										if len(room_type_to_general_rooms[room_type][room_idx].SharingDepartments) > 0 {
											if !slices.Contains(room_type_to_general_rooms[room_type][room_idx].SharingDepartments, curriculum.DepartmentID) {
												continue
											}
										}

										has_available_room = true

										for room_time_slot := time_slot; room_time_slot < (time_slot + subject_total_time_slots); room_time_slot++ {
											has_available_room = has_available_room && room_type_to_general_rooms[room_type][room_idx].GetTimeSlotClassCount(day, room_time_slot) < uint8(room_type_to_general_rooms[room_type][room_idx].Capacity)
										}

										if !has_available_room {
											continue
										}

										// fmt.Printf("selecting the available room[type:%d] for the time slot [d:%d, ts:%d]...\n", room_type, day, time_slot) // DEBUG PRINTS
										selected_room = &room_type_to_general_rooms[room_type][room_idx]
										break
									}
								}

								// TODO: [implement below] search available lab room for lecture subjects (consult first)
							}

							room_search_iteration++

							if !has_available_room && (i == (total_iterations - 1)) && (target_instructor_idx == (number_of_target_instructors - 1)) {
								is_to_return = true

								return_uni_time_table = university_schedules
								return_encoding_resource = nil
								return_error = fmt.Errorf(
									"error encode individual genome, not enough %s rooms (%d) in %s for %s, %s, %s, section %s, after generating schedules for the previous %d other sections",
									Rooms.ROOM_TYPE_NAMES[room_type], len(room_type_to_rooms[room_type]), ro_dept_id_to_department[curriculum.DepartmentID].Name,
									curriculum.CurriculumCode, semester.Name, year_level.Name, Curriculum.SECTION[section_idx], successful_generated_section_schedules,
								)

								return IterBreakCurriculumLoop
							}

							if !has_available_room && (i == (total_iterations - 1)) {
								continue target_instructor_loop // if there is no available room for the current time slot and instructor, find other instructor and time slots.
							}

							if !has_available_room {
								// fmt.Printf("No room found for the time slot [d:%d, ts:%d]...\n", day, time_slot) // DEBUG PRINTS
								subj_assign_fail_possible_reason["not-enough-rooms"]++
								continue // find another time slot
							}

							/////////////////////////////////////////////////////////////////////////////////////////////////////////
							//   if there is an available room then finalize instructor selection if there is no one selected yet
							/////////////////////////////////////////////////////////////////////////////////////////////////////////

							// fmt.Printf("room found for the time slot [d:%d, ts:%d]...\n", day, time_slot) // DEBUG PRINTS

							if selected_instructor == nil {
								// fmt.Printf("selecting the instructor found for the time slot [d:%d, ts:%d]...\n", day, time_slot) // DEBUG PRINTS

								if len(subject.DesignatedInstructors) > 0 {
									selected_instructor = specialized_instructors[selected_instructor_idx]
								} else {
									selected_instructor = &instructors[selected_instructor_idx]
								}
							}

							/////////////////////////////////////////////////////////////////////////////////////////////////////////
							//                       ALLOCATE THE FINAL AVAILABLE INSTRUCTOR FOR THE TIME SLOT
							/////////////////////////////////////////////////////////////////////////////////////////////////////////

							time_slot_assignment_sanity_counter := 0

							for selected_time_slot := time_slot; selected_time_slot < (time_slot + subject_total_time_slots); selected_time_slot++ {

								if day_sched.GetTimeSlot(selected_time_slot).GetSubjectID() != 0 {
									panic("woah woah woah! you are overwriting a subject allocated in that time slot")
								}

								if day_sched.GetTimeSlot(selected_time_slot).GetInstructorID() != 0 {
									panic("woah woah woah! you are overwriting an instructor allocated in that time slot")
								}

								if day_sched.GetTimeSlot(selected_time_slot).GetRoomID() != 0 {
									panic("woah woah woah! you are overwriting a room allocated in that time slot")
								}

								selected_instructor.Time.SetAvailability(false, day, selected_time_slot)
								selected_room.IncTimeSlotClassCount(day, selected_time_slot)

								if subject.ID == 0 {
									panic(fmt.Sprintf(
										"%s %s %s section[%d] %s %s's subject id should never be zero",
										curriculum.CurriculumCode, semester.Name, year_level.Name, section_idx, subject.Code, subject.Name,
									))
								}

								day_sched.GetTimeSlot(selected_time_slot).SetSubjectID(subject.ID)
								day_sched.GetTimeSlot(selected_time_slot).SetInstructorID(selected_instructor.InstructorID)
								day_sched.GetTimeSlot(selected_time_slot).SetRoomID(selected_room.RoomID)

								time_slot_assignment_sanity_counter++
							}

							if class_type_iter == 0 && is_double_block_subject {
								prev_initial_block_instructor = selected_instructor
								prev_initial_block_room = selected_room

								prev_initial_block_day = day
								prev_initial_block_timeslot = time_slot
								prev_initial_block_timeslot_count = subject_total_time_slots

								is_prev_initial_block_success = true
							}

							if class_type_iter == 1 && is_double_block_subject {
								is_2nd_block_success = true
							}

							if time_slot_assignment_sanity_counter != subject_total_time_slots {
								panic(
									"total time slot assigned did not match the subject total time slot",
								)
							}

							if !is_subject_type_added_once {
								selected_instructor.AssignedSubjects++
								is_subject_type_added_once = true
							}

							selected_instructor.TotalTeachingHours += float32(subject_hours)

							subject_recorder[subject.ID] = subject

							/////////////////////////////////////////////////////////////////////////////////////////////////////////
							//                                  EXIT THE LOOP AFTER SUCCESSFUL ASSIGNMENT
							/////////////////////////////////////////////////////////////////////////////////////////////////////////

							break
						} // ------------- end of time slot loop -------------
					} // ------------- end of class_type_iter loop -------------

					// map encoding resource that this subject is already assigned.

					if _, has_sched_idx := encoding_resource.IsSchedIdxToSubIdToSkip[non_final_sched_idx]; !has_sched_idx {
						encoding_resource.IsSchedIdxToSubIdToSkip[non_final_sched_idx] = make(map[uint16]bool)
						encoding_resource.IsSchedIdxToSubIdToSkip[non_final_sched_idx][subject.ID] = true
					} else {
						if _, has_sub_id := encoding_resource.IsSchedIdxToSubIdToSkip[non_final_sched_idx][subject.ID]; !has_sub_id {
							encoding_resource.IsSchedIdxToSubIdToSkip[non_final_sched_idx][subject.ID] = true
						} else {
							panic("woah woah woah!, you're not supposed to be here")
						}
					}

					break
				} // ------------- end of instructor loop -------------
			} // ------------- end of subject loop -------------

			// front compressed distribution : end

			if len(subject_recorder) != len(semester.Subjects) {
				log.Printf("this is an unkown failure in EncodeIndividualGenome : %s", fmt.Sprintf(
					"there are some subjects in %s %s %s %s that was not assigned for some reason s(%d/%d), i(%d), r(%d), IvsR(%d/%d)",
					ro_dept_id_to_department[curriculum.DepartmentID].Code,
					curriculum.CurriculumCode,
					year_level.Name,
					semester.Name,
					len(subject_recorder), len(semester.Subjects),
					len(instructors),
					len(room_type_to_rooms[Rooms.ROOM_TYPE_LAB])+len(room_type_to_rooms[Rooms.ROOM_TYPE_LEC]),
					subj_assign_fail_possible_reason["not-enough-instructors"],
					subj_assign_fail_possible_reason["not-enough-rooms"],
				))

				is_to_return = true

				return_uni_time_table = university_schedules
				return_encoding_resource = nil
				return_error = fmt.Errorf(
					"error encode individual genome, there are some subjects in %s, %s, %s, %s, that was not assigned for some reason s(%d/%d), i(%d), r(%d), IvsR(%d/%d)",
					ro_dept_id_to_department[curriculum.DepartmentID].Code,
					curriculum.CurriculumCode,
					year_level.Name,
					semester.Name,
					len(subject_recorder), len(semester.Subjects),
					len(instructors),
					len(room_type_to_rooms[Rooms.ROOM_TYPE_LAB])+len(room_type_to_rooms[Rooms.ROOM_TYPE_LEC]),
					subj_assign_fail_possible_reason["not-enough-instructors"],
					subj_assign_fail_possible_reason["not-enough-rooms"],
				)

				return IterBreakCurriculumLoop
			}

			university_schedules[usi] = week_time_table
			successful_generated_section_schedules++

			return IterProceed
		},
	)

	if is_to_return {
		return return_uni_time_table, return_encoding_resource, return_error
	}

	flatten_room_id_to_room := make(map[uint16]*Rooms.Room)

	for out_key, out_v := range encoding_resource.DeptIdToRoomtypeToRooms {
		for in_key, in_v := range out_v {
			for room_idx, room := range in_v {
				flatten_room_id_to_room[room.RoomID] = &encoding_resource.DeptIdToRoomtypeToRooms[out_key][in_key][room_idx]
			}
		}
	}

	flatten_instructor_id_to_instructor := make(map[uint16]*Instructors.Instructor)

	for k, v := range encoding_resource.DeptIdToInstructors {
		for instructor_idx, instructor := range v {
			flatten_instructor_id_to_instructor[instructor.InstructorID] = &encoding_resource.DeptIdToInstructors[k][instructor_idx]
		}
	}

	encoding_resource.IdToRoom = flatten_room_id_to_room
	encoding_resource.IdToInstructor = flatten_instructor_id_to_instructor

	return university_schedules, encoding_resource, nil
}

func NewEmptyIndividual(
	curriculums []Curriculum.Curriculum,
	selected_semester int,
) Schedule.UniTimeTables {

	individual_university_schedules := make(Schedule.UniTimeTables, 0, 128)

	IterateSectionsWeekSchedule(individual_university_schedules, curriculums, selected_semester, nil, nil, func(indicies IterIndices, values IterValues) IterReturnType {
		individual_university_schedules = append(individual_university_schedules, Schedule.WeekTimeTable{})
		return IterProceed
	})

	return individual_university_schedules
}

// func debug_log(encoding_resource *EncodingResource, dept_id_to_department map[uint16]Departments.Department) {
// 	fmt.Print("\n\ndebug_log:\n")

// 	for dept_id, instructors := range encoding_resource.DeptIdToInstructors {
// 		fmt.Printf("the numbers of instructors in %s is %d\n", dept_id_to_department[dept_id].Code, len(instructors))
// 	}

// 	fmt.Print("\n\n")

// 	for dept_id, room_types := range encoding_resource.DeptIdToRoomtypeToRooms {
// 		for room_type, rooms := range room_types {
// 			fmt.Printf("number of %s rooms in %s is %d\n", Rooms.ROOM_TYPE_NAMES[room_type], dept_id_to_department[dept_id].Code, len(rooms))
// 		}
// 	}

// 	fmt.Print("\n\n")
// }
