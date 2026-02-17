package GeneticAlgorithm

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"time"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Departments"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"github.com/mrdcvlsc/scheduling-system-backend/StorageResources"
)

const MAX_BASE_SCHEDULE_REPAIR_TRAIALS int = 512
const MAX_GENESIS_INDIVIDUAL_GENERATION_TRIALS int = 512
const MAX_CROSSOVER_TRIALS int = 384
const MAX_RE_ENCODE_REPAIR_TRIALS int = 384

type SchedAndResources struct {
	UniSched  Schedule.UniTimeTables
	Resources *EncodingResource
}

func RunGeneticAlgorithm(
	base_uni_sched Schedule.UniTimeTables,
	curriculums []Curriculum.Curriculum,
	rooms []Rooms.Room,
	dept_id_to_department map[uint16]Departments.Department,
	default_empty_encoding_resource, base_encoding_resource *EncodingResource,
	department_to_encode map[uint16]bool,
	selected_semester, population_size, generations int,
	resource_persistence *StorageResources.Persistence,
	cb_fn_generation func(generation int, generation_fittest_sched Schedule.UniTimeTables, fitness float64),
) (Schedule.UniTimeTables, *EncodingResource, error) {

	rng := rand.New(rand.NewSource(time.Now().UnixMilli()))

	department_id := uint16(0)

	if len(department_to_encode) != 1 {
		panic("department to encode should only have one value for now")
	}

	for k := range department_to_encode {
		department_id = k
	}

	default_instructor_id_to_instructor, err_generate_instructor_id_to_instructor := GenerateMapInstructorIdToInstructor(resource_persistence)

	if err_generate_instructor_id_to_instructor != nil {
		return nil, nil, fmt.Errorf("unable to generate instructor id to instructor map, cause by error: %s", err_generate_instructor_id_to_instructor.Error())
	}

	log.Printf("||||||||||||||||||||||||||||||| Performing GA with %s ||||||||||||||||||||||||||||||| ", dept_id_to_department[department_id].Name)

	////////////////////////////////////////////////////////////////////////////////////////
	//             PUT THE BASE SCHEDULE AT THE TOP OF GENESIS POPULATION
	////////////////////////////////////////////////////////////////////////////////////////

	genesis_population := make([]SchedAndResources, 0)
	base_schedule_repair_tries := 0

	for {
		new_base_sched, new_base_sched_resource, err_encoding_new_base_sched := EncodeIndividualGenome(
			base_uni_sched,
			curriculums,
			dept_id_to_department,
			base_encoding_resource,
			department_to_encode,
			selected_semester, 0,
		)

		base_schedule_repair_tries++

		if err_encoding_new_base_sched != nil {

			if base_schedule_repair_tries < MAX_BASE_SCHEDULE_REPAIR_TRAIALS {
				continue
			}

			fmt.Printf(
				"RunGeneticAlgorithm [base-schedule-repair-error], caused by: %s",
				err_encoding_new_base_sched.Error(),
			)

			return new_base_sched, new_base_sched_resource, fmt.Errorf(
				"base schedule repair error, caused by: %s",
				err_encoding_new_base_sched.Error(),
			)
		}

		log.Printf("university schedule fitness : %f", MeasureUniSchedBasicFitness(base_uni_sched, curriculums, nil, selected_semester))
		log.Printf("department schedule fitness : %f", MeasureUniSchedBasicFitness(base_uni_sched, curriculums, department_to_encode, selected_semester))

		genesis_population = append(genesis_population, SchedAndResources{
			UniSched:  new_base_sched,
			Resources: new_base_sched_resource,
		})

		break
	}

	////////////////////////////////////////////////////////////////////////////////////////
	//               POPULATE THE GENESIS POPULATION WITH RANDOM INDIVIDUALS
	////////////////////////////////////////////////////////////////////////////////////////

	start := time.Now()

	genesis_generation_tries := 0

	log.Print("generating genesis population...")

	for len(genesis_population) < population_size {

		copy_uni_sched := make(Schedule.UniTimeTables, len(base_uni_sched))

		copied_week_time_table := copy(copy_uni_sched, base_uni_sched)

		if copied_week_time_table != len(base_uni_sched) {
			log.Printf("RunGeneticAlgorithm [copy-uni-sched-error]: slice elements copied %d, internal university schedule copy operation failed in generate new individual function", copied_week_time_table)

			return nil, nil, fmt.Errorf("genetic algorithm run error: slice elements copied %d, internal university schedule copy operation failed in generate new individual function", copied_week_time_table)
		}

		ClearDepartmentSchedule(copy_uni_sched, curriculums, department_id, selected_semester)

		copy_encoding_resource, err_gen_copy_encoding_resource := GenerateEncodingResourceFromUniTimeTable(copy_uni_sched, curriculums, selected_semester, default_empty_encoding_resource)

		if err_gen_copy_encoding_resource != nil {
			log.Printf("RunGeneticAlgorithm [copy-encoding-resource-error]: unable to generate encoding resource from individual during genesis generation")

			return nil, nil, fmt.Errorf("genetic algorithm run error: unable to generate encoding resource from individual during genesis generation")
		}

		initial_sched, initial_encoding_resource, err_encode_initial := EncodeIndividualGenome(
			copy_uni_sched,
			curriculums,
			dept_id_to_department,
			copy_encoding_resource,
			department_to_encode,
			selected_semester, 0,
		)

		if err_encode_initial != nil {
			genesis_generation_tries++

			if genesis_generation_tries >= MAX_GENESIS_INDIVIDUAL_GENERATION_TRIALS {
				if initial_sched == nil {
					log.Printf(
						"RunGeneticAlgorithm [Genesis Population]: unable to generate new a individual for genesis generation after %d tries, cause by error: %s",
						MAX_GENESIS_INDIVIDUAL_GENERATION_TRIALS, err_encode_initial.Error(),
					)

					return nil, nil, fmt.Errorf(
						"unable to generate new a individual for genesis generation after %d tries, cause by error: %s",
						MAX_GENESIS_INDIVIDUAL_GENERATION_TRIALS, err_encode_initial.Error(),
					)
				} else {
					return initial_sched, nil, fmt.Errorf(
						"unable to generate new a individual for genesis generation after %d tries, cause by error: %s",
						MAX_GENESIS_INDIVIDUAL_GENERATION_TRIALS, err_encode_initial.Error(),
					)
				}
			}

			continue
		} else {
			genesis_generation_tries = 0
		}

		genesis_population = append(genesis_population, SchedAndResources{
			UniSched:  initial_sched,
			Resources: initial_encoding_resource,
		})
	}

	fmt.Printf("ga: [generate genesis population] - took %s\n", time.Since(start))

	////////////////////////////////////////////////////////////////////////////////////////
	//                   START GENETIC ALGORITHM SCHEDULE GENERATION
	////////////////////////////////////////////////////////////////////////////////////////

	// NOTE: "base schedule" are the current initial or most fit individual in a generation
	// this should never be replaced or modify by the algorithm, the "base schedule"
	// will only change if a new fitter inidividual emerges as a new "base schedule".

	for g := range generations {

		////////////////////////////////////////////////////////////////////////////////////////
		//                     CREATE POPULATION FOR A NEW GENERATION
		////////////////////////////////////////////////////////////////////////////////////////

		log.Printf("running genetic algorithm generation %d", g)

		population := make([]SchedAndResources, 0, population_size+1)

		////////////////////////////////////////////////////////////////////////////////////////
		//				                TOURNAMENT SELECTION
		////////////////////////////////////////////////////////////////////////////////////////

		log.Printf("apply tournament selection to the population")

		start = time.Now()

		rng.Shuffle(len(genesis_population[1:]), func(i, j int) {
			genesis_population[1:][i], genesis_population[1:][j] = genesis_population[1:][j], genesis_population[1:][i]
		})

		for i := 0; i < len(genesis_population); i += 2 {
			A := MeasureUniSchedBasicFitness(genesis_population[i].UniSched, curriculums, department_to_encode, selected_semester)
			B := MeasureUniSchedBasicFitness(genesis_population[i+1].UniSched, curriculums, department_to_encode, selected_semester)

			if A > B {
				population = append(population, genesis_population[i])
			} else {
				population = append(population, genesis_population[i+1])
			}
		}

		if len(genesis_population)%2 == 1 {
			population = append(population, genesis_population[len(genesis_population)-1])
		}

		remaining_missing_population := population_size - len(population)

		fmt.Printf(
			"ga: [tournament selection] - took %s, remaining missing population after tournament selection %d\n",
			time.Since(start), remaining_missing_population,
		)

		////////////////////////////////////////////////////////////////////////////////////////
		//				                     CROSSOVER
		////////////////////////////////////////////////////////////////////////////////////////

		start = time.Now()

		crossover_tries := 0

		population_size_before_crossover := len(population)

		if population_size_before_crossover == 0 {
			panic("detected population size of 0 after tournament selection, that should not happen")
		}

		for len(population) < population_size {

			parent1_idx := rng.Intn(population_size_before_crossover)
			parent2_idx := rng.Intn(population_size_before_crossover)

			if parent1_idx == parent2_idx {
				continue
			}

			parent1 := population[parent1_idx]
			parent2 := population[parent2_idx]

			offspring, err_crossover := Crossover(
				parent1.UniSched, parent2.UniSched, default_empty_encoding_resource,
				curriculums, rooms, selected_semester,
				dept_id_to_department, department_to_encode,
				default_instructor_id_to_instructor,
				resource_persistence,
			)

			if err_crossover != nil {
				crossover_tries++

				if crossover_tries >= MAX_CROSSOVER_TRIALS {
					log.Printf(
						"RunGeneticAlgorithm [Crossover]: unable to produce offspring at generation %d after %d tries during crossovers, cause by error : %s",
						g, MAX_CROSSOVER_TRIALS, err_crossover.Error(),
					)

					return nil, nil, fmt.Errorf(
						"unable to produce offspring at generation %d after %d tries during crossovers, cause by error : %s",
						g, MAX_CROSSOVER_TRIALS, err_crossover.Error(),
					)
				}

				continue
			} else {
				crossover_tries = 0
			}

			population = append(population, *offspring)
		}

		remaining_missing_population = population_size - len(population)

		fmt.Printf(
			"ga: [crossover] - took %s, remaining missing population after crossover %d\n",
			time.Since(start), remaining_missing_population,
		)

		////////////////////////////////////////////////////////////////////////////////////////
		//				                   RANDOM MUTATIONS
		////////////////////////////////////////////////////////////////////////////////////////

		start = time.Now()

		for i := 1; i < len(population); i++ {

			prev_encoding_resource, err_copy_enc_re := population[i].Resources.MakeCopy()

			if err_copy_enc_re != nil {
				log.Panic("error copying encoding resource in random mutation : ", err_copy_enc_re)
			}

			err_v_val := population[i].UniSched.VerticalValidation(rooms)

			if len(err_v_val) > 0 {
				log.Panicf("OPPSv1!  THERE IS SOMETHING WRONG (VARTICAL VALIDATION) - RANDOM MUTATION INDEX [%d]", i)
			}

			err_h_val := HorizontalValidation(population[i].UniSched, curriculums, department_to_encode, selected_semester)

			if len(err_h_val) > 0 {
				log.Panicf("OPPSv1!  THERE IS SOMETHING WRONG (HORIZONTAL VALIDATION) - RANDOM MUTATION INDEX [%d]", i)
			}

			// apply random mutations to some of the CURRENT individuals in the population

			ApplyRandomDaySwapTimeSlots(population[i].UniSched, population[i].Resources, curriculums, department_id, selected_semester)
			ApplyRandomSubjectDaySwap(population[i].UniSched, population[i].Resources, curriculums, department_id, selected_semester)
			ApplyRandomSubjectTimeSlotNudge(population[i].UniSched, population[i].Resources, curriculums, department_id, selected_semester)
			ApplyRandomSubjectTimeSlotAndDayNudge(population[i].UniSched, population[i].Resources, curriculums, department_id, selected_semester)

			err_v2_val := population[i].UniSched.VerticalValidation(rooms)

			if len(err_v2_val) > 0 {
				log.Panicf("OPPSv2!  THERE IS SOMETHING WRONG (VARTICAL VALIDATION) - RANDOM MUTATION INDEX [%d]\n\n%v", i, err_v2_val)
			}

			err_h2_val := HorizontalValidation(population[i].UniSched, curriculums, department_to_encode, selected_semester)

			if len(err_h2_val) > 0 {
				log.Panicf("OPPSv2!  THERE IS SOMETHING WRONG (HORIZONTAL VALIDATION) - RANDOM MUTATION INDEX [%d]\n\n%v", i, err_h2_val)
			}

			// repair broken genome after mutations

			re_encode_tries := 0

			for re_encode_tries < MAX_RE_ENCODE_REPAIR_TRIALS {

				generated_encoding_resource, err_generate_encoding_resource := GenerateEncodingResourceFromUniTimeTable(
					population[i].UniSched, curriculums, selected_semester, default_empty_encoding_resource,
				)

				if err_generate_encoding_resource != nil {
					log.Printf(
						"RunGeneticAlgorithm [Random Mutation][%d]: unable to generate encoding resource needed to repair a mutated individual on generation %d, caused by %s",
						i, g, err_generate_encoding_resource.Error(),
					)

					return nil, nil, fmt.Errorf(
						"unable to generate encoding resource needed to repair a mutated individual on generation %d, caused by %s",
						g, err_generate_encoding_resource.Error(),
					)
				}

				if !IsEqualEncodingResource(generated_encoding_resource, population[i].Resources) {

					if IsEqualEncodingResource(prev_encoding_resource, population[i].Resources) {
						log.Printf(
							"RunGeneticAlgorithm: [Random Mutation][%d] PREVIOUS encoding resource after crossover should NOT be equal to the population encoding resource, why is this one equal? ERROR DETECTED!",
							i,
						)
					}

					// TODO: if tested many times, and there is no instance of this panic, then directly use
					log.Panicf(
						"RunGeneticAlgorithm: [Random Mutation][%d] encoding resource after crossover should be equal to the generated encoding resource, why is this one not? ERROR DETECTED!",
						i,
					)
				}

				repaired_uni_sched, repaired_encoding_resource, err_repair_schedule := EncodeIndividualGenome(
					population[i].UniSched, curriculums, dept_id_to_department,
					generated_encoding_resource, department_to_encode,
					selected_semester, 0,
				)

				if err_repair_schedule != nil {
					re_encode_tries++

					if re_encode_tries >= MAX_RE_ENCODE_REPAIR_TRIALS {
						log.Printf(
							"RunGeneticAlgorithm [Random Mutation]: unable to repair an individual on generation %d after %d tries : caused by error %s",
							g, MAX_RE_ENCODE_REPAIR_TRIALS, err_repair_schedule.Error(),
						)

						return nil, nil, fmt.Errorf(
							"unable to repair an individual on generation %d after %d tries : caused by error %s",
							g, MAX_RE_ENCODE_REPAIR_TRIALS, err_repair_schedule.Error(),
						)
					} else {
						ApplyRandomDaySwapTimeSlots(population[i].UniSched, population[i].Resources, curriculums, department_id, selected_semester)
						ApplyRandomSubjectDaySwap(population[i].UniSched, population[i].Resources, curriculums, department_id, selected_semester)
						ApplyRandomSubjectTimeSlotNudge(population[i].UniSched, population[i].Resources, curriculums, department_id, selected_semester)
						ApplyRandomSubjectTimeSlotAndDayNudge(population[i].UniSched, population[i].Resources, curriculums, department_id, selected_semester)
					}

					continue
				} else {
					re_encode_tries = 0
				}

				// apply repairs to the individual

				population[i].UniSched = repaired_uni_sched
				population[i].Resources = repaired_encoding_resource

				break
			}
		}

		fmt.Printf("ga: [random mutation] - took %s\n", time.Since(start))

		////////////////////////////////////////////////////////////////////////////////////////
		//				PREPARE PREPARE FINAL POPULATION FOR THE NEXT GENERATION
		////////////////////////////////////////////////////////////////////////////////////////

		start = time.Now()

		sort.Slice(population, func(i, j int) bool {
			fitness_a := MeasureUniSchedBasicFitness(population[i].UniSched, curriculums, department_to_encode, selected_semester)
			fitness_b := MeasureUniSchedBasicFitness(population[j].UniSched, curriculums, department_to_encode, selected_semester)
			return fitness_a > fitness_b
		})

		// add back to the genesis population

		genesis_population = population

		fittest_individual_fitness := MeasureUniSchedBasicFitness(genesis_population[0].UniSched, curriculums, nil, selected_semester)

		fmt.Printf(
			"ga: [population to transfer to next generation] - took %s, best individual fitness : %f\n",
			time.Since(start), fittest_individual_fitness,
		)

		if cb_fn_generation != nil {
			cb_fn_generation(g, genesis_population[0].UniSched, fittest_individual_fitness)
		}

		estimated_bytes_of_encoding_resource := float64(genesis_population[0].Resources.EstimateMemoryUsageInBytes())

		log.Printf(
			"Estimated Memory Usage Of Encoding Resource [generation %d]: %d bytes, %.2fKB, %.2fMB, %.2fGB",
			g, uint64(estimated_bytes_of_encoding_resource),
			estimated_bytes_of_encoding_resource/1000.0,
			(estimated_bytes_of_encoding_resource/1000.0)/1000.0,
			((estimated_bytes_of_encoding_resource/1000.0)/1000.0)/1000.0,
		)

		if os.Getenv("LOG_MODE") == "verbose" {
			for i, uni_gen_sched := range genesis_population {
				fmt.Printf("generation %d, individual %d -> fitness : %f\n", g, i+1, MeasureUniSchedBasicFitness(
					uni_gen_sched.UniSched, curriculums, department_to_encode, selected_semester,
				))
			}
		}
	}

	////////////////////////////////////////////////////////////////////////////////////////
	//            GENETIC ALGORITHM END : PICK THE BEST INDIVIDUAL SOLUTION
	////////////////////////////////////////////////////////////////////////////////////////

	log.Printf("ga: fittest individual fitness : %f", MeasureUniSchedBasicFitness(genesis_population[0].UniSched, curriculums, department_to_encode, selected_semester))

	if genesis_population[0].UniSched.IsEmpty() {
		log.Printf("GA-ERROR: fittest university schedule is empty")
		return nil, nil, errors.New("fittest university schedule is empty")
	}

	if errs := genesis_population[0].UniSched.VerticalValidation(rooms); len(errs) > 0 {
		log.Printf("GA-ERROR: fittest university schedule have vertical overlaps:\n\n%v\n\n", errs)
		return nil, nil, errors.New("fittest university schedule have vertical overlaps")
	} else {
		for k, v := range department_to_encode {
			if v {
				log.Printf(
					"ga: [passed] final vertical validation for %s %s fittest university schedule",
					dept_id_to_department[k].Code, Curriculum.SEMESTER_INDEX_NAME[selected_semester],
				)
			}
		}
	}

	errs_horizontal_validation := HorizontalValidation(genesis_population[0].UniSched, curriculums, department_to_encode, selected_semester)

	if len(errs_horizontal_validation) > 0 {
		log.Printf("GA-ERROR: fittest university schedule is empty")
		return nil, nil, errors.New("fittest university schedule is empty")
	} else {
		for k, v := range department_to_encode {
			if v {
				log.Printf(
					"ga: [passed] final horizontal validation for %s %s fittest university schedule",
					dept_id_to_department[k].Code, Curriculum.SEMESTER_INDEX_NAME[selected_semester],
				)
			}
		}
	}

	return genesis_population[0].UniSched, genesis_population[0].Resources, nil
}
