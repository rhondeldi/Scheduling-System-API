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

func MeasureWeekTimeTableBasicFitness(week_sched Schedule.WeekTimeTable) float64 {
	week_sched_fitness := 0.0
	days_with_class := 0.0

	for day := range Const.N_WEEKLY_SCHOOL_DAYS {

		has_class_after_5pm := false
		has_time_for_lunch := false
		day_total_hours := 0.0

		for time_slot := range Const.N_DAILY_TIME_SLOTS {
			if week_sched[day][time_slot].GetSubjectID() > 0 {
				day_total_hours += (1.0 / float64(Const.N_HOUR_TIME_SLOTS))
			}

			if time_slot >= 20 {
				has_class_after_5pm = true
			}

			if (time_slot >= 8) && (time_slot <= 11) && (week_sched[day][time_slot].GetSubjectID() == 0) {
				has_time_for_lunch = true
			}
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

		// class hours beyond the prefered are punished, below are rewarded
		if day_total_hours > PREFERRED_MAX_CLASS_HOUR_PER_DAY {
			week_sched_fitness -= 3.5
		} else {
			week_sched_fitness += 3.5
		}

		// long class hours during saturday are punished, short hours are rewarded
		if (day_total_hours > (PREFERRED_MAX_CLASS_HOUR_PER_DAY / 2)) && (day == (Const.N_WEEKLY_SCHOOL_DAYS - 1)) {
			week_sched_fitness -= 1.0
		} else {
			week_sched_fitness += 1.0
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

	accumulated_fitness := 0.0
	total_fitness_measurements := 0

	IterateSectionsWeekSchedule(complete_uni_sched, all_curriculums, selected_semester, nil, nil, func(indicies IterIndices, values IterValues) IterReturnType {

		if len(department_to_measure) > 0 {
			if !department_to_measure[values.Curriculum.DepartmentID] {
				return IterProceed
			}
		}

		total_fitness_measurements++
		accumulated_fitness += MeasureWeekTimeTableBasicFitness(*values.WeekSched)

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
