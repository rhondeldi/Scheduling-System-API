// the fitness function & some constants are slightly different to the paper presented during the final defense.
// this one considers new condition(s), and some previous values are changed to hopefully improve generated schedules.
package GeneticAlgorithm

import (
	"math"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

const PREFERRED_MAX_CLASS_HOUR_PER_DAY float64 = 10.0

// EARLY_MORNING_CLASS_SLOT is the time slot for the 7:00am start (slot 0, since
// the teaching day begins at 7:00am). Days whose first class lands on this slot
// are penalised so the GA avoids scheduling too many 7am classes; days that
// start later are rewarded.
const EARLY_MORNING_CLASS_SLOT int = 0

// Tuned up so avoiding a 7:00am start carries weight comparable to the
// after-5pm term (±4): without this the GA preferred the cheaper after-5pm
// reward and kept starting days at 7am. 7am avoidance is fitness-only — an
// encoder-level slot-0 skip was tried but reverted because it broke genesis
// feasibility for dense sections (see GenerateIndividual.go).
const EARLY_MORNING_CLASS_PENALTY float64 = 4.0
const EARLY_MORNING_CLASS_REWARD float64 = 2.0

// Per-day subject-count preference: a class day with only ONE subject is
// penalised, while days holding 2 or 3 subjects are rewarded, so the GA avoids
// isolating a single subject on its own day and instead packs 2-3 subjects per
// day. Days with 4+ subjects are left neutral so dense curricula can still fit.
const LONELY_SUBJECT_DAY_PENALTY float64 = 4.0
const BALANCED_SUBJECT_DAY_REWARD float64 = 2.0

// SATURDAY_NON_NSTP_PENALTY is subtracted for each regular (non-NSTP,
// non-SaturdayOnly) subject placed on Saturday, keeping Saturday reserved mainly
// for the 1st-year NSTP classes instead of crowding it with regular subjects.
// Tuned up (was 4.0): the per-day score is averaged over all class days before
// the week-level adjustment, which dilutes a Saturday-only penalty, so it needs
// a larger raw value to meaningfully steer regular classes off Saturday.
const SATURDAY_NON_NSTP_PENALTY float64 = 6.0

/*
* Output Range:

	(0, 1]

* Behavior:

	When actual_hours = target_hours, return = 1
	As |actual_hours - target_hours| --> +Inf, return --> 0
*/
func reciprocal_distance(actual_hours, target_hours float64) float64 {
	return 1.0 / (1.0 + math.Abs(actual_hours-target_hours))

}

func buildSubjectIDToAsyncHoursMap(curriculums []Curriculum.Curriculum) map[uint16]float64 {
	subjectIDToAsyncHours := make(map[uint16]float64)

	for _, curriculum := range curriculums {
		for _, yearLevel := range curriculum.YearLevels {
			if !yearLevel.IsActive {
				continue
			}

			for _, semester := range yearLevel.Semesters {
				for _, subject := range semester.Subjects {
					asyncHours := subject.EffectiveAsynchronousHours()
					if asyncHours <= 0 {
						continue
					}

					subjectIDToAsyncHours[subject.ID] = asyncHours
				}
			}
		}
	}

	return subjectIDToAsyncHours
}

func buildSubjectIDToAsyncHoursMapFromSubjects(subjects []Curriculum.Subject) map[uint16]float64 {
	subjectIDToAsyncHours := make(map[uint16]float64)

	for _, subject := range subjects {
		asyncHours := subject.EffectiveAsynchronousHours()
		if asyncHours <= 0 {
			continue
		}
		subjectIDToAsyncHours[subject.ID] = asyncHours
	}

	return subjectIDToAsyncHours
}

// MeasureWeekTimeTableBasicFitness scores one section's weekly time table.
//
// Per-day score contributions (before averaging over days-with-class):
//
//	lunch break        : +8.0  / -12.0
//	classes after 5pm  : +4.0  / -4.0
//	daily hours vs pref : +3.5 / -3.5
//	saturday hours      : +1.0 / -1.0
//	inter-subject gaps  : +Reward (default +1.5) when a multi-subject day has
//	                      only valid gaps, or -Penalty (default -3.0) per gap
//	                      violation. With G violations on a day this term ranges
//	                      from +Reward down to -Penalty·G (G is bounded by the
//	                      number of subject-block boundaries in the day).
//
// The summed per-day score is divided by the number of days with class, then a
// final ±2/2.5 week-level adjustment is applied for the number of class days.
//
// Theoretical range: with the gap constraint enabled the lower bound drops
// relative to the pre-constraint system (each day can now subtract up to
// Penalty·G extra) and the upper bound rises by up to Reward per day; when the
// constraint is disabled (GA_MIN_GAP_HOURS=0) the range is identical to before.
func MeasureWeekTimeTableBasicFitness(week_sched Schedule.WeekTimeTable, subject_id_to_async_hours map[uint16]float64) float64 {
	week_sched_fitness := 0.0
	days_with_class := 0.0

	for day := range Const.N_WEEKLY_SCHOOL_DAYS {

		has_class_after_5pm := false
		has_time_for_lunch := false
		day_total_hours := 0.0
		earliest_class_slot := -1
		day_subject_ids := make(map[uint16]bool)

		for time_slot := range Const.N_DAILY_TIME_SLOTS {
			subject_id := week_sched[day][time_slot].GetSubjectID()
			if subject_id > 0 {
				day_total_hours += (1.0 / float64(Const.N_HOUR_TIME_SLOTS))
				day_subject_ids[subject_id] = true

				// remember the first occupied slot to detect 7am starts.
				if earliest_class_slot == -1 {
					earliest_class_slot = time_slot
				}

				// mark if there is any class after 5pm (slot index >= 20)
				if time_slot >= 20 {
					has_class_after_5pm = true
				}
			}

			if (time_slot >= 8) && (time_slot <= 11) && (subject_id == 0) {
				has_time_for_lunch = true
			}
		}

		for subjectID := range day_subject_ids {
			day_total_hours += subject_id_to_async_hours[subjectID]
		}

		if day_total_hours == 0 {
			continue
		} else {
			days_with_class += 1.0
		}

		// days that don't have break time during lunch hours are punished, and rewarded if there are
		if has_time_for_lunch {
			week_sched_fitness += 8.0
		} else {
			week_sched_fitness -= 12.0
		}

		// class hours after 5pm are punished, rewarded if classes are only until 5pm
		if has_class_after_5pm {
			week_sched_fitness -= 4.0
		} else {
			week_sched_fitness += 4.0
		}

		// 7am classes are punished, later starts are rewarded, so the GA avoids
		// scheduling too many early-morning (7:00am) classes.
		if earliest_class_slot == EARLY_MORNING_CLASS_SLOT {
			week_sched_fitness -= EARLY_MORNING_CLASS_PENALTY
		} else {
			week_sched_fitness += EARLY_MORNING_CLASS_REWARD
		}

		// class hours beyond the prefered are punished, below are rewarded
		if day_total_hours > PREFERRED_MAX_CLASS_HOUR_PER_DAY {
			week_sched_fitness -= 3.5
		} else {
			week_sched_fitness += 3.5
		}

		// prefer 2-3 subjects per class day: a day with a single subject is
		// penalised, days with 2 or 3 subjects are rewarded. 4+ is left neutral
		// so dense curricula can still be scheduled.
		day_subject_count := len(day_subject_ids)
		if day_subject_count == 1 {
			week_sched_fitness -= LONELY_SUBJECT_DAY_PENALTY
		} else if day_subject_count == 2 || day_subject_count == 3 {
			week_sched_fitness += BALANCED_SUBJECT_DAY_REWARD
		}

		// keep Saturday reserved for NSTP: penalise each regular (non-NSTP,
		// non-SaturdayOnly) subject scheduled on Saturday so it is not crowded
		// with classes that belong on weekdays. Skipped when the allowed-set is
		// not initialised (outside a GA run).
		if day == (Const.N_WEEKLY_SCHOOL_DAYS-1) && saturdayAllowedSubjectIDs != nil {
			for subjectID := range day_subject_ids {
				if !saturdayAllowedSubjectIDs[subjectID] {
					week_sched_fitness -= SATURDAY_NON_NSTP_PENALTY
				}
			}
		}

		// long class hours during saturday are punished, short hours are rewarded
		if (day_total_hours > (PREFERRED_MAX_CLASS_HOUR_PER_DAY / 2)) && (day == (Const.N_WEEKLY_SCHOOL_DAYS - 1)) {
			week_sched_fitness -= 1.0
		} else {
			week_sched_fitness += 1.0
		}

		// inter-subject gap scoring (see gap_constraint.go).
		//
		// A valid gap between two different subject blocks is 1-2 hours
		// (MinGapSlots..MaxGapSlots). Each gap that is too small or too large
		// is penalised; a day with more than one subject and only valid gaps
		// is rewarded. The whole block is skipped when the constraint is
		// disabled (GA_MIN_GAP_HOURS=0) or for Saturday when it is opted out,
		// which keeps the fitness identical to the pre-constraint system in
		// those configurations. Single-subject days never produce a violation
		// nor earn the reward.
		if gapShouldApplyToDay(day) {
			day_blocks := ExtractSubjectBlocks(week_sched[day])
			gap_violations := CheckGapsBetweenSubjects(
				week_sched[day], gapConfig.MinGapSlots, gapConfig.MaxGapSlots,
			)

			if len(gap_violations) > 0 {
				// penalise per violation
				week_sched_fitness -= gapConfig.Penalty * float64(len(gap_violations))
			} else if len(day_blocks) > 1 {
				// reward if multiple subjects and all gaps are correct
				week_sched_fitness += gapConfig.Reward
			}

			// graded LONG-gap penalty: on top of the per-violation penalty above,
			// subtract an amount proportional to how many HOURS the day's gaps run
			// beyond the allowed maximum, so a very long gap is penalised much more
			// than a slightly-too-long one. This steers the GA toward compact days
			// instead of schedules with large idle gaps between subjects.
			excess_gap_slots := TotalExcessGapSlots(week_sched[day], gapConfig.MaxGapSlots)
			if excess_gap_slots > 0 {
				week_sched_fitness -= gapConfig.Penalty *
					(float64(excess_gap_slots) / float64(Const.N_HOUR_TIME_SLOTS))
			}
		}
	}

	if days_with_class == 0.0 {
		return -24.0
	}

	week_sched_fitness = week_sched_fitness / days_with_class

	// total class days in one week above 4 days are punished, and rewarded if not (panel revision recommendation)
	if days_with_class > 4 {
		week_sched_fitness -= 2
	} else {
		week_sched_fitness += 2.5
	}

	return week_sched_fitness
}

// A basic fitness function, if `department_to_measure` is nil, it measures the whole university schedule
func MeasureUniSchedBasicFitness(complete_uni_sched Schedule.UniTimeTables, all_curriculums []Curriculum.Curriculum, department_to_measure map[uint16]bool, selected_semester int) float64 {
	if complete_uni_sched.IsEmpty() {
		return -24.0
	}

	subject_id_to_async_hours := buildSubjectIDToAsyncHoursMap(all_curriculums)

	accumulated_fitness := 0.0
	total_fitness_measurements := 0

	IterateSectionsWeekSchedule(complete_uni_sched, all_curriculums, selected_semester, nil, nil, func(indicies IterIndices, values IterValues) IterReturnType {

		if len(department_to_measure) > 0 {
			if !department_to_measure[values.Curriculum.DepartmentID] {
				return IterProceed
			}
		}

		if values.WeekSched != nil {
			total_fitness_measurements++
			accumulated_fitness += MeasureWeekTimeTableBasicFitness(*values.WeekSched, subject_id_to_async_hours)
		}

		return IterProceed
	})

	if total_fitness_measurements == 0 {
		return -24.0
	}

	return accumulated_fitness / float64(total_fitness_measurements)
}

func MeasureFitnessPrefHeatMapComparison(uni_sched Schedule.DayTimeTable) float64 {
	// TODO: implement preference heat map comparison based fitness function

	return 1.0
}
