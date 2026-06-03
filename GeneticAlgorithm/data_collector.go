package GeneticAlgorithm

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Rooms"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

// ─────────────────────────────────────────────────────────────────────────────
//  DATA STRUCTURES — one schema per ANN training target
// ─────────────────────────────────────────────────────────────────────────────

// MutationSample is one training row for the Mutation Predictor model.
// before/after schedules are [6][24][3] = [day][time_slot][subject,instructor,room].
type MutationSample struct {
	BeforeSchedule [][][]int `json:"before_schedule"`
	AfterSchedule  [][][]int `json:"after_schedule"`
	MutationType   string    `json:"mutation_type"`
	BeforeFitness  float64   `json:"before_fitness"`
	AfterFitness   float64   `json:"after_fitness"`
	Delta          float64   `json:"delta"`
	Label          string    `json:"label"`
	DepartmentID   uint16    `json:"department_id"`
	Generation     int       `json:"generation"`
}

// ConstraintLabels enumerates the multi-label constraint violations the
// Constraint Classifier learns to predict.
type ConstraintLabels struct {
	InstructorConflict     bool `json:"instructor_conflict"`
	RoomConflict           bool `json:"room_conflict"`
	NoLunchBreak           bool `json:"no_lunch_break"`
	LateClasses            bool `json:"late_classes"`
	ExcessiveHours         bool `json:"excessive_hours"`
	SaturdayOverload       bool `json:"saturday_overload"`
	ResourceUnavailable    bool `json:"resource_unavailable"`
	CurriculumConflict     bool `json:"curriculum_conflict"`
	RoomCapacity           bool `json:"room_capacity"`
	InstructorAvailability bool `json:"instructor_availability"`
}

// CrossSectionAggregates summarises instructor/room collisions between the
// target section's occupied slots and every other section's occupied slots
// at the same (day, slot).  Computed once per individual and attached to
// every section sample so the Python feature extractor doesn't need the
// full university schedule serialised in each row.
type CrossSectionAggregates struct {
	TotalInstructorConflicts        int `json:"total_instructor_conflicts"`
	TotalRoomConflicts              int `json:"total_room_conflicts"`
	MaxInstructorConflictsInOneSlot int `json:"max_instructor_conflicts_in_one_slot"`
	MaxRoomConflictsInOneSlot       int `json:"max_room_conflicts_in_one_slot"`
	SlotsWithAnyConflict            int `json:"slots_with_any_conflict"`
}

// ConstraintSample is one training row for the Constraint Classifier.
type ConstraintSample struct {
	SectionSchedule [][][]int              `json:"section_schedule"`
	DepartmentID    uint16                 `json:"department_id"`
	Violations      ConstraintLabels       `json:"violations"`
	CrossSection    CrossSectionAggregates `json:"cross_section"`
}

// CrossoverSample is one training row for the Crossover Recommender.
// Each sample captures a (parent1, parent2) attempt and the offspring outcome.
type CrossoverSample struct {
	Parent1          [][][]int `json:"parent1"`
	Parent2          [][][]int `json:"parent2"`
	ProducedValid    int       `json:"produced_valid"`
	OffspringFitness float64   `json:"offspring_fitness"`
	Parent1Fitness   float64   `json:"parent1_fitness"`
	Parent2Fitness   float64   `json:"parent2_fitness"`
	DepartmentID     uint16    `json:"department_id"`
	Generation       int       `json:"generation"`
}

// ─────────────────────────────────────────────────────────────────────────────
//  DATA COLLECTOR
// ─────────────────────────────────────────────────────────────────────────────

type DataCollector struct {
	enabled        bool
	outputDir      string
	mutationFile   *os.File
	constraintFile *os.File
	crossoverFile  *os.File
	mutationEnc    *json.Encoder
	constraintEnc  *json.Encoder
	crossoverEnc   *json.Encoder
	mu             sync.Mutex

	MutationCount   int
	ConstraintCount int
	CrossoverCount  int
}

// NewDataCollector returns a collector writing JSONL files into outputDir.
// An empty outputDir yields a disabled, zero-cost collector — every Log* call
// becomes a no-op.
func NewDataCollector(outputDir string) (*DataCollector, error) {
	if outputDir == "" {
		return &DataCollector{enabled: false}, nil
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("data_collector: cannot create output dir %q: %w", outputDir, err)
	}

	openAppend := func(name string) (*os.File, error) {
		return os.OpenFile(
			filepath.Join(outputDir, name),
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0o644,
		)
	}

	mFile, err := openAppend("mutation_samples.jsonl")
	if err != nil {
		return nil, fmt.Errorf("data_collector: cannot open mutation file: %w", err)
	}
	cFile, err := openAppend("constraint_samples.jsonl")
	if err != nil {
		_ = mFile.Close()
		return nil, fmt.Errorf("data_collector: cannot open constraint file: %w", err)
	}
	xFile, err := openAppend("crossover_samples.jsonl")
	if err != nil {
		_ = mFile.Close()
		_ = cFile.Close()
		return nil, fmt.Errorf("data_collector: cannot open crossover file: %w", err)
	}

	dc := &DataCollector{
		enabled:        true,
		outputDir:      outputDir,
		mutationFile:   mFile,
		constraintFile: cFile,
		crossoverFile:  xFile,
		mutationEnc:    json.NewEncoder(mFile),
		constraintEnc:  json.NewEncoder(cFile),
		crossoverEnc:   json.NewEncoder(xFile),
	}

	log.Printf("data collection enabled → outputDir=%s", outputDir)
	return dc, nil
}

func (dc *DataCollector) IsEnabled() bool {
	return dc != nil && dc.enabled
}

func (dc *DataCollector) LogMutation(s MutationSample) {
	if !dc.IsEnabled() {
		return
	}
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if err := dc.mutationEnc.Encode(s); err != nil {
		log.Printf("data_collector: failed to encode mutation sample: %s", err)
		return
	}
	dc.MutationCount++
}

func (dc *DataCollector) LogConstraint(s ConstraintSample) {
	if !dc.IsEnabled() {
		return
	}
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if err := dc.constraintEnc.Encode(s); err != nil {
		log.Printf("data_collector: failed to encode constraint sample: %s", err)
		return
	}
	dc.ConstraintCount++
}

func (dc *DataCollector) LogCrossover(s CrossoverSample) {
	if !dc.IsEnabled() {
		return
	}
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if err := dc.crossoverEnc.Encode(s); err != nil {
		log.Printf("data_collector: failed to encode crossover sample: %s", err)
		return
	}
	dc.CrossoverCount++
}

func (dc *DataCollector) Close() {
	if !dc.IsEnabled() {
		return
	}
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.mutationFile != nil {
		if err := dc.mutationFile.Close(); err != nil {
			log.Printf("data_collector: error closing mutation file: %s", err)
		}
	}
	if dc.constraintFile != nil {
		if err := dc.constraintFile.Close(); err != nil {
			log.Printf("data_collector: error closing constraint file: %s", err)
		}
	}
	if dc.crossoverFile != nil {
		if err := dc.crossoverFile.Close(); err != nil {
			log.Printf("data_collector: error closing crossover file: %s", err)
		}
	}
	dc.enabled = false

	log.Printf("data collection complete: mutations=%d constraints=%d crossovers=%d",
		dc.MutationCount, dc.ConstraintCount, dc.CrossoverCount)
}

func (dc *DataCollector) PrintStats() {
	if !dc.IsEnabled() {
		return
	}
	dc.mu.Lock()
	m, c, x := dc.MutationCount, dc.ConstraintCount, dc.CrossoverCount
	dc.mu.Unlock()
	log.Printf("data collection counts: mutations=%d constraints=%d crossovers=%d", m, c, x)
}

// ─────────────────────────────────────────────────────────────────────────────
//  HELPERS
// ─────────────────────────────────────────────────────────────────────────────

// weekTimeTableToSlice flattens a WeekTimeTable into a [6][24][3] int slice:
// each slot is [subject_id, instructor_id, room_id].
func weekTimeTableToSlice(week Schedule.WeekTimeTable) [][][]int {
	out := make([][][]int, Const.N_WEEKLY_SCHOOL_DAYS)
	for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
		out[day] = make([][]int, Const.N_DAILY_TIME_SLOTS)
		for slot := 0; slot < Const.N_DAILY_TIME_SLOTS; slot++ {
			ts := week[day][slot]
			out[day][slot] = []int{
				int(ts.GetSubjectID()),
				int(ts.GetInstructorID()),
				int(ts.GetRoomID()),
			}
		}
	}
	return out
}

// scheduleSlicesEqual returns true when two [6][24][3] schedule slices
// match cell-for-cell. Used by the mutation logger to drop no-op rows.
func scheduleSlicesEqual(a, b [][][]int) bool {
	if len(a) != len(b) {
		return false
	}
	for d := range a {
		if len(a[d]) != len(b[d]) {
			return false
		}
		for s := range a[d] {
			if len(a[d][s]) != len(b[d][s]) {
				return false
			}
			for k := range a[d][s] {
				if a[d][s][k] != b[d][s][k] {
					return false
				}
			}
		}
	}
	return true
}

// crossSectionStats walks every section in `sched` once and returns, for
// each (day, slot), the multiset of instructor ids and room ids in use.
// Used by computeAggregatesForSection.
type crossSectionStats struct {
	instructorAt [Const.N_WEEKLY_SCHOOL_DAYS][Const.N_DAILY_TIME_SLOTS]map[uint16]int
	roomAt       [Const.N_WEEKLY_SCHOOL_DAYS][Const.N_DAILY_TIME_SLOTS]map[uint16]int
}

func newCrossSectionStats(
	sched Schedule.UniTimeTables,
	curriculums []Curriculum.Curriculum,
	departmentToMeasure map[uint16]bool,
	selectedSemester int,
) *crossSectionStats {
	cs := &crossSectionStats{}
	for d := 0; d < Const.N_WEEKLY_SCHOOL_DAYS; d++ {
		for s := 0; s < Const.N_DAILY_TIME_SLOTS; s++ {
			cs.instructorAt[d][s] = map[uint16]int{}
			cs.roomAt[d][s] = map[uint16]int{}
		}
	}
	IterateSectionsWeekSchedule(sched, curriculums, selectedSemester, nil, nil,
		func(indices IterIndices, values IterValues) IterReturnType {
			if values.WeekSched == nil || values.Curriculum == nil {
				return IterProceed
			}
			if len(departmentToMeasure) > 0 && !departmentToMeasure[values.Curriculum.DepartmentID] {
				return IterProceed
			}
			for d := 0; d < Const.N_WEEKLY_SCHOOL_DAYS; d++ {
				for s := 0; s < Const.N_DAILY_TIME_SLOTS; s++ {
					ts := (*values.WeekSched)[d][s]
					if ts.GetSubjectID() == 0 {
						continue
					}
					if iid := ts.GetInstructorID(); iid != 0 {
						cs.instructorAt[d][s][iid]++
					}
					if rid := ts.GetRoomID(); rid != 0 {
						cs.roomAt[d][s][rid]++
					}
				}
			}
			return IterProceed
		})
	return cs
}

// computeAggregatesForSection returns the 5 cross-section aggregates for one
// section against the pre-built per-slot multiset. Counts only other
// sections by subtracting the target's own occupancy from each count.
func (cs *crossSectionStats) computeAggregatesForSection(
	section Schedule.WeekTimeTable,
) CrossSectionAggregates {
	out := CrossSectionAggregates{}
	distinctConflictSlots := 0

	for d := 0; d < Const.N_WEEKLY_SCHOOL_DAYS; d++ {
		for s := 0; s < Const.N_DAILY_TIME_SLOTS; s++ {
			ts := section[d][s]
			if ts.GetSubjectID() == 0 {
				continue
			}
			iid := ts.GetInstructorID()
			rid := ts.GetRoomID()
			instConflicts := 0
			roomConflicts := 0
			if iid != 0 {
				if total := cs.instructorAt[d][s][iid]; total > 0 {
					instConflicts = total - 1
				}
			}
			if rid != 0 {
				if total := cs.roomAt[d][s][rid]; total > 0 {
					roomConflicts = total - 1
				}
			}
			out.TotalInstructorConflicts += instConflicts
			out.TotalRoomConflicts += roomConflicts
			if instConflicts > out.MaxInstructorConflictsInOneSlot {
				out.MaxInstructorConflictsInOneSlot = instConflicts
			}
			if roomConflicts > out.MaxRoomConflictsInOneSlot {
				out.MaxRoomConflictsInOneSlot = roomConflicts
			}
			if instConflicts > 0 || roomConflicts > 0 {
				distinctConflictSlots++
			}
		}
	}
	out.SlotsWithAnyConflict = distinctConflictSlots
	return out
}

// detectHardViolations runs the hard-constraint checks on the full university
// schedule and returns labels with only the hard fields populated.  Cheap to
// share across every section sample emitted from the same `sched`.
func detectHardViolations(
	sched Schedule.UniTimeTables,
	curriculums []Curriculum.Curriculum,
	rooms []Rooms.Room,
	departmentToMeasure map[uint16]bool,
	selectedSemester int,
) ConstraintLabels {
	labels := ConstraintLabels{}

	// VerticalValidation returns errors as fmt.Errorf-wrapped JSON strings
	// (see Schedule.UniInstructorValidationError / UniRoomValidationError).
	// Substring-match on the serialised payload to bucket the violations.
	for _, vErr := range sched.VerticalValidation(rooms) {
		msg := vErr.Error()
		if strings.Contains(msg, "InstructorID") || strings.Contains(msg, "instructor") {
			labels.InstructorConflict = true
		}
		if strings.Contains(msg, "RooomID") || strings.Contains(msg, "RoomID") || strings.Contains(msg, "room") {
			labels.RoomConflict = true
		}
	}

	if errs := HorizontalValidation(sched, curriculums, departmentToMeasure, selectedSemester); len(errs) > 0 {
		labels.CurriculumConflict = true
	}

	return labels
}

// detectSoftViolationsForSection evaluates the soft per-section constraints
// for a single section's week schedule.  Per-section semantics avoid the
// "always true" degeneracy that an OR-across-the-whole-university produces.
func detectSoftViolationsForSection(week Schedule.WeekTimeTable) ConstraintLabels {
	labels := ConstraintLabels{}
	for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
		occupiedSlots := 0
		hasLunchBreak := false
		hasLateClass := false

		for slot := 0; slot < Const.N_DAILY_TIME_SLOTS; slot++ {
			subjectID := week[day][slot].GetSubjectID()
			if subjectID != 0 {
				occupiedSlots++
				if slot >= 20 {
					hasLateClass = true
				}
			}
			if slot >= 8 && slot <= 11 && subjectID == 0 {
				hasLunchBreak = true
			}
		}

		if occupiedSlots == 0 {
			continue
		}
		if !hasLunchBreak {
			labels.NoLunchBreak = true
		}
		if hasLateClass {
			labels.LateClasses = true
		}
		if occupiedSlots > 20 {
			labels.ExcessiveHours = true
		}
		if day == Const.N_WEEKLY_SCHOOL_DAYS-1 && occupiedSlots > 10 {
			labels.SaturdayOverload = true
		}
	}
	return labels
}

// mergeViolations returns the bitwise OR of two label sets.  Used to combine
// schedule-wide hard flags with section-specific soft flags.
func mergeViolations(a, b ConstraintLabels) ConstraintLabels {
	return ConstraintLabels{
		InstructorConflict:     a.InstructorConflict || b.InstructorConflict,
		RoomConflict:           a.RoomConflict || b.RoomConflict,
		NoLunchBreak:           a.NoLunchBreak || b.NoLunchBreak,
		LateClasses:            a.LateClasses || b.LateClasses,
		ExcessiveHours:         a.ExcessiveHours || b.ExcessiveHours,
		SaturdayOverload:       a.SaturdayOverload || b.SaturdayOverload,
		ResourceUnavailable:    a.ResourceUnavailable || b.ResourceUnavailable,
		CurriculumConflict:     a.CurriculumConflict || b.CurriculumConflict,
		RoomCapacity:           a.RoomCapacity || b.RoomCapacity,
		InstructorAvailability: a.InstructorAvailability || b.InstructorAvailability,
	}
}

// detectConstraintViolations is kept for compatibility.  It composes the
// per-section soft checks across every in-department section (OR semantics)
// and merges them with the schedule-wide hard checks.
func detectConstraintViolations(
	sched Schedule.UniTimeTables,
	curriculums []Curriculum.Curriculum,
	rooms []Rooms.Room,
	departmentToMeasure map[uint16]bool,
	selectedSemester int,
) ConstraintLabels {
	labels := detectHardViolations(sched, curriculums, rooms, departmentToMeasure, selectedSemester)
	IterateSectionsWeekSchedule(sched, curriculums, selectedSemester, nil, nil,
		func(indices IterIndices, values IterValues) IterReturnType {
			if values.WeekSched == nil || values.Curriculum == nil {
				return IterProceed
			}
			if len(departmentToMeasure) > 0 && !departmentToMeasure[values.Curriculum.DepartmentID] {
				return IterProceed
			}
			labels = mergeViolations(labels, detectSoftViolationsForSection(*values.WeekSched))
			return IterProceed
		})
	return labels
}

