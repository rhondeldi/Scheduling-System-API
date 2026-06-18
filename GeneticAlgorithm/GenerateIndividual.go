package GeneticAlgorithm

import (
	"fmt"
	"log"
	"math/rand"
	"os"
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

const ALLOW_SPECIALIZED_INSTRUCTOR_FALLBACK bool = true

// LOCK_INSTRUCTORS_TO_DESIGNATED_SUBJECTS, when true, forbids an instructor who
// has been assigned (designated) to specific subjects from ever being scheduled
// for a subject they were not assigned to. Subjects without an assigned
// instructor can still be filled by general (unassigned) instructors only.
// Set to false to allow assigned instructors to be used as a last resort for
// other subjects when no one else is available.
//
// Kept false so the generator does not restrict itself to a subject's
// designated and general instructors: when those are all booked, any other
// instructor in the department who is free at the time slot may be borrowed as
// a last resort (designated instructors are still tried first), rather than
// failing with "not enough specialized/fallback instructors".
const LOCK_INSTRUCTORS_TO_DESIGNATED_SUBJECTS bool = false

// ENFORCE_FOUR_DAY_PACKING, when true, forbids a section's non-NSTP classes from
// being spread across more than Const.MAX_NON_NSTP_SCHOOL_DAYS distinct days.
// NSTP 1/2 subjects are Saturday-pinned and are exempt from this count, so a
// section can still hold an NSTP class on an additional day. Set to false to
// restore the previous behaviour of allowing classes on every school day.
const ENFORCE_FOUR_DAY_PACKING bool = true

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

	// Helper to count total sections for the selected semester.
	// Keep this aligned with Curriculum.GetTotalNumberOfSections and
	// IterateSectionsWeekSchedule (active year levels only, valid semester only).
	countTotalSections := func(curriculums []Curriculum.Curriculum, selected_semester int) int {
		total := 0
		for _, c := range curriculums {
			for _, yl := range c.YearLevels {
				if !yl.IsActive {
					continue
				}
				if selected_semester < 0 || selected_semester >= len(yl.Semesters) {
					continue
				}
				total += yl.Semesters[selected_semester].Sections
			}
		}
		return total
	}

	total_sections := countTotalSections(curriculums, selected_semester)
	university_schedules := make(Schedule.UniTimeTables, total_sections)

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

	/////////////////////////////////////////////////////////////////////////////////////////////////////////
	//        PRE-COMPUTE INSTRUCTORS THAT ARE DESIGNATED/ASSIGNED TO AT LEAST ONE SUBJECT (ANYWHERE)
	/////////////////////////////////////////////////////////////////////////////////////////////////////////
	//
	// An instructor can be tied to subjects through TWO independent mechanisms:
	//
	//   1. instructor-side: Instructor.DesignatedSubjectIDs (assigned via the
	//      "assign subjects to an instructor" route), and
	//   2. subject-side: Subject.DesignatedInstructors / "DesignatedInstructorsID"
	//      (entered by the panelist when adding the subject to a curriculum).
	//
	// Either one makes the instructor a "specialist". With
	// LOCK_INSTRUCTORS_TO_DESIGNATED_SUBJECTS enabled, a specialist may ONLY be
	// scheduled for the subjects they are designated to and must never be poached
	// for any other subject/year/semester. We must therefore detect specialists
	// using BOTH mechanisms, otherwise an instructor assigned purely on the
	// subject-side (empty DesignatedSubjectIDs) would look like a free "general"
	// instructor and leak into unrelated subjects.

	instructor_is_designated_somewhere := make(map[uint16]bool)

	for dept_id := range encoding_resource.DeptIdToInstructors {
		dept_instructors := encoding_resource.DeptIdToInstructors[dept_id]
		for i := range dept_instructors {
			if len(dept_instructors[i].DesignatedSubjectIDs) > 0 {
				instructor_is_designated_somewhere[dept_instructors[i].InstructorID] = true
			}
		}
	}

	for c := range curriculums {
		for y := range curriculums[c].YearLevels {
			for s := range curriculums[c].YearLevels[y].Semesters {
				for _, subject := range curriculums[c].YearLevels[y].Semesters[s].Subjects {
					for _, designated_id := range subject.DesignatedInstructors {
						instructor_is_designated_somewhere[designated_id] = true
					}
				}
			}
		}
	}

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

			if usi < 0 || usi >= len(university_schedules) {
				log.Fatalf("EncodeIndividualGenome: usi index out of range: usi=%d, len(university_schedules)=%d. This usually means the number of sections in all curriculums for the selected semester exceeds the length of university_schedules. Check allocation logic.", usi, len(university_schedules))
			}
			week_time_table := university_schedules[usi]

			// 4-DAY PACKING TRACKER (per section): number of non-NSTP subject blocks
			// placed on each day index. A day is "in use" by non-NSTP classes when
			// its count is > 0. Used to keep a section's non-NSTP classes within
			// Const.MAX_NON_NSTP_SCHOOL_DAYS distinct days (see ENFORCE_FOUR_DAY_PACKING).
			section_non_nstp_day_block_count := make(map[int]int)
			distinct_non_nstp_days := func() int {
				count := 0
				for _, block_count := range section_non_nstp_day_block_count {
					if block_count > 0 {
						count++
					}
				}
				return count
			}

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
			// larger first
			sort.Slice(semester.Subjects, func(i, j int) bool {
				slotsI := semester.Subjects[i].SlotsToAssign()
				slotsJ := semester.Subjects[j].SlotsToAssign()
				return slotsI > slotsJ
			})

			/////////////////////////////////////////////////////////////////////////////////////////////////////////

			// TODO: the map below was added for debugging purposes only, remove when the code becomes stable and optimized.

			subject_recorder := make(map[uint16]Curriculum.Subject)

			subj_assign_fail_possible_reason := make(map[string]int)

			subj_assign_fail_possible_reason["not-enough-time-slots"] = 0
			subj_assign_fail_possible_reason["not-enough-instructors"] = 0
			subj_assign_fail_possible_reason["not-enough-rooms"] = 0

			// Spread this section's non-NSTP subjects over a TARGET number of class
			// days so each day ends up with ~2-3 subjects rather than one day holding
			// many and another a lone subject. Target ≈ round(count / 2.5), clamped to
			// [1, MAX_NON_NSTP_SCHOOL_DAYS]; subjects are then round-robined across
			// those days (see subject_window_day below).
			non_nstp_subject_total := 0
			for _, s := range semester.Subjects {
				if !isNSTP1Or2Subject(s) {
					non_nstp_subject_total++
				}
			}
			target_class_days := (4*non_nstp_subject_total + 5) / 10 // ≈ round(total / 2.5)
			if target_class_days < 1 {
				target_class_days = 1
			}
			if target_class_days > Const.MAX_NON_NSTP_SCHOOL_DAYS {
				target_class_days = Const.MAX_NON_NSTP_SCHOOL_DAYS
			}
			non_nstp_subject_index := 0

			for _, subject := range semester.Subjects {
				isNSTPSubject := isNSTP1Or2Subject(subject)

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

				// round-robin position of this non-NSTP subject across the section's
				// target class days, so day-major search starts each successive subject
				// on the next day in the window and the subjects spread ~evenly (2-3/day).
				subject_window_day := 0
				if !isNSTPSubject {
					subject_window_day = non_nstp_subject_index % target_class_days
					non_nstp_subject_index++
				}

				var selected_instructor *Instructors.Instructor

				has_curriculum_designated_instructors := len(subject.DesignatedInstructors) > 0
				if !has_curriculum_designated_instructors {
					for _, instructor := range instructors {
						if slices.Contains(instructor.DesignatedSubjectIDs, subject.ID) {
							subject.DesignatedInstructors = append(subject.DesignatedInstructors, instructor.InstructorID)
						}
					}

					for _, instructor := range encoding_resource.DeptIdToInstructors[0] {
						if slices.Contains(instructor.DesignatedSubjectIDs, subject.ID) {
							subject.DesignatedInstructors = append(subject.DesignatedInstructors, instructor.InstructorID)
						}
					}
				}

				/////////////////////////////////////////////////////////////////////////////////////////////////////////
				//        BUILD THE ORDERED INSTRUCTOR CANDIDATE LIST FOR THE SUBJECT (TIERED PRIORITIZATION)
				/////////////////////////////////////////////////////////////////////////////////////////////////////////
				//
				// Instructors are tried in this strict order so that an instructor who was
				// explicitly assigned (designated) to a subject is prioritized, and an
				// instructor who is designated to OTHER subjects is NOT poached for an
				// unrelated subject unless there is genuinely no one else available:
				//
				//   tier 1 - instructors designated to THIS subject       (least loaded first)
				//   tier 2 - general instructors with no designations      (least loaded first)
				//   tier 3 - instructors designated to OTHER subjects      (least loaded first) <- last resort
				//
				// This fixes the case where an instructor assigned to a few specific subjects
				// would get pulled into other year levels / unrelated subjects just because
				// they had the lightest teaching load.
				//
				// When LOCK_INSTRUCTORS_TO_DESIGNATED_SUBJECTS is true (the default), tier 3 is
				// dropped entirely: an instructor who is designated to some subjects is locked
				// to those subjects and can NEVER teach a subject they were not assigned to,
				// even if that leaves a subject with no available instructor (which surfaces as
				// a generation error rather than silently poaching the assigned instructor).
				// tier 2 (general, unassigned instructors) can still fill any gap.

				designated_ids := make(map[uint16]bool, len(subject.DesignatedInstructors))
				for _, designated_id := range subject.DesignatedInstructors {
					designated_ids[designated_id] = true
				}

				tier_designated := make([]*Instructors.Instructor, 0)
				tier_general := make([]*Instructors.Instructor, 0)
				tier_other_specialist := make([]*Instructors.Instructor, 0)

				// candidate pool = department instructors + general (department 0) instructors.
				seen_candidate_ids := make(map[uint16]bool)
				classify_candidate := func(instructor *Instructors.Instructor) {
					if seen_candidate_ids[instructor.InstructorID] {
						return
					}
					seen_candidate_ids[instructor.InstructorID] = true

					switch {
					case designated_ids[instructor.InstructorID]:
						// designated to THIS subject -> highest priority.
						tier_designated = append(tier_designated, instructor)
					case !instructor_is_designated_somewhere[instructor.InstructorID]:
						// not assigned to any subject anywhere -> a free, general instructor.
						tier_general = append(tier_general, instructor)
					default:
						// assigned to OTHER subjects -> locked out (last resort only).
						tier_other_specialist = append(tier_other_specialist, instructor)
					}
				}

				for i := range instructors {
					classify_candidate(&instructors[i])
				}
				for i := range encoding_resource.DeptIdToInstructors[0] {
					classify_candidate(&encoding_resource.DeptIdToInstructors[0][i])
				}

				// a designated subject whose designated instructor(s) are not in the pool
				// is only an error when fallback to other instructors is disallowed.
				if len(subject.DesignatedInstructors) > 0 && len(tier_designated) == 0 && !ALLOW_SPECIALIZED_INSTRUCTOR_FALLBACK {
					is_to_return = true

					return_uni_time_table = nil
					return_encoding_resource = nil
					return_error = fmt.Errorf(
						"error encode individual genome, the specialized instructor(s) added in %s, %s, %s, section %s, subject %s, are not found in the department instructors and general instructors list",
						curriculum.CurriculumCode, year_level.Name, semester.Name, Curriculum.SECTION[section_idx], subject.Code,
					)

					return IterBreakCurriculumLoop
				}

				// within each tier, shuffle then sort by load so the least loaded
				// instructor is tried first. Balancing is primarily by assigned UNITS
				// (so subjects spill over to other instructors as one approaches their
				// unit cap, keeping the load balanced), with teaching hours as a
				// tie-breaker.
				shuffle_and_sort_by_load := func(tier []*Instructors.Instructor) {
					rng.Shuffle(len(tier), func(i, j int) {
						tier[i], tier[j] = tier[j], tier[i]
					})
					sort.Slice(tier, func(i, j int) bool {
						if tier[i].AssignedUnits != tier[j].AssignedUnits {
							return tier[i].AssignedUnits < tier[j].AssignedUnits
						}
						return tier[i].TotalTeachingHours < tier[j].TotalTeachingHours
					})
				}

				shuffle_and_sort_by_load(tier_designated)
				shuffle_and_sort_by_load(tier_general)
				shuffle_and_sort_by_load(tier_other_specialist)

				target_instructors := make([]*Instructors.Instructor, 0, len(seen_candidate_ids))
				target_instructors = append(target_instructors, tier_designated...)

				// fall back to the rest of the pool when the subject has no designation,
				// or when fallback is explicitly allowed for a designated subject.
				subject_has_assigned_instructor := len(subject.DesignatedInstructors) > 0
				if !subject_has_assigned_instructor || ALLOW_SPECIALIZED_INSTRUCTOR_FALLBACK {
					// general (unassigned) instructors can always fill a gap.
					target_instructors = append(target_instructors, tier_general...)

					// instructors designated to OTHER subjects (specialists) are only used as
					// an absolute last resort, AFTER general instructors. With the lock on we
					// still refuse to poach a specialist into a subject that already has its
					// own assigned instructor; but a subject that has NO assigned instructor at
					// all may borrow a specialist (least loaded first, thanks to the sort
					// above) so the schedule can still complete.
					allow_specialist_last_resort := !subject_has_assigned_instructor || !LOCK_INSTRUCTORS_TO_DESIGNATED_SUBJECTS
					if allow_specialist_last_resort {
						target_instructors = append(target_instructors, tier_other_specialist...)
					}
				}

				// it is still possible that no instructor is eligible for a subject (e.g. an
				// assigned subject whose only designated instructor is not in the department
				// pool and there are no general instructors to fall back on). surface this as
				// a clean generation error instead of letting the assignment loop index into
				// an empty candidate list.
				if len(target_instructors) == 0 {
					is_to_return = true

					return_uni_time_table = university_schedules
					return_encoding_resource = nil
					return_error = fmt.Errorf(
						"error encode individual genome, no eligible instructor for subject %s in %s, %s, %s, section %s: no designated, general, or borrowable instructor is available for it",
						subject.Code, curriculum.CurriculumCode, semester.Name, year_level.Name, Curriculum.SECTION[section_idx],
					)

					return IterBreakCurriculumLoop
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

					number_of_target_instructors := len(target_instructors)
					target_instructor := target_instructors[target_instructor_idx]

					// undo the allocation of the previous first subject block if the previous second block failed

					if is_double_block_subject && is_prev_initial_block_success && prev_initial_block_instructor != nil && prev_initial_block_room != nil && is_2nd_block_tried && !is_2nd_block_success {
						for time_slot_iter := range prev_initial_block_timeslot_count {
							prev_initial_block_instructor.Time.SetAvailability(true, prev_initial_block_day, (prev_initial_block_timeslot + time_slot_iter))
							prev_initial_block_room.DecTimeSlotClassCount(prev_initial_block_day, (prev_initial_block_timeslot + time_slot_iter))
							week_time_table.GetDayTimeTable(prev_initial_block_day).GetTimeSlot(prev_initial_block_timeslot+time_slot_iter).Set(0, 0, 0)
						}

						prev_initial_block_instructor.AssignedSubjects--
						prev_initial_block_instructor.TotalTeachingHours -= (float32(prev_initial_block_timeslot_count) / float32(Const.N_HOUR_TIME_SLOTS))
						prev_initial_block_instructor.AssignedUnits -= uint16(subject.Units)

						if !isNSTPSubject && prev_initial_block_day >= 0 {
							section_non_nstp_day_block_count[prev_initial_block_day]--
						}
					}

					/////////////////////////////////////////////////////////////////////////////////////////////////////////
					//                    HARD CONSTRAINT — INSTRUCTOR WEEKLY UNIT CAP (NO OVERLOAD)
					/////////////////////////////////////////////////////////////////////////////////////////////////////////
					//
					// Skip any candidate whose weekly unit load would exceed their cap once this
					// subject is added. Regular instructors are capped at the regular maximum;
					// part-time instructors at their (lower) configured cap. Subjects with 0
					// units (legacy data) never trigger this. When every remaining candidate
					// would be overloaded we surface a clean generation error instead of
					// silently overloading an instructor.
					//
					// Check the instructor that will actually receive this subject's load:
					// once a (double-block) subject has pinned an instructor via
					// selected_instructor, retries re-use that same instructor, so the cap
					// must be evaluated against it — the undo above already restored its
					// headroom, so a previously-fitting instructor still fits here.
					cap_check_instructor := target_instructor
					if selected_instructor != nil {
						cap_check_instructor = selected_instructor
					}

					subject_units := uint16(subject.Units)
					if subject_units > 0 && (cap_check_instructor.AssignedUnits+subject_units) > uint16(cap_check_instructor.EffectiveMaxUnits()) {
						if selected_instructor == nil && target_instructor_idx >= number_of_target_instructors-1 {
							is_to_return = true

							return_uni_time_table = university_schedules
							return_encoding_resource = nil
							return_error = fmt.Errorf(
								"error encode individual genome, no instructor can teach subject %s (%d units) without exceeding their weekly unit cap in %s, %s, %s, section %s",
								subject.Code, subject_units, curriculum.CurriculumCode, semester.Name, year_level.Name, Curriculum.SECTION[section_idx],
							)

							return IterBreakCurriculumLoop
						}

						continue target_instructor_loop
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

						subject_total_time_slots := subject.SlotsToAssignByClassType(class_type)

						if subject_total_time_slots <= 0 {
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

						subject_hours := float64(subject_total_time_slots) / float64(Const.N_HOUR_TIME_SLOTS)

						/////////////////////////////////////////////////////////////////////////////////////////////////////////
						//                                ITERATE THROUGH THE WEEKLY TIME SLOTS
						/////////////////////////////////////////////////////////////////////////////////////////////////////////

						distribution_type := int(rng.Int31n(2))

						m := Const.N_DAILY_TIME_SLOTS - subject_total_time_slots + 1
						n := Const.N_WEEKLY_SCHOOL_DAYS
						total_iterations := n * m

						// Stagger each section's preferred starting day so sections spread
						// across the whole week instead of all packing onto the same first
						// days — that clustering is what makes a tight day cap (e.g. 4)
						// infeasible. There are (n - MAX_NON_NSTP_SCHOOL_DAYS + 1) contiguous
						// day-windows of MAX_NON_NSTP_SCHOOL_DAYS days; section usi takes the
						// window usi % windowCount and is explored day-major from its first
						// day, so it fills consecutive days (compact, large-gap-free) and the
						// packing limit then keeps it inside that window. Remaining days are
						// still visited afterwards as a fallback so the exhaustion checks below
						// stay correct.
						windowCount := n - Const.MAX_NON_NSTP_SCHOOL_DAYS + 1
						if windowCount < 1 {
							windowCount = 1
						}
						section_day_offset := usi % windowCount

						for i := range total_iterations {
							var day, time_slot int

							// NSTP keeps the original ordering: it is Saturday-pinned, and the
							// natural order guarantees the final attempt lands on the last day
							// (Saturday) so the NSTP-only Saturday `continue` below does not skip
							// the exhaustion/placement check. Non-NSTP subjects use staggered
							// day-major search.
							if isNSTPSubject {
								if distribution_type == 0 {
									day = i / m
									time_slot = i % m
								} else {
									time_slot = i / n
									day = i % n
								}
							} else {
								day = (section_day_offset + subject_window_day + (i / m)) % n
								time_slot = i % m
							}

							// NSTP 1 and NSTP 2 subjects are constrained to Saturday only.
							if isNSTPSubject && day != saturdayDayIndex() {
								continue
							}

							day_sched := week_time_table.GetDayTimeTable(day)

							/////////////////////////////////////////////////////////////////////////////////////////////////////////
							//                      CHECK IF CURRENT TIME SLOT IS AVAILABLE FOR THE SUBJECT
							/////////////////////////////////////////////////////////////////////////////////////////////////////////

							is_time_slot_available := day_sched.IsTimeAvailable(time_slot, subject_total_time_slots)

							// HARD CONSTRAINT — 4-day packing. A non-NSTP subject may not open an
							// additional teaching day for the section once it already occupies the
							// maximum number of distinct non-NSTP days. Treated as slot
							// unavailability so the existing exhaustion / next-instructor / error
							// machinery handles it uniformly.
							if ENFORCE_FOUR_DAY_PACKING && is_time_slot_available && !isNSTPSubject &&
								section_non_nstp_day_block_count[day] == 0 &&
								distinct_non_nstp_days() >= Const.MAX_NON_NSTP_SCHOOL_DAYS {
								is_time_slot_available = false
							}

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
							//                    SOFT MINIMUM INTER-SUBJECT GAP STEER (see gap_constraint.go)
							/////////////////////////////////////////////////////////////////////////////////////////////////////////

							// Prefer slots that keep at least the minimum gap from the
							// neighbouring subjects on this day. This is a soft preference,
							// never a hard failure: for the FINAL instructor we stop enforcing
							// it so a dense schedule can always fall back to a gap-less
							// placement (CHALLENGE 1) — the fitness function still penalises the
							// resulting gap. The maximum gap is never enforced here (fitness only).
							if gapShouldApplyToDay(day) && !hasMinimumGapFromPreviousSubject(*day_sched, time_slot, subject_total_time_slots, gapConfig.MinGapSlots) {
								if target_instructor_idx < number_of_target_instructors-1 {
									// not the last instructor: keep looking for a gap-respecting
									// slot, mirroring how slot unavailability is handled above.
									if i == total_iterations-1 {
										continue target_instructor_loop
									}
									continue
								}

								// last instructor and no gap-respecting slot remains: place
								// without gap enforcement so the subject is never dropped.
								if os.Getenv("LOG_MODE") == "verbose" {
									log.Printf(
										"gap constraint: placing subject %s in %s section %s without minimum gap on day %d slot %d (no gap-respecting slot available)",
										subject.Code, curriculum.CurriculumCode, Curriculum.SECTION[section_idx], day, time_slot,
									)
								}
							}

							// NOTE: 7:00am avoidance is handled purely by the fitness function
							// (EARLY_MORNING_CLASS_PENALTY/REWARD), NOT by skipping slot 0 here.
							// An earlier encoder-level hard skip of slot 0 removed a column of
							// morning capacity and, combined with the hard 4-day packing rule and
							// instructor locking/unit caps, made dense upper-year sections
							// infeasible (genesis "no time slot" / "not enough instructors"
							// failures). Steering 7am at genesis is not worth breaking feasibility;
							// the GA still shifts classes later over generations via the fitness
							// penalty.

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

							if (!is_available_instructor && ((target_instructor_idx == number_of_target_instructors-1) || selected_instructor != nil)) && i == total_iterations-1 {
								is_to_return = true

								return_uni_time_table = university_schedules
								return_encoding_resource = nil
								return_error = fmt.Errorf(
									"error encode individual genome, not enough specialized/fallback instructors (%d) in %s for %s, %s, %s, section %s, after generating schedules for the previous %d other sections",
									number_of_target_instructors, ro_dept_id_to_department[curriculum.DepartmentID].Name,
									curriculum.CurriculumCode, semester.Name, year_level.Name, Curriculum.SECTION[section_idx], successful_generated_section_schedules,
								)

								return IterBreakCurriculumLoop
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

								selected_instructor = target_instructors[selected_instructor_idx]
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
								selected_instructor.AssignedUnits += uint16(subject.Units)
								is_subject_type_added_once = true
							}

							// record the day this non-NSTP block occupies for the section's
							// 4-day packing budget (counted once per placed block; a double-block
							// lec+lab subject may legitimately occupy two days).
							if ENFORCE_FOUR_DAY_PACKING && !isNSTPSubject {
								section_non_nstp_day_block_count[day]++
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

					// asynchronous hours do not consume slots, but still count toward instructor load.
					if selected_instructor != nil && subject.EffectiveAsynchronousHours() > 0 {
						selected_instructor.TotalTeachingHours += float32(subject.EffectiveAsynchronousHours())
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
