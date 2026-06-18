package GeneticAlgorithm

import (
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Departments"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
	"github.com/mrdcvlsc/scheduling-system-backend/StorageResources"
)

// ── constants ─────────────────────────────────────────────────────────────────

const MAX_BASE_SCHEDULE_REPAIR_TRAIALS int = 512
const MAX_GENESIS_INDIVIDUAL_GENERATION_TRIALS int = 512
const MAX_CROSSOVER_TRIALS int = 384
const MAX_RE_ENCODE_REPAIR_TRIALS int = 384
const ANN_TOP_K_RATIO float64 = 0.20
const ANN_TOP_K_MIN int = 8
const MAX_FITNESS_CACHE_SIZE int = 50000
const ELITE_PRESERVE_COUNT int = 2

// Default thresholds — override via environment variables.
const ANN_MUTATION_REVERT_CONFIDENCE_DEFAULT float64 = 0.75
const ANN_CONSTRAINT_PENALTY_DEFAULT float64 = 0.5

// ── types ─────────────────────────────────────────────────────────────────────

type SchedAndResources struct {
	UniSched  Schedule.UniTimeTables
	Resources *EncodingResource
}

// ANNFitnessEvalStats collects per-generation ANN telemetry for the log line.
type ANNFitnessEvalStats struct {
	// Fitness ranking
	ANNPredictionCount int
	ANNRequestCount    int
	ANNFailureCount    int
	ANNFallbackCount   int
	ANNUsedEvalCount   int
	FallbackEvalCount  int
	ClassicEvalCount   int
	CacheHitCount      int
	CacheMissCount     int
	// Crossover (step 4)
	CrossoverRequestCount int
	CrossoverFailureCount int
	// Constraint pre-screening (step 5)
	ConstraintRequestCount int
	ConstraintFailureCount int
	ConstraintPenaltyTotal float64
	// Mutation filtering (step 6)
	MutationRequestCount int
	MutationFailureCount int
	MutationRevertCount  int
}

type finalScheduleQualitySummary struct {
	ExcellentSections int
	GoodSections      int
	FairSections      int
	PoorSections      int
	TotalViolations   int
	OverallQuality    float64
	TotalSections     int
	Sections          []finalScheduleQualitySectionSummary
}

type finalScheduleQualitySectionSummary struct {
	SectionLabel string
	Bucket       string
	Violations   []string
}

func countConstraintLabelViolations(labels ConstraintLabels) int {
	violations := 0
	if labels.InstructorConflict {
		violations++
	}
	if labels.RoomConflict {
		violations++
	}
	if labels.NoLunchBreak {
		violations++
	}
	if labels.LateClasses {
		violations++
	}
	if labels.ExcessiveHours {
		violations++
	}
	if labels.SaturdayOverload {
		violations++
	}
	if labels.ResourceUnavailable {
		violations++
	}
	if labels.CurriculumConflict {
		violations++
	}
	if labels.RoomCapacity {
		violations++
	}
	if labels.InstructorAvailability {
		violations++
	}
	return violations
}

func constraintLabelNames(labels ConstraintLabels) []string {
	violations := make([]string, 0, 10)
	if labels.InstructorConflict {
		violations = append(violations, "InstructorConflict")
	}
	if labels.RoomConflict {
		violations = append(violations, "RoomConflict")
	}
	if labels.NoLunchBreak {
		violations = append(violations, "NoLunchBreak")
	}
	if labels.LateClasses {
		violations = append(violations, "LateClasses")
	}
	if labels.ExcessiveHours {
		violations = append(violations, "ExcessiveHours")
	}
	if labels.SaturdayOverload {
		violations = append(violations, "SaturdayOverload")
	}
	if labels.ResourceUnavailable {
		violations = append(violations, "ResourceUnavailable")
	}
	if labels.CurriculumConflict {
		violations = append(violations, "CurriculumConflict")
	}
	if labels.RoomCapacity {
		violations = append(violations, "RoomCapacity")
	}
	if labels.InstructorAvailability {
		violations = append(violations, "InstructorAvailability")
	}
	return violations
}

func qualityBucketName(violationCount int) string {
	switch {
	case violationCount == 0:
		return "Excellent"
	case violationCount == 1:
		return "Good"
	case violationCount == 2:
		return "Fair"
	default:
		return "Poor"
	}
}

func sectionDisplayName(values IterValues, indicies IterIndices) string {
	curriculumCode := "unknown-curriculum"
	if values.Curriculum != nil {
		curriculumCode = values.Curriculum.CurriculumCode
	}
	yearLevelName := "unknown-year"
	if values.YearLevel != nil {
		yearLevelName = values.YearLevel.Name
	}
	semesterName := "unknown-semester"
	if values.Semester != nil {
		semesterName = values.Semester.Name
	}
	sectionName := fmt.Sprintf("#%d", indicies.Section)
	if indicies.Section >= 0 && indicies.Section < len(Curriculum.SECTION) {
		sectionName = Curriculum.SECTION[indicies.Section]
	}
	return fmt.Sprintf("%s %s %s section %s", curriculumCode, yearLevelName, semesterName, sectionName)
}

func sectionQualityScore(violationCount int) float64 {
	switch {
	case violationCount == 0:
		return 100.0
	case violationCount == 1:
		return 75.0
	case violationCount == 2:
		return 50.0
	default:
		return 25.0
	}
}

func percentage(part, total int) float64 {
	if total <= 0 {
		return 0
	}
	return (float64(part) / float64(total)) * 100.0
}

func measureFinalScheduleQuality(
	sched Schedule.UniTimeTables,
	curriculums []Curriculum.Curriculum,
	rooms []Rooms.Room,
	departmentToMeasure map[uint16]bool,
	selectedSemester int,
) finalScheduleQualitySummary {
	summary := finalScheduleQualitySummary{}
	hardViolationCount := countConstraintLabelViolations(
		detectHardViolations(sched, curriculums, rooms, departmentToMeasure, selectedSemester),
	)
	qualityTotal := 0.0

	IterateSectionsWeekSchedule(sched, curriculums, selectedSemester, nil, nil,
		func(indicies IterIndices, values IterValues) IterReturnType {
			_ = indicies
			if values.Curriculum == nil || values.WeekSched == nil {
				return IterProceed
			}
			if len(departmentToMeasure) > 0 && !departmentToMeasure[values.Curriculum.DepartmentID] {
				return IterProceed
			}

			softViolations := detectSoftViolationsForSection(*values.WeekSched)
			violationCount := countConstraintLabelViolations(softViolations)
			bucketName := qualityBucketName(violationCount)

			summary.TotalSections++
			summary.TotalViolations += violationCount
			qualityTotal += sectionQualityScore(violationCount)
			if violationCount > 0 {
				summary.Sections = append(summary.Sections, finalScheduleQualitySectionSummary{
					SectionLabel: sectionDisplayName(values, indicies),
					Bucket:       bucketName,
					Violations:   constraintLabelNames(softViolations),
				})
			}

			switch {
			case violationCount == 0:
				summary.ExcellentSections++
			case violationCount == 1:
				summary.GoodSections++
			case violationCount == 2:
				summary.FairSections++
			default:
				summary.PoorSections++
			}

			return IterProceed
		},
	)
	summary.TotalViolations += hardViolationCount

	if summary.TotalSections > 0 {
		summary.OverallQuality = qualityTotal / float64(summary.TotalSections)
		if hardViolationCount > 0 {
			summary.OverallQuality -= float64(hardViolationCount) * 5.0
			if summary.OverallQuality < 0 {
				summary.OverallQuality = 0
			}
		}
	}

	return summary
}

// ── payload helpers ───────────────────────────────────────────────────────────

func weekTimeTableToANNPayload(week *Schedule.WeekTimeTable) [][][]int {
	payload := make([][][]int, 6)
	for day := 0; day < 6; day++ {
		payload[day] = make([][]int, 24)
		for slot := 0; slot < 24; slot++ {
			ts := week[day][slot]
			payload[day][slot] = []int{
				int(ts.GetSubjectID()),
				int(ts.GetInstructorID()),
				int(ts.GetRoomID()),
			}
		}
	}
	return payload
}

func collectSectionPayloads(
	completeUniSched Schedule.UniTimeTables,
	allCurriculums []Curriculum.Curriculum,
	departmentToMeasure map[uint16]bool,
	selectedSemester int,
) [][][][]int {
	payloads := make([][][][]int, 0)
	IterateSectionsWeekSchedule(completeUniSched, allCurriculums, selectedSemester, nil, nil,
		func(indicies IterIndices, values IterValues) IterReturnType {
			_ = indicies
			if len(departmentToMeasure) > 0 && !departmentToMeasure[values.Curriculum.DepartmentID] {
				return IterProceed
			}
			if values.WeekSched == nil {
				return IterProceed
			}
			payloads = append(payloads, weekTimeTableToANNPayload(values.WeekSched))
			return IterProceed
		},
	)
	return payloads
}

func firstSectionScheduleData(
	completeUniSched Schedule.UniTimeTables,
	allCurriculums []Curriculum.Curriculum,
	departmentToMeasure map[uint16]bool,
	selectedSemester int,
) ([][][]int, bool) {
	payloads := collectSectionPayloads(completeUniSched, allCurriculums, departmentToMeasure, selectedSemester)
	if len(payloads) == 0 {
		return nil, false
	}
	return payloads[0], true
}

// ── external-section detection ────────────────────────────────────────────────
//
// identifyExternalSections finds sections in `department_to_encode` that are
// PARTIALLY scheduled in the base university schedule — some subjects placed
// from a previous run, others missing.  These belong to OTHER programs in the
// same department (e.g. BSPsych within department 2 when the GA is generating
// DAS).  EncodeIndividualGenome cannot fully fill them once the active program
// has consumed the relevant instructors and rooms, so leaving them eligible
// for re-encoding produces partially-filled sections that fail
// HorizontalValidation on every single crossover attempt.
//
// The fix: return a usi → {subject_id → true} map of every curriculum subject
// in those sections.  Callers add this to their EncodingResource's
// IsSchedIdxToSubIdToSkip *before* invoking EncodeIndividualGenome, which
// then leaves those sections empty.  HV's empty-section guard then skips
// them cleanly during validation.
//
// Heuristic: a section is "external" iff `0 < distinct_placed < expected`.
//   - distinct_placed == 0 → empty: assumed to be a target section that needs
//     generating (do NOT mark it).
//   - distinct_placed == expected → fully scheduled: the GA can re-encode it
//     without losing data (do NOT mark it).
//   - otherwise → partial from another program: mark and skip.
type crossoverPairCandidate struct {
	parent1Idx int
	parent2Idx int
	confidence float64
	compatible bool
}

func buildParentPairBatch(
	population []SchedAndResources,
	allCurriculums []Curriculum.Curriculum,
	departmentToMeasure map[uint16]bool,
	selectedSemester int,
	maxPairs int,
	rng *rand.Rand,
) ([]crossoverPairCandidate, [][2][][][]int) {
	type parentSection struct {
		idx     int
		payload [][][]int
	}

	sections := make([]parentSection, 0, len(population))
	for i := range population {
		payload, ok := firstSectionScheduleData(
			population[i].UniSched, allCurriculums, departmentToMeasure, selectedSemester,
		)
		if ok {
			sections = append(sections, parentSection{idx: i, payload: payload})
		}
	}

	candidates := make([]crossoverPairCandidate, 0)
	pairs := make([][2][][][]int, 0)
	for i := 0; i < len(sections); i++ {
		for j := i + 1; j < len(sections); j++ {
			candidates = append(candidates, crossoverPairCandidate{
				parent1Idx: sections[i].idx,
				parent2Idx: sections[j].idx,
			})
			pairs = append(pairs, [2][][][]int{sections[i].payload, sections[j].payload})
		}
	}

	if rng != nil {
		rng.Shuffle(len(candidates), func(i, j int) {
			candidates[i], candidates[j] = candidates[j], candidates[i]
			pairs[i], pairs[j] = pairs[j], pairs[i]
		})
	}
	if maxPairs > 0 && len(candidates) > maxPairs {
		candidates = candidates[:maxPairs]
		pairs = pairs[:maxPairs]
	}

	return candidates, pairs
}

func identifyExternalSections(
	baseUniSched Schedule.UniTimeTables,
	curriculums []Curriculum.Curriculum,
	departmentToEncode map[uint16]bool,
	selectedSemester int,
	skipCurriculaCodes []string,
) map[uint16]map[uint16]bool {
	skipMap := make(map[string]bool, len(skipCurriculaCodes))
	for _, code := range skipCurriculaCodes {
		if trimmed := strings.TrimSpace(code); trimmed != "" {
			skipMap[trimmed] = true
		}
	}
	external := make(map[uint16]map[uint16]bool)

	IterateSectionsWeekSchedule(baseUniSched, curriculums, selectedSemester, nil, nil,
		func(indicies IterIndices, values IterValues) IterReturnType {
			isToEncode, hasKey := departmentToEncode[values.Curriculum.DepartmentID]
			if !(isToEncode && hasKey) {
				return IterProceed
			}
			if int(indicies.Usi) >= len(baseUniSched) {
				return IterProceed
			}

			// Explicit skip via GA_SKIP_CURRICULA env var.
			if skipMap[values.Curriculum.CurriculumCode] {
				subjects := make(map[uint16]bool, len(values.Semester.Subjects))
				for _, subj := range values.Semester.Subjects {
					subjects[subj.ID] = true
				}
				external[uint16(indicies.Usi)] = subjects
				return IterProceed
			}

			placed := make(map[uint16]bool)
			week := baseUniSched[indicies.Usi]
			for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
				for slot := 0; slot < Const.N_DAILY_TIME_SLOTS; slot++ {
					sid := week[day][slot].GetSubjectID()
					if sid > 0 {
						placed[sid] = true
					}
				}
			}

			expected := len(values.Semester.Subjects)
			if len(placed) > 0 && len(placed) < expected {
				subjects := make(map[uint16]bool, expected)
				for _, subj := range values.Semester.Subjects {
					subjects[subj.ID] = true
				}
				external[uint16(indicies.Usi)] = subjects
			}
			return IterProceed
		},
	)
	return external
}

// applyExternalSectionsToEncodingResource adds every curriculum subject of
// every external section into resource.IsSchedIdxToSubIdToSkip, idempotently.
func applyExternalSectionsToEncodingResource(
	resource *EncodingResource,
	externalSubjects map[uint16]map[uint16]bool,
) {
	if resource == nil || len(externalSubjects) == 0 {
		return
	}
	for usi, subjects := range externalSubjects {
		if _, exists := resource.IsSchedIdxToSubIdToSkip[usi]; !exists {
			resource.IsSchedIdxToSubIdToSkip[usi] = make(map[uint16]bool)
		}
		for sid := range subjects {
			resource.IsSchedIdxToSubIdToSkip[usi][sid] = true
		}
	}
}

// buildFreshIndividual generates a single new genesis-style individual.
// Used by stagnation injection to introduce fresh genetic material when the
// population has converged to a local optimum.
//
// Identical to the body of the genesis loop in RunGeneticAlgorithm, factored
// out so it can be called per-injection without duplicating the logic:
//  1. Copy the base schedule
//  2. Clear the active department's portion
//  3. Build a fresh encoding resource with the external-section skip applied
//  4. Encode the active department from scratch
//
// Returns nil + error if the new individual cannot be built within a 10s
// timeout — caller should keep the existing individual in that slot.
func buildFreshIndividual(
	base_uni_sched Schedule.UniTimeTables,
	curriculums []Curriculum.Curriculum,
	dept_id_to_department map[uint16]Departments.Department,
	default_empty_encoding_resource *EncodingResource,
	externalSectionSubjects map[uint16]map[uint16]bool,
	department_to_encode map[uint16]bool,
	department_id uint16,
	selected_semester int,
) (*SchedAndResources, error) {
	copy_uni_sched := make(Schedule.UniTimeTables, len(base_uni_sched))
	if copied := copy(copy_uni_sched, base_uni_sched); copied != len(base_uni_sched) {
		return nil, fmt.Errorf("buildFreshIndividual: copy failed — copied %d of %d", copied, len(base_uni_sched))
	}

	ClearDepartmentSchedule(copy_uni_sched, curriculums, department_id, selected_semester)

	copy_encoding_resource, err := GenerateEncodingResourceFromUniTimeTable(
		copy_uni_sched, curriculums, selected_semester, default_empty_encoding_resource,
	)
	if err != nil {
		return nil, fmt.Errorf("buildFreshIndividual: encoding resource: %w", err)
	}

	applyExternalSectionsToEncodingResource(copy_encoding_resource, externalSectionSubjects)

	done := make(chan struct{})
	var encodeErr error
	var fresh_sched Schedule.UniTimeTables
	var fresh_resource *EncodingResource

	go func() {
		defer close(done)
		fresh_sched, fresh_resource, encodeErr = EncodeIndividualGenome(
			copy_uni_sched, curriculums, dept_id_to_department,
			copy_encoding_resource, department_to_encode, selected_semester, 0,
		)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("buildFreshIndividual: encoding timed out after 10s")
	}

	if encodeErr != nil {
		return nil, fmt.Errorf("buildFreshIndividual: encoding failed: %w", encodeErr)
	}
	return &SchedAndResources{UniSched: fresh_sched, Resources: fresh_resource}, nil
}

// ── cache helpers ─────────────────────────────────────────────────────────────

type fitnessCacheKey struct {
	scheduleHash uint64
	deptKey      uint32
}

func hashSchedule(sched Schedule.UniTimeTables) uint64 {
	serialized := Schedule.SerializeUniversitySchedule(sched)
	h := fnv.New64a()
	_, _ = h.Write(serialized)
	return h.Sum64()
}

func deptFilterKey(deptFilter map[uint16]bool) uint32 {
	if len(deptFilter) == 0 {
		return 0
	}
	keys := make([]int, 0, len(deptFilter))
	for k, enabled := range deptFilter {
		if enabled {
			keys = append(keys, int(k))
		}
	}
	if len(keys) == 0 {
		return 0
	}
	sort.Ints(keys)
	h := fnv.New32a()
	for _, k := range keys {
		_, _ = h.Write([]byte{byte(k), byte(k >> 8)})
	}
	return h.Sum32()
}

func makeFitnessCacheKey(sched Schedule.UniTimeTables, deptFilter map[uint16]bool) fitnessCacheKey {
	return fitnessCacheKey{scheduleHash: hashSchedule(sched), deptKey: deptFilterKey(deptFilter)}
}

func resetFitnessCacheIfNeeded(cache map[fitnessCacheKey]float64) map[fitnessCacheKey]float64 {
	if len(cache) <= MAX_FITNESS_CACHE_SIZE {
		return cache
	}
	return make(map[fitnessCacheKey]float64)
}

// ── env helpers ───────────────────────────────────────────────────────────────

func envInt(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func envFloat(name string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

// ── top-K resolution ─────────────────────────────────────────────────────────

func resolveAnnTopK(popSize int, ratio float64, min int) int {
	if popSize <= 0 {
		return 0
	}
	k := int(math.Ceil(float64(popSize) * ratio))
	if k < min {
		k = min
	}
	if k > popSize {
		k = popSize
	}
	return k
}

// ── fitness evaluation ────────────────────────────────────────────────────────

type annBatchScheduleInfo struct {
	start int
	count int
}

func measurePopulationFitnessClassic(
	schedules []Schedule.UniTimeTables,
	allCurriculums []Curriculum.Curriculum,
	departmentToMeasure map[uint16]bool,
	selectedSemester int,
	stats *ANNFitnessEvalStats,
	fitnessCache map[fitnessCacheKey]float64,
) []float64 {
	results := make([]float64, len(schedules))
	for i, sched := range schedules {
		if sched.IsEmpty() {
			results[i] = -24.0
			continue
		}
		if fitnessCache != nil {
			cacheKey := makeFitnessCacheKey(sched, departmentToMeasure)
			if cached, ok := fitnessCache[cacheKey]; ok {
				if stats != nil {
					stats.CacheHitCount++
				}
				results[i] = cached
				continue
			}
			if stats != nil {
				stats.CacheMissCount++
			}
		}
		if stats != nil {
			stats.ClassicEvalCount++
		}
		fitness := MeasureUniSchedBasicFitness(sched, allCurriculums, departmentToMeasure, selectedSemester)
		results[i] = fitness
		if fitnessCache != nil {
			fitnessCache[makeFitnessCacheKey(sched, departmentToMeasure)] = fitness
		}
	}
	return results
}

// measurePopulationFitnessWithANNTopK implements step 3 of the hybrid flow.
//
// FALLBACK: When annClient == nil this routes immediately to
// measurePopulationFitnessClassic.  No ANN code is touched, no extra work
// is done.  This is the foundation of the pure-GA mode.
func measurePopulationFitnessWithANNTopK(
	annClient *ANNClient,
	schedules []Schedule.UniTimeTables,
	allCurriculums []Curriculum.Curriculum,
	departmentToMeasure map[uint16]bool,
	selectedSemester int,
	stats *ANNFitnessEvalStats,
	classicCache map[fitnessCacheKey]float64,
	annCache map[fitnessCacheKey]float64,
	rng *rand.Rand,
	annTopKRatio float64,
	annTopKMin int,
) []float64 {
	_ = rng

	if annClient == nil {
		return measurePopulationFitnessClassic(
			schedules, allCurriculums, departmentToMeasure, selectedSemester, stats, classicCache,
		)
	}

	rankingScores, annErr := measurePopulationRankingScores(
		annClient, schedules, allCurriculums, departmentToMeasure, selectedSemester, stats, annCache,
	)
	if annErr != nil {
		// Mid-run ANN failure: fall back to classic for this generation only.
		log.Printf("ga: [step-3] ANN ranking failed (%s) — falling back to classic fitness for this generation", annErr)
		if stats != nil {
			stats.ANNFailureCount++
			stats.ANNFallbackCount++
			stats.FallbackEvalCount++
		}
		return measurePopulationFitnessClassic(
			schedules, allCurriculums, departmentToMeasure, selectedSemester, stats, classicCache,
		)
	}

	topK := resolveAnnTopK(len(schedules), annTopKRatio, annTopKMin)
	if topK == 0 {
		return rankingScores
	}

	indices := make([]int, len(schedules))
	for i := range schedules {
		indices[i] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		return rankingScores[indices[i]] > rankingScores[indices[j]]
	})
	if topK > len(indices) {
		topK = len(indices)
	}

	finalistIndices := indices[:topK]
	finalistSchedules := make([]Schedule.UniTimeTables, len(finalistIndices))
	for i, idx := range finalistIndices {
		finalistSchedules[i] = schedules[idx]
	}
	classicFitness := measurePopulationFitnessClassic(
		finalistSchedules, allCurriculums, departmentToMeasure, selectedSemester, stats, classicCache,
	)
	for i, idx := range finalistIndices {
		rankingScores[idx] = classicFitness[i]
	}
	return rankingScores
}

func measurePopulationRankingScores(
	annClient *ANNClient,
	schedules []Schedule.UniTimeTables,
	allCurriculums []Curriculum.Curriculum,
	departmentToMeasure map[uint16]bool,
	selectedSemester int,
	stats *ANNFitnessEvalStats,
	fitnessCache map[fitnessCacheKey]float64,
) ([]float64, error) {
	results := make([]float64, len(schedules))
	if annClient == nil {
		return measurePopulationFitnessClassic(
			schedules, allCurriculums, departmentToMeasure, selectedSemester, stats, fitnessCache,
		), nil
	}

	allPayloads := make([][][][]int, 0)
	infoByIndex := make(map[int]annBatchScheduleInfo, len(schedules))
	annScheduleIdx := make([]int, 0, len(schedules))

	for i, sched := range schedules {
		if sched.IsEmpty() {
			results[i] = -24.0
			continue
		}
		if fitnessCache != nil {
			cacheKey := makeFitnessCacheKey(sched, departmentToMeasure)
			if cached, ok := fitnessCache[cacheKey]; ok {
				if stats != nil {
					stats.CacheHitCount++
				}
				results[i] = cached
				continue
			}
			if stats != nil {
				stats.CacheMissCount++
			}
		}
		payloads := collectSectionPayloads(sched, allCurriculums, departmentToMeasure, selectedSemester)
		if len(payloads) == 0 {
			results[i] = -24.0
			continue
		}
		infoByIndex[i] = annBatchScheduleInfo{start: len(allPayloads), count: len(payloads)}
		annScheduleIdx = append(annScheduleIdx, i)
		allPayloads = append(allPayloads, payloads...)
		if stats != nil {
			stats.ANNRequestCount += len(payloads)
		}
	}

	if len(allPayloads) == 0 {
		return results, nil
	}

	predictions, err := annClient.BatchPredictFitness(allPayloads)
	if err != nil {
		if stats != nil {
			stats.ANNFailureCount++
		}
		return nil, err
	}

	for _, idx := range annScheduleIdx {
		info := infoByIndex[idx]
		if info.count == 0 {
			results[idx] = -24.0
			continue
		}
		acc := 0.0
		for i := 0; i < info.count; i++ {
			acc += predictions[info.start+i]
		}
		annFitness := acc / float64(info.count)
		if stats != nil {
			stats.ANNPredictionCount += info.count
			stats.ANNUsedEvalCount++
		}
		results[idx] = annFitness
		if fitnessCache != nil {
			fitnessCache[makeFitnessCacheKey(schedules[idx], departmentToMeasure)] = annFitness
		}
	}
	return results, nil
}

// ══════════════════════════════════════════════════════════════════════════════
//  RunGeneticAlgorithm — hybrid ANN-assisted GA with pure-GA fallback
//
//  WHEN annActive (ann_client != nil):
//  ─────────────────────────────────────
//    Step 3  Fitness model    — ANN batch-ranks full population; classic re-scores top-K
//    Step 4  Crossover model  — ANN guides every crossover (toggle with GA_ANN_CROSSOVER_ENABLED)
//    Step 5  Constraint model — checks all new offspring; penalises violators before sort
//    Step 6  Mutation model   — reverts mutations predicted to "worsen" with high confidence
//
//  WHEN !annActive (ann_client == nil) — pure classic GA:
//  ──────────────────────────────────────────────────────
//    Step 3 → measurePopulationFitnessWithANNTopK detects nil and routes to classic
//    Step 4 → annForCrossover stays nil; Crossover uses random split-point selection
//    Step 5 → constraint screen block is skipped entirely; postCrossoverPenalties stays nil
//    Step 6 → mutation prediction block is skipped entirely; all valid mutations are kept
//
//  All ANN-touching code paths gate on the single annActive flag, so disabling
//  ANN is one nil pointer away from a clean classic GA — no API calls, no
//  payload collection, no penalty allocation, no mutation reverts.
// ══════════════════════════════════════════════════════════════════════════════

func RunGeneticAlgorithm(
	base_uni_sched Schedule.UniTimeTables,
	curriculums []Curriculum.Curriculum,
	rooms []Rooms.Room,
	dept_id_to_department map[uint16]Departments.Department,
	default_empty_encoding_resource, base_encoding_resource *EncodingResource,
	department_to_encode map[uint16]bool,
	selected_semester, population_size, generations int,
	resource_persistence *StorageResources.Persistence,
	ann_client *ANNClient,
	cb_fn_generation func(generation int, generation_fittest_sched Schedule.UniTimeTables, fitness float64),
) (Schedule.UniTimeTables, *EncodingResource, error) {

	rng := rand.New(rand.NewSource(time.Now().UnixMilli()))

	// ── Training-data collection (opt-in via env var) ────────────────────────
	// When COLLECT_TRAINING_DATA_DIR is empty NewDataCollector returns a
	// disabled collector and every Log* call below is a no-op, so the GA's
	// hot loop pays nothing in the default configuration.
	collectDir := os.Getenv("COLLECT_TRAINING_DATA_DIR")
	dataCollector, err_dc := NewDataCollector(collectDir)
	if err_dc != nil {
		log.Printf("data_collector: disabled (init failed): %s", err_dc)
		dataCollector = &DataCollector{}
	}
	defer dataCollector.Close()

	// ── ANN availability flag ────────────────────────────────────────────────
	// Single source of truth for "is ANN guidance active this run".  Every
	// ANN-using code path below gates on this boolean, so the GA degrades
	// cleanly to a pure classic GA when ann_client is nil.
	annActive := ann_client != nil

	department_id := uint16(0)
	if len(department_to_encode) != 1 {
		panic("department to encode should only have one value for now")
	}
	for k := range department_to_encode {
		department_id = k
	}

	default_instructor_id_to_instructor, err := GenerateMapInstructorIdToInstructor(resource_persistence)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to generate instructor id to instructor map: %w", err)
	}

	// gaLoadConstraintErrors enforces the two new hard constraints — the 4-day
	// packing rule and the per-instructor weekly unit cap — at the same gates as
	// HorizontalValidation. Kept separate from HV so non-GA serve routes don't
	// retroactively reject pre-existing schedules (see LoadConstraints.go).
	gaLoadConstraintErrors := func(sched Schedule.UniTimeTables) []error {
		errs := ValidateFourDayPacking(sched, curriculums, department_to_encode, selected_semester)
		errs = append(errs, ValidateInstructorUnitLoad(sched, curriculums, default_instructor_id_to_instructor, selected_semester)...)
		return errs
	}

	classicFitnessCache := make(map[fitnessCacheKey]float64)
	annFitnessCache := make(map[fitnessCacheKey]float64)

	// ── External-section detection ───────────────────────────────────────────
	skipCurriculaCodes := []string{}
	if raw := strings.TrimSpace(os.Getenv("GA_SKIP_CURRICULA")); raw != "" {
		for _, code := range strings.Split(raw, ",") {
			if trimmed := strings.TrimSpace(code); trimmed != "" {
				skipCurriculaCodes = append(skipCurriculaCodes, trimmed)
			}
		}
	}
	externalSectionSubjects := identifyExternalSections(
		base_uni_sched, curriculums, department_to_encode, selected_semester,
		skipCurriculaCodes,
	)
	if len(skipCurriculaCodes) > 0 {
		log.Printf("ga: GA_SKIP_CURRICULA=%v — those curricula will be left untouched", skipCurriculaCodes)
	}
	if len(externalSectionSubjects) > 0 {
		log.Printf(
			"ga: detected %d external section(s) in dept %d — they will be left untouched",
			len(externalSectionSubjects), department_id,
		)
		applyExternalSectionsToEncodingResource(base_encoding_resource, externalSectionSubjects)
	}

	// ── tunable parameters ────────────────────────────────────────────────────
	earlyStopPatience := envInt("GA_EARLY_STOP_PATIENCE", 0)
	earlyStopMinGen := envInt("GA_EARLY_STOP_MIN_GEN", 0)
	earlyStopMinImprovement := envFloat("GA_EARLY_STOP_MIN_IMPROVEMENT", 0.0)

	crossoverRate := envFloat("GA_CROSSOVER_RATE", 0.85)
	mutationRate := envFloat("GA_MUTATION_RATE", 0.15)
	if crossoverRate < 0 {
		crossoverRate = 0
	} else if crossoverRate > 1 {
		crossoverRate = 1
	}
	if mutationRate < 0 {
		mutationRate = 0
	} else if mutationRate > 1 {
		mutationRate = 1
	}

	// ── Diversity / convergence-escape parameters ────────────────────────────
	// These knobs exist because schedules tend to converge fast — a few good
	// individuals dominate the population, crossover produces near-copies, and
	// mutation alone can't introduce enough novelty to escape local optima.
	//
	//   • tournamentSize (default 3) — number of individuals randomly drawn for
	//     each tournament.  Binary tournaments (size 2) are too soft; size 3-5
	//     applies stronger selection pressure without being too greedy.
	//   • adaptiveMutation (default on) — when fitness stagnates, multiply the
	//     mutation rate by (1 + 0.2 × stagnant_generations), capped at
	//     maxMutationRate.  More mutation = more exploration when stuck.
	//   • maxMutationRate (default 0.50) — ceiling for adaptive mutation.
	//   • stagnationInjectThreshold (default 5) — after this many stagnant
	//     generations, replace the worst stagnationInjectFraction of the
	//     population with freshly-generated random individuals.  Brings new
	//     genetic material that crossover and mutation cannot synthesise.
	//   • stagnationInjectFraction (default 0.20) — fraction of population to
	//     replace per injection event.
	tournamentSize := envInt("GA_TOURNAMENT_SIZE", 3)
	if tournamentSize < 2 {
		tournamentSize = 2
	}
	adaptiveMutation := envInt("GA_ADAPTIVE_MUTATION", 1) != 0
	maxMutationRate := envFloat("GA_MAX_MUTATION_RATE", 0.50)
	if maxMutationRate < mutationRate {
		maxMutationRate = mutationRate
	}
	if maxMutationRate > 1 {
		maxMutationRate = 1
	}
	stagnationInjectThreshold := envInt("GA_STAGNATION_INJECT_THRESHOLD", 5)
	stagnationInjectFraction := envFloat("GA_STAGNATION_INJECT_FRACTION", 0.20)
	if stagnationInjectFraction < 0 {
		stagnationInjectFraction = 0
	} else if stagnationInjectFraction > 0.5 {
		stagnationInjectFraction = 0.5 // never replace more than half the population
	}

	// ── Quality-pressure parameters ──────────────────────────────────────────
	// Two protections against the population accumulating dead weight.
	//
	//   • hillClimbThreshold (default 2.0) — when a mutation drops fitness by
	//     more than this many points, revert it.  Lenient enough to allow
	//     small drops (preserves exploration / escapes local optima), strict
	//     enough to reject catastrophic mutations.  Set to 0 to disable.
	//   • purgeEmpties (default on) — after fitness evaluation each generation,
	//     replace any individual whose schedule is empty (fitness == -24)
	//     with a fresh random one.  Empty schedules accumulate in the
	//     population otherwise; they survive sorts because they're always
	//     last, but they're dead weight — they can't be parents (selection
	//     filters them out), can't contribute to crossover, and pollute the
	//     average fitness reading.
	hillClimbThreshold := envFloat("GA_HILL_CLIMB_THRESHOLD", 2.0)
	purgeEmpties := envInt("GA_PURGE_EMPTIES", 1) != 0

	// ANN-only knobs.  Read but only used when annActive.
	annCrossoverEnabled := envInt("GA_ANN_CROSSOVER_ENABLED", 1) != 0
	annConstraintEnabled := envInt("GA_ANN_CONSTRAINT_ENABLED", 1) != 0
	annMutationEnabled := envInt("GA_ANN_MUTATION_ENABLED", 1) != 0
	annConstraintPenalty := envFloat("GA_ANN_CONSTRAINT_PENALTY", ANN_CONSTRAINT_PENALTY_DEFAULT)
	annMutationRevertConfidence := envFloat("GA_ANN_MUTATION_REVERT_CONFIDENCE", ANN_MUTATION_REVERT_CONFIDENCE_DEFAULT)
	annTopKRatio := envFloat("GA_ANN_TOP_K_RATIO", ANN_TOP_K_RATIO)
	annTopKMin := envInt("GA_ANN_TOP_K_MIN", ANN_TOP_K_MIN)
	if annTopKRatio < 0 {
		annTopKRatio = ANN_TOP_K_RATIO
	} else if annTopKRatio > 1 {
		annTopKRatio = 1
	}
	if annTopKMin < 0 {
		annTopKMin = ANN_TOP_K_MIN
	}

	// ── Inter-subject gap constraint (see gap_constraint.go) ──────────────────
	// Soft 1-2 hour gap between different subjects on the same day. Read here so
	// it can be tuned without recompilation; installed into the package-wide
	// gapConfig consulted by the fitness function, the encoder and horizontal
	// validation. GA_MIN_GAP_HOURS=0 disables it entirely (backward compatible).
	gapCfg := LoadGapConfigFromEnv()

	// Build the set of subjects allowed on Saturday (NSTP 1/2 + SaturdayOnly) so
	// the fitness function keeps Saturday reserved for NSTP and steers regular
	// subjects onto weekdays.
	SetSaturdayAllowedSubjects(curriculums)

	bestFitnessSeen := math.Inf(-1)
	stagnantGenerations := 0
	totalInjectionsPerformed := 0

	classicFitnessOf := func(sched Schedule.UniTimeTables, deptFilter map[uint16]bool) float64 {
		cacheKey := makeFitnessCacheKey(sched, deptFilter)
		if cached, ok := classicFitnessCache[cacheKey]; ok {
			return cached
		}
		fitness := MeasureUniSchedBasicFitness(sched, curriculums, deptFilter, selected_semester)
		classicFitnessCache[cacheKey] = fitness
		return fitness
	}

	log.Printf("||||||||||||||||||||||||||||||| Performing GA with %s |||||||||||||||||||||||||||||||",
		dept_id_to_department[department_id].Name)

	// ── ANN status banner ────────────────────────────────────────────────────
	// Print this front and centre so the active mode is unambiguous in the log.
	// Crucial for GA-vs-GA+ANN comparison runs: if you expected ANN to be on
	// but see [ANN DISABLED] here, the ann_client never reached this function
	// (check USE_ANN_FOR_GA env var and the encode_schedule wiring).
	if annActive {
		log.Printf(
			"ga: [ANN ENABLED] — fitness ranking active; crossover=%t constraint=%t mutation=%t",
			annCrossoverEnabled, annConstraintEnabled, annMutationEnabled,
		)
	} else {
		log.Printf("ga: [ANN DISABLED] — running pure classic Genetic Algorithm (no ANN calls will be made)")
	}

	log.Printf("ga: parameters — population=%d generations=%d crossover_rate=%.2f mutation_rate=%.2f elite=%d",
		population_size, generations, crossoverRate, mutationRate, ELITE_PRESERVE_COUNT)
	log.Printf("ga: diversity — tournament_size=%d adaptive_mutation=%t max_mutation_rate=%.2f stagnation_inject[threshold=%d fraction=%.2f]",
		tournamentSize, adaptiveMutation, maxMutationRate, stagnationInjectThreshold, stagnationInjectFraction)
	if gapCfg.Enabled {
		log.Printf("ga: gap constraint — ENABLED min=%d slots max=%d slots penalty=%.2f reward=%.2f apply_on_saturday=%t",
			gapCfg.MinGapSlots, gapCfg.MaxGapSlots, gapCfg.Penalty, gapCfg.Reward, gapCfg.ApplyOnSaturday)
	} else {
		log.Printf("ga: gap constraint — DISABLED (GA_MIN_GAP_HOURS=0); behaviour identical to pre-constraint system")
	}
	log.Printf("ga: quality — hill_climb_threshold=%.2f purge_empties=%t",
		hillClimbThreshold, purgeEmpties)
	if annActive {
		log.Printf("ga: ann parameters — crossover_enabled=%t constraint_enabled=%t mutation_enabled=%t top_k_ratio=%.2f top_k_min=%d constraint_penalty=%.2f mutation_revert_confidence=%.2f",
			annCrossoverEnabled, annConstraintEnabled, annMutationEnabled, annTopKRatio, annTopKMin,
			annConstraintPenalty, annMutationRevertConfidence)
	}

	////////////////////////////////////////////////////////////////////////////////////////
	// STEP 1 — BASE SCHEDULE REPAIR
	////////////////////////////////////////////////////////////////////////////////////////

	genesis_population := make([]SchedAndResources, 0)
	base_schedule_repair_tries := 0

	for {
		log.Printf("RunGeneticAlgorithm [Base Repair]: attempt %d/%d",
			base_schedule_repair_tries+1, MAX_BASE_SCHEDULE_REPAIR_TRAIALS)

		done := make(chan struct{})
		var err_encoding_new_base_sched error
		var new_base_sched Schedule.UniTimeTables
		var new_base_sched_resource *EncodingResource

		go func() {
			defer close(done)
			new_base_sched, new_base_sched_resource, err_encoding_new_base_sched = EncodeIndividualGenome(
				base_uni_sched, curriculums, dept_id_to_department,
				base_encoding_resource, department_to_encode, selected_semester, 0,
			)
		}()

		select {
		case <-done:
		case <-time.After(10 * time.Second):
			return nil, nil, fmt.Errorf("base schedule repair timed out after 10 seconds")
		}

		base_schedule_repair_tries++

		if err_encoding_new_base_sched != nil {
			if base_schedule_repair_tries < MAX_BASE_SCHEDULE_REPAIR_TRAIALS {
				continue
			}
			return new_base_sched, new_base_sched_resource, fmt.Errorf(
				"base schedule repair error after %d tries: %w",
				MAX_BASE_SCHEDULE_REPAIR_TRAIALS, err_encoding_new_base_sched,
			)
		}

		log.Printf("university schedule fitness : %f", classicFitnessOf(base_uni_sched, nil))
		log.Printf("department schedule fitness : %f", classicFitnessOf(base_uni_sched, department_to_encode))

		genesis_population = append(genesis_population, SchedAndResources{
			UniSched:  new_base_sched,
			Resources: new_base_sched_resource,
		})
		break
	}

	////////////////////////////////////////////////////////////////////////////////////////
	// STEP 2 — GENESIS POPULATION
	////////////////////////////////////////////////////////////////////////////////////////

	start := time.Now()
	genesis_generation_tries := 0
	log.Print("generating genesis population...")

	for len(genesis_population) < population_size {
		copy_uni_sched := make(Schedule.UniTimeTables, len(base_uni_sched))
		if copied := copy(copy_uni_sched, base_uni_sched); copied != len(base_uni_sched) {
			return nil, nil, fmt.Errorf("internal copy failed: copied %d of %d", copied, len(base_uni_sched))
		}

		ClearDepartmentSchedule(copy_uni_sched, curriculums, department_id, selected_semester)

		copy_encoding_resource, err_enc := GenerateEncodingResourceFromUniTimeTable(
			copy_uni_sched, curriculums, selected_semester, default_empty_encoding_resource,
		)
		if err_enc != nil {
			return nil, nil, fmt.Errorf("unable to generate encoding resource during genesis: %w", err_enc)
		}

		applyExternalSectionsToEncodingResource(copy_encoding_resource, externalSectionSubjects)

		done := make(chan struct{})
		var err_encode_initial error
		var initial_sched Schedule.UniTimeTables
		var initial_encoding_resource *EncodingResource

		go func() {
			defer close(done)
			initial_sched, initial_encoding_resource, err_encode_initial = EncodeIndividualGenome(
				copy_uni_sched, curriculums, dept_id_to_department,
				copy_encoding_resource, department_to_encode, selected_semester, 0,
			)
		}()

		select {
		case <-done:
		case <-time.After(10 * time.Second):
			return nil, nil, fmt.Errorf("genesis generation timed out after 10 seconds")
		}

		// POINT A — Constraint sample collection (one per section per genesis
		// individual).  Hard violations are evaluated once for the whole
		// schedule and merged with each section's own soft-constraint flags
		// — per-section semantics avoid the "always true" degeneracy that
		// OR-ing soft flags across the entire university produces.
		if dataCollector.IsEnabled() && err_encode_initial == nil {
			hardViolations := detectHardViolations(
				initial_sched, curriculums, rooms,
				department_to_encode, selected_semester,
			)
			crossStats := newCrossSectionStats(
				initial_sched, curriculums,
				department_to_encode, selected_semester,
			)
			IterateSectionsWeekSchedule(initial_sched, curriculums, selected_semester, nil, nil,
				func(indices IterIndices, values IterValues) IterReturnType {
					if values.WeekSched == nil || values.Curriculum == nil {
						return IterProceed
					}
					if !department_to_encode[values.Curriculum.DepartmentID] {
						return IterProceed
					}
					soft := detectSoftViolationsForSection(*values.WeekSched)
					dataCollector.LogConstraint(ConstraintSample{
						SectionSchedule: weekTimeTableToSlice(*values.WeekSched),
						DepartmentID:    department_id,
						Violations:      mergeViolations(hardViolations, soft),
						CrossSection:    crossStats.computeAggregatesForSection(*values.WeekSched),
					})
					return IterProceed
				},
			)
			_ = initial_encoding_resource
		}

		if err_encode_initial != nil {
			genesis_generation_tries++
			log.Printf("RunGeneticAlgorithm [Genesis Population]: retry %d, cause: %s",
				genesis_generation_tries, err_encode_initial.Error())

			if genesis_generation_tries >= MAX_GENESIS_INDIVIDUAL_GENERATION_TRIALS {
				if initial_sched == nil {
					return nil, nil, fmt.Errorf(
						"unable to generate genesis individual after %d tries: %w",
						MAX_GENESIS_INDIVIDUAL_GENERATION_TRIALS, err_encode_initial,
					)
				}
				return initial_sched, nil, fmt.Errorf(
					"unable to generate genesis individual after %d tries: %w",
					MAX_GENESIS_INDIVIDUAL_GENERATION_TRIALS, err_encode_initial,
				)
			}
			continue
		}
		genesis_generation_tries = 0

		genesis_population = append(genesis_population, SchedAndResources{
			UniSched:  initial_sched,
			Resources: initial_encoding_resource,
		})
	}

	fmt.Printf("ga: [generate genesis population] - took %s\n", time.Since(start))

	{
		type schedFit struct {
			s SchedAndResources
			f float64
		}
		ranked := make([]schedFit, len(genesis_population))
		for i, s := range genesis_population {
			ranked[i] = schedFit{s, classicFitnessOf(s.UniSched, department_to_encode)}
		}
		sort.Slice(ranked, func(i, j int) bool { return ranked[i].f > ranked[j].f })
		for i, r := range ranked {
			genesis_population[i] = r.s
		}
		log.Printf("ga: genesis population sorted — best initial fitness: %.4f", ranked[0].f)
	}

	////////////////////////////////////////////////////////////////////////////////////////
	// GENERATIONAL LOOP
	////////////////////////////////////////////////////////////////////////////////////////

	for g := range generations {
		// genStats is allocated unconditionally because Crossover's signature
		// requires a non-nil pointer.  In pure-GA mode it stays at zero values
		// and is never read; cost is negligible.
		genStats := &ANNFitnessEvalStats{}
		classicFitnessCache = resetFitnessCacheIfNeeded(classicFitnessCache)
		annFitnessCache = resetFitnessCacheIfNeeded(annFitnessCache)

		log.Printf("running genetic algorithm generation %d", g)

		population := make([]SchedAndResources, 0, population_size+1)

		// ── TOURNAMENT SELECTION ──────────────────────────────────────────────
		//
		// Uses fitness values from the previous generation's evaluation.
		// In pure-GA mode this is purely classic fitness; in ANN mode it's
		// the blend of ANN-ranked + classic-rescored top-K.  Either way,
		// tournament selection itself does not call ANN.
		log.Printf("apply tournament selection to the population")
		start = time.Now()

		previousFitnessScores := make(map[fitnessCacheKey]float64, len(genesis_population))
		for _, individual := range genesis_population {
			key := makeFitnessCacheKey(individual.UniSched, department_to_encode)
			if f, ok := classicFitnessCache[key]; ok {
				previousFitnessScores[key] = f
			} else if f, ok := annFitnessCache[key]; ok {
				previousFitnessScores[key] = f
			} else {
				previousFitnessScores[key] = classicFitnessOf(individual.UniSched, department_to_encode)
			}
		}

		// ── K-WAY TOURNAMENT SELECTION ────────────────────────────────────────
		//
		// For each tournament: draw `tournamentSize` random individuals,
		// take the fittest.  Repeat until we have half the population.
		// Tournament size 3-5 applies stronger selection pressure than
		// binary (size 2): fitter individuals win disproportionately,
		// pushing average population fitness up faster, while still
		// occasionally letting weaker ones through (preserving diversity).
		//
		// Why "half the population"?  Crossover doubles each survivor pair
		// into two offspring (one survivor + one offspring per slot), so
		// half-survivors + half-offspring = full next generation.
		fitnessOfIndividual := func(ind SchedAndResources) float64 {
			key := makeFitnessCacheKey(ind.UniSched, department_to_encode)
			if f, ok := previousFitnessScores[key]; ok {
				return f
			}
			return classicFitnessOf(ind.UniSched, department_to_encode)
		}

		survivorCount := len(genesis_population) / 2
		if survivorCount < 2 {
			survivorCount = 2
		}
		for s := 0; s < survivorCount; s++ {
			bestIdx := rng.Intn(len(genesis_population))
			bestFit := fitnessOfIndividual(genesis_population[bestIdx])
			for k := 1; k < tournamentSize; k++ {
				candIdx := rng.Intn(len(genesis_population))
				candFit := fitnessOfIndividual(genesis_population[candIdx])
				if candFit > bestFit {
					bestIdx = candIdx
					bestFit = candFit
				}
			}
			population = append(population, genesis_population[bestIdx])
		}

		fmt.Printf("ga: [tournament selection] - took %s, k=%d survivors=%d\n",
			time.Since(start), tournamentSize, len(population))

		// ── STEP 4: CROSSOVER ─────────────────────────────────────────────────
		//
		// FALLBACK: When annActive == false, annForCrossover stays nil and
		// Crossover uses random split-point selection.  No ANN calls are made,
		// genStats.CrossoverRequestCount stays at 0, and crossover proceeds
		// purely classically.
		start = time.Now()
		crossover_tries := 0
		pop_before_co := len(population)

		if pop_before_co == 0 {
			panic("population size 0 after tournament selection")
		}

		rankedCrossoverPairs := make([]crossoverPairCandidate, 0)
		crossoverPairCursor := 0
		if annActive && annCrossoverEnabled {
			candidates, pairs := buildParentPairBatch(
				population[:pop_before_co], curriculums,
				department_to_encode, selected_semester,
				200, rng,
			)
			if len(pairs) > 0 {
				genStats.CrossoverRequestCount++
				predictions, err := ann_client.BatchPredictCrossover(pairs)
				if err != nil {
					genStats.CrossoverFailureCount++
					if os.Getenv("LOG_MODE") == "verbose" {
						log.Printf("ga: [crossover batch fallback] %s", err.Error())
					}
				} else {
					for i := range candidates {
						candidates[i].confidence = predictions[i].Confidence
						candidates[i].compatible = predictions[i].Compatible
					}
					sort.Slice(candidates, func(i, j int) bool {
						if candidates[i].compatible != candidates[j].compatible {
							return candidates[i].compatible
						}
						return candidates[i].confidence > candidates[j].confidence
					})
					rankedCrossoverPairs = candidates
				}
			}
		}

		for len(population) < population_size {
			parent1_idx := rng.Intn(pop_before_co)
			parent2_idx := rng.Intn(pop_before_co)
			if parent1_idx == parent2_idx {
				continue
			}
			if crossoverPairCursor < len(rankedCrossoverPairs) {
				pair := rankedCrossoverPairs[crossoverPairCursor]
				crossoverPairCursor++
				parent1_idx = pair.parent1Idx
				parent2_idx = pair.parent2Idx
			}

			// Probabilistic crossover roll — classic GA contract.
			if rng.Float64() >= crossoverRate {
				parentToClone := population[parent1_idx].UniSched
				if classicFitnessOf(population[parent2_idx].UniSched, department_to_encode) >
					classicFitnessOf(population[parent1_idx].UniSched, department_to_encode) {
					parentToClone = population[parent2_idx].UniSched
				}
				cloned, err_clone := cloneParentAsOffspring(
					parentToClone, default_empty_encoding_resource, curriculums, selected_semester,
				)
				if err_clone != nil {
					return nil, nil, fmt.Errorf("crossover-rate clone failed at generation %d: %w", g, err_clone)
				}
				crossover_tries = 0
				population = append(population, *cloned)
				continue
			}

			offspring, err_co := Crossover(
				population[parent1_idx].UniSched, population[parent2_idx].UniSched,
				default_empty_encoding_resource, curriculums, rooms, selected_semester,
				dept_id_to_department, department_to_encode,
				default_instructor_id_to_instructor, resource_persistence,
				nil, genStats,
			)

			// POINT C — Crossover sample collection.  Captures first section of
			// each parent and the resulting offspring fitness (or 0 when invalid).
			if dataCollector.IsEnabled() {
				p1Fitness := classicFitnessOf(population[parent1_idx].UniSched, department_to_encode)
				p2Fitness := classicFitnessOf(population[parent2_idx].UniSched, department_to_encode)

				var p1Section, p2Section [][][]int
				IterateSectionsWeekSchedule(population[parent1_idx].UniSched, curriculums, selected_semester, nil, nil,
					func(indices IterIndices, values IterValues) IterReturnType {
						if p1Section != nil {
							return IterBreakCurriculumLoop
						}
						if values.WeekSched == nil || values.Curriculum == nil {
							return IterProceed
						}
						if !department_to_encode[values.Curriculum.DepartmentID] {
							return IterProceed
						}
						p1Section = weekTimeTableToSlice(*values.WeekSched)
						return IterBreakCurriculumLoop
					},
				)
				IterateSectionsWeekSchedule(population[parent2_idx].UniSched, curriculums, selected_semester, nil, nil,
					func(indices IterIndices, values IterValues) IterReturnType {
						if p2Section != nil {
							return IterBreakCurriculumLoop
						}
						if values.WeekSched == nil || values.Curriculum == nil {
							return IterProceed
						}
						if !department_to_encode[values.Curriculum.DepartmentID] {
							return IterProceed
						}
						p2Section = weekTimeTableToSlice(*values.WeekSched)
						return IterBreakCurriculumLoop
					},
				)

				valid := 0
				offspringFit := 0.0
				if err_co == nil && offspring != nil {
					valid = 1
					offspringFit = classicFitnessOf(offspring.UniSched, department_to_encode)
				}

				if p1Section != nil && p2Section != nil {
					dataCollector.LogCrossover(CrossoverSample{
						Parent1:          p1Section,
						Parent2:          p2Section,
						ProducedValid:    valid,
						OffspringFitness: offspringFit,
						Parent1Fitness:   p1Fitness,
						Parent2Fitness:   p2Fitness,
						DepartmentID:     department_id,
						Generation:       g,
					})
				}

				// POINT C' — emit constraint samples from each in-department
				// section of the offspring.  Provides another stream of
				// constraint-classifier training rows from schedules produced
				// by crossover rather than mutation/genesis.
				if err_co == nil && offspring != nil {
					hardViolations := detectHardViolations(
						offspring.UniSched, curriculums, rooms,
						department_to_encode, selected_semester,
					)
					crossStats := newCrossSectionStats(
						offspring.UniSched, curriculums,
						department_to_encode, selected_semester,
					)
					IterateSectionsWeekSchedule(offspring.UniSched, curriculums, selected_semester, nil, nil,
						func(indices IterIndices, values IterValues) IterReturnType {
							if values.WeekSched == nil || values.Curriculum == nil {
								return IterProceed
							}
							if !department_to_encode[values.Curriculum.DepartmentID] {
								return IterProceed
							}
							soft := detectSoftViolationsForSection(*values.WeekSched)
							dataCollector.LogConstraint(ConstraintSample{
								SectionSchedule: weekTimeTableToSlice(*values.WeekSched),
								DepartmentID:    department_id,
								Violations:      mergeViolations(hardViolations, soft),
								CrossSection:    crossStats.computeAggregatesForSection(*values.WeekSched),
							})
							return IterProceed
						},
					)
				}
			}

			if err_co != nil {
				crossover_tries++
				if crossover_tries >= MAX_CROSSOVER_TRIALS {
					return nil, nil, fmt.Errorf(
						"unable to produce offspring at generation %d after %d tries: %w",
						g, MAX_CROSSOVER_TRIALS, err_co,
					)
				}
				continue
			}
			crossover_tries = 0
			population = append(population, *offspring)
		}

		if annActive {
			fmt.Printf("ga: [crossover] - took %s (ann calls: %d, ann fail: %d)\n",
				time.Since(start), genStats.CrossoverRequestCount, genStats.CrossoverFailureCount)
		} else {
			fmt.Printf("ga: [crossover] - took %s (classic, no ANN)\n", time.Since(start))
		}

		// Preserve elite individuals.
		eliteCount := ELITE_PRESERVE_COUNT
		if eliteCount > len(genesis_population) {
			eliteCount = len(genesis_population)
		}
		if eliteCount > len(population) {
			eliteCount = len(population)
		}
		for i := 0; i < eliteCount; i++ {
			population[i] = genesis_population[i]
		}

		// ── STEP 5: CONSTRAINT PRE-SCREENING ──────────────────────────────────
		//
		// FALLBACK: When annActive == false, this entire block is skipped.
		// postCrossoverPenalties stays nil — no allocation, no iteration over
		// the population, no API calls, no log line.  The penalty-application
		// loop further down also skips (gated on annActive && != nil).
		var postCrossoverPenalties []float64

		if annActive && annConstraintEnabled {
			start = time.Now()
			postCrossoverPenalties = make([]float64, len(population))
			penaltyCount := 0
			allConstraintPayloads := make([][][][]int, 0)
			constraintInfoByIndex := make(map[int]annBatchScheduleInfo, len(population))
			for i := eliteCount; i < len(population); i++ {
				payloads := collectSectionPayloads(
					population[i].UniSched, curriculums, department_to_encode, selected_semester,
				)
				if len(payloads) == 0 {
					continue
				}
				constraintInfoByIndex[i] = annBatchScheduleInfo{start: len(allConstraintPayloads), count: len(payloads)}
				allConstraintPayloads = append(allConstraintPayloads, payloads...)
			}
			checkedCount := len(allConstraintPayloads)
			if checkedCount > 0 {
				genStats.ConstraintRequestCount++
				predictions, err := ann_client.BatchPredictConstraints(allConstraintPayloads)
				if err != nil {
					genStats.ConstraintFailureCount++
					if os.Getenv("LOG_MODE") == "verbose" {
						log.Printf("ga: [constraint batch fallback] %s", err.Error())
					}
				} else {
					for i, info := range constraintInfoByIndex {
						violationCount := 0
						for offset := 0; offset < info.count; offset++ {
							pred := predictions[info.start+offset]
							if pred.InstructorConflict > 0.7 {
								violationCount++
							}
							if pred.RoomConflict > 0.7 {
								violationCount++
							}
							if pred.NoLunchBreak > 0.5 {
								violationCount++
							}
							if pred.LateClasses > 0.5 {
								violationCount++
							}
							if pred.ExcessiveHours > 0.5 {
								violationCount++
							}
							if pred.SaturdayOverload > 0.5 {
								violationCount++
							}
						}
						if violationCount > 0 {
							penalty := float64(violationCount) * annConstraintPenalty
							postCrossoverPenalties[i] = penalty
							genStats.ConstraintPenaltyTotal += penalty
							penaltyCount++
						}
					}
				}
			}
			if checkedCount > 0 {
				fmt.Printf(
					"ga: [constraint screen] - took %s, ann calls=%d checked=%d penalised=%d total_penalty=%.2f\n",
					time.Since(start),
					genStats.ConstraintRequestCount,
					checkedCount,
					penaltyCount,
					genStats.ConstraintPenaltyTotal,
				)
			}
		}

		// ── STEP 6: MUTATION ──────────────────────────────────────────────────
		//
		// FALLBACK: When annActive == false, the ANN mutation prediction block
		// at the bottom of the loop body is skipped.  All other behaviour —
		// the four mutation operators, structural validation, repair, revert
		// on failure — is unchanged.  Mutations that pass validation are kept.
		//
		// ADAPTIVE MUTATION: when the GA has stagnated, scale the mutation
		// rate up to encourage exploration.  Boost grows linearly with
		// stagnant_generations and is capped at maxMutationRate.  When fitness
		// improves, stagnantGenerations resets to 0 (downstream in the fitness
		// evaluation block), so the mutation rate naturally returns to base.
		start = time.Now()

		currentMutationRate := mutationRate
		if adaptiveMutation && stagnantGenerations > 0 {
			boost := 1.0 + 0.2*float64(stagnantGenerations)
			currentMutationRate = mutationRate * boost
			if currentMutationRate > maxMutationRate {
				currentMutationRate = maxMutationRate
			}
		}

		mutationsAttempted := 0
		hillClimbReverts := 0
		annMutationRequests := make([]MutationRequest, 0)

		for i := eliteCount; i < len(population); i++ {
			prev_uni_sched := make(Schedule.UniTimeTables, len(population[i].UniSched))
			if copied := copy(prev_uni_sched, population[i].UniSched); copied != len(population[i].UniSched) {
				log.Panicf("error copying university schedule before random mutation index [%d]: copied %d of %d",
					i, copied, len(population[i].UniSched))
			}
			prev_encoding_resource, err_copy := population[i].Resources.MakeCopy()
			if err_copy != nil {
				log.Panic("error copying encoding resource in random mutation: ", err_copy)
			}

			if errs := population[i].UniSched.VerticalValidation(rooms); len(errs) > 0 {
				log.Printf("ga: vertical validation failed before mutation [individual %d] — skipping mutation\n%v", i, errs)
				continue
			}
			if errs := HorizontalValidation(population[i].UniSched, curriculums, department_to_encode, selected_semester); len(errs) > 0 {
				log.Printf("ga: horizontal validation failed before mutation [individual %d] — skipping mutation\n%v", i, errs)
				continue
			}

			preMutationFitness := classicFitnessOf(population[i].UniSched, department_to_encode)

			// Probabilistic mutation roll using the adaptive rate.
			if rng.Float64() >= currentMutationRate {
				continue
			}
			mutationsAttempted++

			// POINT B — per-operator mutation sampling. Snapshot each section's state
			// just before each operator runs so the resulting (before, after, delta)
			// triple is correctly attributed to a single operator from the spec set.
			// Rows where the operator left the section unchanged (byte-identical
			// before/after) are skipped — those carry no learning signal and would
			// otherwise dominate the dataset 9:1.
			type beforeState struct {
				slice   [][][]int
				fitness float64
			}

			snapshot := func() map[int]beforeState {
				states := make(map[int]beforeState)
				IterateSectionsWeekSchedule(population[i].UniSched, curriculums, selected_semester, nil, nil,
					func(indices IterIndices, values IterValues) IterReturnType {
						if values.WeekSched == nil || values.Curriculum == nil {
							return IterProceed
						}
						if !department_to_encode[values.Curriculum.DepartmentID] {
							return IterProceed
						}
						section_subject_async_hours := buildSubjectIDToAsyncHoursMapFromSubjects(values.Semester.Subjects)
						states[indices.Usi] = beforeState{
							slice:   weekTimeTableToSlice(*values.WeekSched),
							fitness: MeasureWeekTimeTableBasicFitness(*values.WeekSched, section_subject_async_hours),
						}
						return IterProceed
					},
				)
				return states
			}

			type operatorApplication struct {
				name string
				fn   func()
			}
			operators := []operatorApplication{
				{"day_swap_timeslots", func() {
					ApplyRandomDaySwapTimeSlots(population[i].UniSched, population[i].Resources, curriculums, department_id, selected_semester)
				}},
				{"subject_day_swap", func() {
					ApplyRandomSubjectDaySwap(population[i].UniSched, population[i].Resources, curriculums, department_id, selected_semester)
				}},
				{"slot_nudge", func() {
					ApplyRandomSubjectTimeSlotNudge(population[i].UniSched, population[i].Resources, curriculums, department_id, selected_semester)
				}},
				{"slot_day_nudge", func() {
					ApplyRandomSubjectTimeSlotAndDayNudge(population[i].UniSched, population[i].Resources, curriculums, department_id, selected_semester)
				}},
			}

			needMutationSnapshots := dataCollector.IsEnabled() || (annActive && annMutationEnabled)
			mutationReverted := false
			mutationRevertConfidence := 0.0
			mutationRevertOperator := ""
		mutationOperatorLoop:
			for _, op := range operators {
				var beforeStates map[int]beforeState
				if needMutationSnapshots {
					beforeStates = snapshot()
				}

				op.fn()

				if beforeStates != nil {
					IterateSectionsWeekSchedule(population[i].UniSched, curriculums, selected_semester, nil, nil,
						func(indices IterIndices, values IterValues) IterReturnType {
							if values.WeekSched == nil || values.Curriculum == nil {
								return IterProceed
							}
							if !department_to_encode[values.Curriculum.DepartmentID] {
								return IterProceed
							}
							before, exists := beforeStates[indices.Usi]
							if !exists {
								return IterProceed
							}
							afterSlice := weekTimeTableToSlice(*values.WeekSched)
							if scheduleSlicesEqual(before.slice, afterSlice) {
								return IterProceed
							}
							section_subject_async_hours := buildSubjectIDToAsyncHoursMapFromSubjects(values.Semester.Subjects)
							afterFitness := MeasureWeekTimeTableBasicFitness(*values.WeekSched, section_subject_async_hours)
							delta := afterFitness - before.fitness
							label := "neutral"
							if delta > 0.5 {
								label = "improve"
							} else if delta < -0.5 {
								label = "worsen"
							}
							if dataCollector.IsEnabled() {
								dataCollector.LogMutation(MutationSample{
									BeforeSchedule: before.slice,
									AfterSchedule:  afterSlice,
									MutationType:   op.name,
									BeforeFitness:  before.fitness,
									AfterFitness:   afterFitness,
									Delta:          delta,
									Label:          label,
									DepartmentID:   department_id,
									Generation:     g,
								})
							}
							if annActive && annMutationEnabled {
								annMutationRequests = append(annMutationRequests, MutationRequest{
									BeforeSchedule: ScheduleData{WeekSchedule: before.slice},
									AfterSchedule:  ScheduleData{WeekSchedule: afterSlice},
									MutationType:   op.name,
								})
							}
							return IterProceed
						},
					)
				}
				if mutationReverted {
					break mutationOperatorLoop
				}
			}
			if mutationReverted {
				population[i].UniSched = prev_uni_sched
				population[i].Resources = prev_encoding_resource
				// Revert the constraint penalty too — the individual is now back
				// to its pre-mutation state.
				if postCrossoverPenalties != nil {
					postCrossoverPenalties[i] = 0
				}
				genStats.MutationRevertCount++
				if os.Getenv("LOG_MODE") == "verbose" {
					log.Printf(
						"ga: [mutation revert] gen %d individual %d operator %s (confidence=%.2f)",
						g, i, mutationRevertOperator, mutationRevertConfidence,
					)
				}
				continue
			}

			// POINT B' — Constraint sample collection from the MUTATED schedule
			// (evaluated before structural validation may revert it).  This is
			// the dataset's primary source of positive examples for hard-conflict
			// labels: mutations occasionally produce instructor/room/curriculum
			// violations that the revert below would otherwise hide.
			if dataCollector.IsEnabled() {
				hardViolations := detectHardViolations(
					population[i].UniSched, curriculums, rooms,
					department_to_encode, selected_semester,
				)
				crossStats := newCrossSectionStats(
					population[i].UniSched, curriculums,
					department_to_encode, selected_semester,
				)
				IterateSectionsWeekSchedule(population[i].UniSched, curriculums, selected_semester, nil, nil,
					func(indices IterIndices, values IterValues) IterReturnType {
						if values.WeekSched == nil || values.Curriculum == nil {
							return IterProceed
						}
						if !department_to_encode[values.Curriculum.DepartmentID] {
							return IterProceed
						}
						soft := detectSoftViolationsForSection(*values.WeekSched)
						dataCollector.LogConstraint(ConstraintSample{
							SectionSchedule: weekTimeTableToSlice(*values.WeekSched),
							DepartmentID:    department_id,
							Violations:      mergeViolations(hardViolations, soft),
							CrossSection:    crossStats.computeAggregatesForSection(*values.WeekSched),
						})
						return IterProceed
					},
				)
			}

			if errs := population[i].UniSched.VerticalValidation(rooms); len(errs) > 0 {
				log.Printf("RunGeneticAlgorithm: invalid schedule after mutation index [%d]; reverting\n%v", i, errs)
				population[i].UniSched = prev_uni_sched
				population[i].Resources = prev_encoding_resource
				continue
			}
			if errs := HorizontalValidation(population[i].UniSched, curriculums, department_to_encode, selected_semester); len(errs) > 0 {
				log.Printf("RunGeneticAlgorithm: invalid schedule after mutation index [%d]; reverting\n%v", i, errs)
				population[i].UniSched = prev_uni_sched
				population[i].Resources = prev_encoding_resource
				continue
			}
			if errs := gaLoadConstraintErrors(population[i].UniSched); len(errs) > 0 {
				log.Printf("RunGeneticAlgorithm: load-constraint (units/4-day) violation after mutation index [%d]; reverting\n%v", i, errs)
				population[i].UniSched = prev_uni_sched
				population[i].Resources = prev_encoding_resource
				continue
			}

			re_encode_tries := 0
			for re_encode_tries < MAX_RE_ENCODE_REPAIR_TRIALS {
				generated_enc_res, err_gen := GenerateEncodingResourceFromUniTimeTable(
					population[i].UniSched, curriculums, selected_semester, default_empty_encoding_resource,
				)
				if err_gen != nil {
					return nil, nil, fmt.Errorf(
						"unable to generate encoding resource for repair at generation %d: %w", g, err_gen,
					)
				}

				// CRITICAL: re-apply the external-section skip to the freshly-
				// generated encoding resource.  GenerateEncodingResourceFromUniTimeTable
				// builds a clean resource with no IsSchedIdxToSubIdToSkip entries,
				// so without this line EncodeIndividualGenome below would try to
				// fill external sections (e.g. BSPsych under GA_SKIP_CURRICULA),
				// fail because those subjects are over-capacity, and leave the
				// section partially scheduled.  HorizontalValidation then catches
				// the missing subjects and reverts the mutation.  Repeated for
				// every individual every generation, that's a GA that runs to
				// completion without ever changing the population.
				applyExternalSectionsToEncodingResource(generated_enc_res, externalSectionSubjects)

				if !IsEqualEncodingResource(generated_enc_res, population[i].Resources) {
					if IsEqualEncodingResource(prev_encoding_resource, population[i].Resources) {
						log.Printf("RunGeneticAlgorithm: [Random Mutation][%d] prev == current encoding resource — ERROR", i)
					}
					log.Printf("RunGeneticAlgorithm: [Random Mutation][%d] generated != current — repairing", i)
					population[i].Resources = generated_enc_res
				}

				repaired_sched, repaired_enc_res, err_repair := EncodeIndividualGenome(
					population[i].UniSched, curriculums, dept_id_to_department,
					generated_enc_res, department_to_encode, selected_semester, 0,
				)
				if err_repair != nil {
					re_encode_tries++
					if re_encode_tries >= MAX_RE_ENCODE_REPAIR_TRIALS {
						return nil, nil, fmt.Errorf(
							"unable to repair individual at generation %d after %d tries: %w",
							g, MAX_RE_ENCODE_REPAIR_TRIALS, err_repair,
						)
					}
					population[i].UniSched = prev_uni_sched
					population[i].Resources = prev_encoding_resource
					ApplyRandomDaySwapTimeSlots(population[i].UniSched, population[i].Resources, curriculums, department_id, selected_semester)
					ApplyRandomSubjectDaySwap(population[i].UniSched, population[i].Resources, curriculums, department_id, selected_semester)
					ApplyRandomSubjectTimeSlotNudge(population[i].UniSched, population[i].Resources, curriculums, department_id, selected_semester)
					ApplyRandomSubjectTimeSlotAndDayNudge(population[i].UniSched, population[i].Resources, curriculums, department_id, selected_semester)
					continue
				}
				re_encode_tries = 0
				population[i].UniSched = repaired_sched
				population[i].Resources = repaired_enc_res

				if errs := population[i].UniSched.VerticalValidation(rooms); len(errs) > 0 {
					log.Printf("RunGeneticAlgorithm: repaired schedule still invalid at mutation index [%d]; reverting\n%v", i, errs)
					population[i].UniSched = prev_uni_sched
					population[i].Resources = prev_encoding_resource
					continue
				}
				if errs := HorizontalValidation(population[i].UniSched, curriculums, department_to_encode, selected_semester); len(errs) > 0 {
					log.Printf("RunGeneticAlgorithm: repaired schedule still horizontally invalid at mutation index [%d]; reverting\n%v", i, errs)
					population[i].UniSched = prev_uni_sched
					population[i].Resources = prev_encoding_resource
					continue
				}
				if errs := gaLoadConstraintErrors(population[i].UniSched); len(errs) > 0 {
					log.Printf("RunGeneticAlgorithm: repaired schedule violates load constraints (units/4-day) at mutation index [%d]; reverting\n%v", i, errs)
					population[i].UniSched = prev_uni_sched
					population[i].Resources = prev_encoding_resource
					continue
				}
				break
			}

			// ── HILL-CLIMBING GUARD ──────────────────────────────────────────
			//
			// Reject mutations that significantly worsen fitness.  The lenient
			// threshold (default 2.0) allows small drops, which is essential —
			// pure hill climbing always rejecting any drop would converge fast
			// to a local optimum and stay there.  The point is to prevent a
			// catastrophic mutation from undoing many generations of progress
			// in a single step, while still allowing the GA to occasionally
			// step downhill in pursuit of better regions.
			if hillClimbThreshold > 0 {
				postMutationFitness := classicFitnessOf(population[i].UniSched, department_to_encode)
				if preMutationFitness-postMutationFitness > hillClimbThreshold {
					if os.Getenv("LOG_MODE") == "verbose" {
						log.Printf("ga: [hill-climb revert] gen %d individual %d: %.4f -> %.4f (drop=%.4f > threshold=%.2f)",
							g, i, preMutationFitness, postMutationFitness,
							preMutationFitness-postMutationFitness, hillClimbThreshold)
					}
					population[i].UniSched = prev_uni_sched
					population[i].Resources = prev_encoding_resource
					if postCrossoverPenalties != nil {
						postCrossoverPenalties[i] = 0
					}
					hillClimbReverts++
					continue
				}
			}

			_ = preMutationFitness
		}

		mutationImproveCount, mutationNeutralCount, mutationWorsenCount := 0, 0, 0
		if annActive && annMutationEnabled && len(annMutationRequests) > 0 {
			genStats.MutationRequestCount++
			predictions, err := ann_client.BatchPredictMutation(annMutationRequests)
			if err != nil {
				genStats.MutationFailureCount++
				if os.Getenv("LOG_MODE") == "verbose" {
					log.Printf("ga: [mutation batch fallback] %s", err.Error())
				}
			} else {
				for _, pred := range predictions {
					switch pred.Label {
					case "improve":
						mutationImproveCount++
					case "worsen":
						mutationWorsenCount++
					default:
						mutationNeutralCount++
					}
				}
			}
		}

		if annActive {
			fmt.Printf("ga: [mutation] - took %s rate=%.2f attempted=%d hill_reverts=%d (ann calls: %d, improve=%d neutral=%d worsen=%d)\n",
				time.Since(start), currentMutationRate, mutationsAttempted, hillClimbReverts,
				genStats.MutationRequestCount, mutationImproveCount, mutationNeutralCount, mutationWorsenCount)
		} else {
			fmt.Printf("ga: [mutation] - took %s rate=%.2f attempted=%d hill_reverts=%d (classic, no ANN)\n",
				time.Since(start), currentMutationRate, mutationsAttempted, hillClimbReverts)
		}

		// ── STEP 3: FITNESS EVALUATION + CONSTRAINT PENALTY APPLICATION ───────
		//
		// FALLBACK: measurePopulationFitnessWithANNTopK detects the nil
		// ann_client and routes to measurePopulationFitnessClassic for the
		// whole population.  postCrossoverPenalties is nil in pure-GA mode,
		// so the penalty-application loop is also skipped.
		start = time.Now()

		populationSchedules := make([]Schedule.UniTimeTables, len(population))
		for i := range population {
			populationSchedules[i] = population[i].UniSched
		}

		populationFitness := measurePopulationFitnessWithANNTopK(
			ann_client, populationSchedules, curriculums,
			department_to_encode, selected_semester,
			genStats, classicFitnessCache, annFitnessCache, rng,
			annTopKRatio, annTopKMin,
		)

		// Apply constraint penalties only when ANN actually generated some.
		// In pure-GA mode postCrossoverPenalties is nil and this is a no-op.
		if annActive && postCrossoverPenalties != nil {
			penaltyApplied := 0
			for i, penalty := range postCrossoverPenalties {
				if penalty > 0 {
					populationFitness[i] -= penalty
					penaltyApplied++
				}
			}
			if penaltyApplied > 0 && os.Getenv("LOG_MODE") == "verbose" {
				log.Printf("ga: applied constraint penalties to %d/%d individuals",
					penaltyApplied, len(population))
			}
		}

		type schedWithFitness struct {
			Sched   SchedAndResources
			Fitness float64
		}
		rankedPopulation := make([]schedWithFitness, len(population))
		for i := range population {
			rankedPopulation[i] = schedWithFitness{Sched: population[i], Fitness: populationFitness[i]}
		}
		sort.Slice(rankedPopulation, func(i, j int) bool {
			return rankedPopulation[i].Fitness > rankedPopulation[j].Fitness
		})
		for i := range rankedPopulation {
			population[i] = rankedPopulation[i].Sched
		}

		genesis_population = population
		fittest_individual_fitness := rankedPopulation[0].Fitness

		if fittest_individual_fitness > bestFitnessSeen+earlyStopMinImprovement {
			bestFitnessSeen = fittest_individual_fitness
			stagnantGenerations = 0
		} else {
			stagnantGenerations++
		}

		if annActive {
			fmt.Printf("ga: [fitness evaluation] - took %s (ANN+classic top-K)\n", time.Since(start))
		} else {
			fmt.Printf("ga: [fitness evaluation] - took %s (classic)\n", time.Since(start))
		}

		if cb_fn_generation != nil {
			cb_fn_generation(g, genesis_population[0].UniSched, fittest_individual_fitness)
		}

		// per-generation summary (best / avg / worst / std-dev / unique).
		// std-dev and unique-fitness count are diversity proxies — when both
		// drop close to zero the population has converged and the GA cannot
		// improve further without injection or higher mutation.
		sumFitness := 0.0
		worstFitness := math.Inf(1)
		uniqueFitness := make(map[float64]bool, len(populationFitness))
		for _, f := range populationFitness {
			sumFitness += f
			if f < worstFitness {
				worstFitness = f
			}
			// Round to 4 decimals so floating noise doesn't inflate the count.
			uniqueFitness[math.Round(f*10000)/10000] = true
		}
		avgFitness := sumFitness / float64(len(populationFitness))
		variance := 0.0
		for _, f := range populationFitness {
			diff := f - avgFitness
			variance += diff * diff
		}
		variance /= float64(len(populationFitness))
		stdDev := math.Sqrt(variance)
		log.Printf("ga: gen %d best=%.4f avg=%.4f worst=%.4f stddev=%.4f unique=%d/%d stagnant=%d",
			g, fittest_individual_fitness, avgFitness, worstFitness, stdDev,
			len(uniqueFitness), len(populationFitness), stagnantGenerations)

		// ── PURGE EMPTY SCHEDULES ─────────────────────────────────────────────
		// Replace any individual whose schedule is empty with a fresh genesis-
		// style individual.  Empty schedules accumulate at the bottom of the
		// ranked population because:
		//   • They can't win tournaments (-24 always loses), so they don't
		//     get selected as parents
		//   • But they survive sorts because nothing replaces them
		//   • So they pile up over generations, polluting the average and
		//     reducing the effective population size that's actually working
		//     on the problem
		// This purge runs every generation, regardless of stagnation status.
		if purgeEmpties {
			purgeStartTime := time.Now()
			purgedCount := 0
			for idx := eliteCount; idx < len(genesis_population); idx++ {
				if !genesis_population[idx].UniSched.IsEmpty() {
					continue
				}
				fresh, err := buildFreshIndividual(
					base_uni_sched, curriculums, dept_id_to_department,
					default_empty_encoding_resource, externalSectionSubjects,
					department_to_encode, department_id, selected_semester,
				)
				if err != nil {
					log.Printf("ga: [purge empties] gen %d index %d failed: %s", g, idx, err)
					continue
				}
				genesis_population[idx] = *fresh
				purgedCount++
			}
			if purgedCount > 0 {
				log.Printf("ga: [purge empties] gen %d — replaced %d empty individual(s) (took %s)",
					g, purgedCount, time.Since(purgeStartTime))
			}
		}

		// ── STAGNATION INJECTION ──────────────────────────────────────────────
		// When the GA hasn't improved for `stagnationInjectThreshold` generations,
		// the population has likely collapsed onto a local optimum.  Replace the
		// worst `stagnationInjectFraction` of the population with freshly-built
		// random individuals.  This brings new genetic material that crossover
		// and mutation alone cannot synthesise — the only reliable way to escape
		// a converged local optimum.
		//
		// The base schedule (index 0) and the elite individuals are preserved
		// regardless; injection always targets the bottom of the ranking.
		// stagnantGenerations is reset on next-gen improvement, not here, so
		// repeated injection won't fire every generation if the new individuals
		// happen not to beat the elite immediately.
		if stagnationInjectThreshold > 0 &&
			stagnantGenerations >= stagnationInjectThreshold &&
			stagnationInjectFraction > 0 {
			injectCount := int(math.Round(float64(len(genesis_population)) * stagnationInjectFraction))
			if injectCount < 1 {
				injectCount = 1
			}
			// Don't replace elites or the base schedule.
			injectStartIdx := len(genesis_population) - injectCount
			if injectStartIdx <= eliteCount {
				injectStartIdx = eliteCount + 1
			}
			actualInjected := 0
			injectStartTime := time.Now()
			for idx := injectStartIdx; idx < len(genesis_population); idx++ {
				fresh, err := buildFreshIndividual(
					base_uni_sched, curriculums, dept_id_to_department,
					default_empty_encoding_resource, externalSectionSubjects,
					department_to_encode, department_id, selected_semester,
				)
				if err != nil {
					log.Printf("ga: [stagnation inject] fresh individual %d failed: %s — keeping current", idx, err)
					continue
				}
				genesis_population[idx] = *fresh
				actualInjected++
			}
			totalInjectionsPerformed++
			log.Printf("ga: [stagnation inject] gen %d — replaced %d of %d worst individuals (took %s, total injections=%d)",
				g, actualInjected, injectCount, time.Since(injectStartTime), totalInjectionsPerformed)
		}

		// ANN telemetry only printed when ANN is actually engaged this run.
		if annActive && os.Getenv("LOG_MODE") == "verbose" {
			cacheTotal := genStats.CacheHitCount + genStats.CacheMissCount
			cacheHitRate := 0.0
			if cacheTotal > 0 {
				cacheHitRate = float64(genStats.CacheHitCount) / float64(cacheTotal)
			}
			log.Printf(
				"ga: gen %d ann — fitness[pred=%d req=%d fail=%d fallback=%d used=%d] cache[hit=%d miss=%d rate=%.2f] crossover[req=%d fail=%d] constraint[req=%d penalty=%.2f] mutation[req=%d revert=%d]",
				g,
				genStats.ANNPredictionCount, genStats.ANNRequestCount, genStats.ANNFailureCount, genStats.ANNFallbackCount, genStats.ANNUsedEvalCount,
				genStats.CacheHitCount, genStats.CacheMissCount, cacheHitRate,
				genStats.CrossoverRequestCount, genStats.CrossoverFailureCount,
				genStats.ConstraintRequestCount, genStats.ConstraintPenaltyTotal,
				genStats.MutationRequestCount, genStats.MutationRevertCount,
			)
		}

		if earlyStopPatience > 0 && (g+1) >= earlyStopMinGen && stagnantGenerations >= earlyStopPatience {
			log.Printf("ga: early stop at generation %d (patience=%d, stagnant=%d)",
				g+1, earlyStopPatience, stagnantGenerations)
			break
		}

		if os.Getenv("LOG_MODE") == "verbose" {
			estimated := float64(genesis_population[0].Resources.EstimateMemoryUsageInBytes())
			log.Printf("Estimated encoding resource memory [gen %d]: %.2fMB", g, (estimated/1000.0)/1000.0)
		}

		dataCollector.PrintStats()
	}

	////////////////////////////////////////////////////////////////////////////////////////
	// STEP 7 — FINAL VALIDATION BEFORE RETURNING
	////////////////////////////////////////////////////////////////////////////////////////

	log.Printf("ga: fittest individual fitness: %f",
		classicFitnessOf(genesis_population[0].UniSched, department_to_encode))

	if genesis_population[0].UniSched.IsEmpty() {
		return nil, nil, errors.New("fittest university schedule is empty")
	}

	if errs := genesis_population[0].UniSched.VerticalValidation(rooms); len(errs) > 0 {
		log.Printf("GA-ERROR: vertical overlaps in fittest schedule:\n%v", errs)
		return nil, nil, errors.New("fittest university schedule has vertical overlaps")
	}
	for k, v := range department_to_encode {
		if v {
			log.Printf("ga: [passed] final vertical validation for %s %s",
				dept_id_to_department[k].Code, Curriculum.SEMESTER_INDEX_NAME[selected_semester])
		}
	}

	if errs := HorizontalValidation(genesis_population[0].UniSched, curriculums, department_to_encode, selected_semester); len(errs) > 0 {
		log.Printf("GA-ERROR: horizontal violations in fittest schedule:\n%v", errs)
		return nil, nil, errors.New("fittest university schedule has horizontal violations")
	}

	if errs := gaLoadConstraintErrors(genesis_population[0].UniSched); len(errs) > 0 {
		log.Printf("GA-ERROR: load-constraint (units/4-day) violations in fittest schedule:\n%v", errs)
		return nil, nil, errors.New("fittest university schedule violates instructor unit cap or 4-day packing constraint")
	}

	if err := ValidateNoUnexpectedEmptySections(
		genesis_population[0].UniSched,
		curriculums,
		department_to_encode,
		selected_semester,
		externalSectionSubjects,
	); err != nil {
		log.Printf("GA-ERROR: %s", err.Error())
		return nil, nil, err
	}
	for k, v := range department_to_encode {
		if v {
			log.Printf("ga: [passed] final horizontal validation for %s %s",
				dept_id_to_department[k].Code, Curriculum.SEMESTER_INDEX_NAME[selected_semester])
		}
	}

	if os.Getenv("LOG_MODE") == "verbose" {
		finalQuality := measureFinalScheduleQuality(
			genesis_population[0].UniSched,
			curriculums,
			rooms,
			department_to_encode,
			selected_semester,
		)
		header := "Final schedule quality (GA-only)"
		if annActive {
			header = "Final schedule quality (ANN-assisted GA)"
		}
		log.Printf("%s", header)
		log.Printf("  Excellent sections : %d (%.1f%%)", finalQuality.ExcellentSections, percentage(finalQuality.ExcellentSections, finalQuality.TotalSections))
		log.Printf("  Good sections      : %d (%.1f%%)", finalQuality.GoodSections, percentage(finalQuality.GoodSections, finalQuality.TotalSections))
		log.Printf("  Fair sections      : %d (%.1f%%)", finalQuality.FairSections, percentage(finalQuality.FairSections, finalQuality.TotalSections))
		log.Printf("  Poor sections      : %d (%.1f%%)", finalQuality.PoorSections, percentage(finalQuality.PoorSections, finalQuality.TotalSections))
		log.Printf("  Total violations   : %d", finalQuality.TotalViolations)
		log.Printf("  Overall quality    : %.1f/100", finalQuality.OverallQuality)
		if len(finalQuality.Sections) > 0 {
			log.Printf("  Violating sections:")
			for _, section := range finalQuality.Sections {
				log.Printf("    - %s [%s]: %s", section.SectionLabel, section.Bucket, strings.Join(section.Violations, ", "))
			}
		} else {
			log.Printf("  Violating sections: none")
		}
	}

	return genesis_population[0].UniSched, genesis_population[0].Resources, nil
}
