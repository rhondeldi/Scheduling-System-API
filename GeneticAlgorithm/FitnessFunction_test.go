package GeneticAlgorithm_test

import (
	"testing"

	"github.com/mrdcvlsc/scheduling-system-backend/GeneticAlgorithm"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"github.com/mrdcvlsc/scheduling-system-backend/StorageResources"
)

func TestEmptyScheduleFitness(t *testing.T) {
	persistence := StorageResources.Persistence{ReaderService: &StorageResources.JsonReader{}}

	curriculums, err_all_curriculums := persistence.ReaderService.ReadAllCurriculum()

	if err_all_curriculums != nil {
		t.Fatal(err_all_curriculums)
	}

	uni_sched := GeneticAlgorithm.NewEmptyIndividual(curriculums, GeneticAlgorithm.TERM_1ST_SEMESTER)

	fitness := GeneticAlgorithm.MeasureUniSchedBasicFitness(uni_sched, curriculums, nil, GeneticAlgorithm.TERM_1ST_SEMESTER)
	t.Logf("empty schedule fitness : %f", fitness)

	for range 50 {
		uni_sched = append(uni_sched, Schedule.WeekTimeTable{})
	}

	fitness_of_empty_schedules := GeneticAlgorithm.MeasureUniSchedBasicFitness(uni_sched, curriculums, nil, GeneticAlgorithm.TERM_1ST_SEMESTER)

	empty_fitness := -24.0

	if fitness_of_empty_schedules != empty_fitness {
		t.Fatalf("fitness of empty schedules should be equal %f, but got %f", empty_fitness, fitness_of_empty_schedules)
	}
}

func TestGeneratedScheduleFitness(t *testing.T) {
	persistence := StorageResources.Persistence{ReaderService: &StorageResources.JsonReader{}}
	target_semester := 0

	////////////////////////////////////////////////////////////////////////////////////////

	departments, err_all_departments := persistence.ReaderService.ReadAllDepartments()

	if err_all_departments != nil {
		t.Fatal(err_all_departments)
	}

	rooms, err_all_rooms := persistence.ReaderService.ReadAllRooms()

	if err_all_rooms != nil {
		t.Fatal(err_all_rooms)
	}

	curriculums, err_all_curriculums := persistence.ReaderService.ReadAllCurriculum()

	if err_all_curriculums != nil {
		t.Fatal(err_all_curriculums)
	}

	dept_id_to_department := GeneticAlgorithm.GenerateMapDeptIdToDepartment(departments)

	////////////////////////////////////////////////////////////////////////////////////////

	default_encoding_resource, err_read_default_encoding_resource := GeneticAlgorithm.ReadDefaultEncodingResource(&persistence)

	if err_read_default_encoding_resource != nil {
		t.Fatal(err_read_default_encoding_resource)
	}

	////////////////////////////////////////////////////////////////////////////////////////

	var new_university_schedule *Schedule.UniTimeTables

	for i := 0; i < 50; i++ {
		empty_university_schedule := GeneticAlgorithm.NewEmptyIndividual(curriculums, target_semester)

		university_schedules, encoding_resource, err := GeneticAlgorithm.EncodeIndividualGenome(
			empty_university_schedule,
			curriculums,
			dept_id_to_department, default_encoding_resource, nil,
			target_semester, 0,
		)

		if len(university_schedules) == 0 {
			t.Fatal("No university schedules generated")
		}

		if university_schedules == nil {
			t.Fatalf("returned a nil university schedule : loop iteration %d\n", i)
		}

		if university_schedules.IsEmpty() {
			t.Fatalf("returned an empty university schedule : loop iteration %d\n", i)
		}

		err_vertical_validations := university_schedules.VerticalValidation(rooms)

		for _, e := range err_vertical_validations {
			t.Error(e)
		}

		if err == nil {
			err_horizontal_validations := GeneticAlgorithm.HorizontalValidation(
				university_schedules,
				curriculums, nil, target_semester,
			)

			for _, e := range err_horizontal_validations {
				t.Fatal(e)
			}
		}

		if encoding_resource != nil && err == nil {
			new_university_schedule = &university_schedules
			break
		}
	}

	fitness := GeneticAlgorithm.MeasureUniSchedBasicFitness(*new_university_schedule, curriculums, nil, target_semester)

	if fitness >= -12.0 && fitness <= 12.0 {
		t.Logf("generated schedule fitness : %f", fitness)
	} else {
		t.Fatalf("generated schedule fitness : %f", fitness)
	}
}
