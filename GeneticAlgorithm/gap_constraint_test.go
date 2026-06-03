package GeneticAlgorithm

import (
	"testing"

	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

// block describes a subject placement used to build a test day:
// subject `id` occupying `count` slots starting at slot `start`.
type block struct {
	id    uint16
	start int
	count int
}

// buildDay constructs a DayTimeTable from a list of subject blocks. Slot 0 is
// 7:00 AM and each slot is 30 minutes.
func buildDay(blocks ...block) Schedule.DayTimeTable {
	var day Schedule.DayTimeTable
	for _, b := range blocks {
		for s := b.start; s < b.start+b.count; s++ {
			day[s].SetSubjectID(b.id)
		}
	}
	return day
}

func TestExtractSubjectBlocks(t *testing.T) {
	// A: 7:00-9:00 (slots 0-3), gap (slots 4-5), B: 10:00-12:00 (slots 6-9)
	day := buildDay(block{1, 0, 4}, block{2, 6, 4})

	blocks := ExtractSubjectBlocks(day)
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0] != (SubjectBlock{SubjectID: 1, StartSlot: 0, EndSlot: 4, SlotCount: 4}) {
		t.Errorf("block 0 mismatch: %+v", blocks[0])
	}
	if blocks[1] != (SubjectBlock{SubjectID: 2, StartSlot: 6, EndSlot: 10, SlotCount: 4}) {
		t.Errorf("block 1 mismatch: %+v", blocks[1])
	}
}

func TestExtractSubjectBlocksAdjacentSameSubjectMerged(t *testing.T) {
	// two contiguous runs of the same subject id form a single block.
	day := buildDay(block{7, 2, 3}, block{7, 5, 2})
	blocks := ExtractSubjectBlocks(day)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 merged block, got %d", len(blocks))
	}
	if blocks[0].SlotCount != 5 || blocks[0].StartSlot != 2 || blocks[0].EndSlot != 7 {
		t.Errorf("merged block mismatch: %+v", blocks[0])
	}
}

func TestCheckGapsBetweenSubjects(t *testing.T) {
	cases := []struct {
		name       string
		day        Schedule.DayTimeTable
		violations int
	}{
		{
			// VALID: A slots 0-3, 2-slot (1 hr) gap, B slots 6-9
			name:       "valid 1 hour gap",
			day:        buildDay(block{1, 0, 4}, block{2, 6, 4}),
			violations: 0,
		},
		{
			// VALID: A slots 0-3, 4-slot (2 hr) gap, B slots 8-11
			name:       "valid 2 hour gap",
			day:        buildDay(block{1, 0, 4}, block{2, 8, 4}),
			violations: 0,
		},
		{
			// INVALID: A slots 0-3, B slots 4-7 — no gap
			name:       "no gap too small",
			day:        buildDay(block{1, 0, 4}, block{2, 4, 4}),
			violations: 1,
		},
		{
			// INVALID: A slots 0-3, 6-slot (3 hr) gap, B slots 10-13
			name:       "gap too large",
			day:        buildDay(block{1, 0, 4}, block{2, 10, 4}),
			violations: 1,
		},
		{
			// single-subject day is exempt
			name:       "single subject exempt",
			day:        buildDay(block{1, 0, 4}),
			violations: 0,
		},
		{
			// empty day
			name:       "empty day",
			day:        buildDay(),
			violations: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckGapsBetweenSubjects(tc.day, MIN_GAP_SLOTS, MAX_GAP_SLOTS)
			if len(got) != tc.violations {
				t.Errorf("expected %d violation(s), got %d: %v", tc.violations, len(got), got)
			}
		})
	}
}

func TestCountGapViolations(t *testing.T) {
	var week Schedule.WeekTimeTable
	week[0] = buildDay(block{1, 0, 4}, block{2, 4, 4})  // 1 too-small gap
	week[1] = buildDay(block{1, 0, 4}, block{2, 10, 4}) // 1 too-large gap
	week[2] = buildDay(block{1, 0, 4}, block{2, 6, 4})  // valid
	week[3] = buildDay(block{1, 0, 4})                  // single subject

	if got := CountGapViolations(week, MIN_GAP_SLOTS, MAX_GAP_SLOTS); got != 2 {
		t.Errorf("expected 2 total violations, got %d", got)
	}
}

func TestHasMinimumGapFromPreviousSubject(t *testing.T) {
	// existing subject at slots 0-3; a proposed block must keep >= 2 slots away.
	day := buildDay(block{1, 0, 4})

	// proposed block at slot 4 (count 2) -> 0-slot gap -> too small
	if hasMinimumGapFromPreviousSubject(day, 4, 2, MIN_GAP_SLOTS) {
		t.Error("expected gap-too-small to be rejected for slot 4")
	}
	// proposed block at slot 6 (count 2) -> 2-slot gap -> acceptable
	if !hasMinimumGapFromPreviousSubject(day, 6, 2, MIN_GAP_SLOTS) {
		t.Error("expected 2-slot gap to be accepted for slot 6")
	}

	// existing subject AFTER the proposed slot: proposed 0-1, existing 2-5
	dayAfter := buildDay(block{9, 2, 4})
	if hasMinimumGapFromPreviousSubject(dayAfter, 0, 2, MIN_GAP_SLOTS) {
		t.Error("expected forward gap-too-small to be rejected")
	}
}

func TestGapConfigEnabledDisable(t *testing.T) {
	// GA_MIN_GAP_HOURS=0 must disable the constraint (backward compatible).
	t.Setenv("GA_MIN_GAP_HOURS", "0")
	cfg := LoadGapConfigFromEnv()
	if cfg.Enabled {
		t.Error("expected gap constraint to be disabled when GA_MIN_GAP_HOURS=0")
	}
	if gapShouldApplyToDay(0) {
		t.Error("gapShouldApplyToDay should be false when disabled")
	}

	// restore defaults for any subsequent code paths
	gapConfig = defaultGapConfig()
}

func TestLoadGapConfigFromEnvDefaults(t *testing.T) {
	t.Setenv("GA_MIN_GAP_HOURS", "1.0")
	t.Setenv("GA_MAX_GAP_HOURS", "2.0")
	cfg := LoadGapConfigFromEnv()

	if !cfg.Enabled {
		t.Fatal("expected enabled with min gap 1.0 hour")
	}
	if cfg.MinGapSlots != 2 {
		t.Errorf("expected MinGapSlots=2, got %d", cfg.MinGapSlots)
	}
	if cfg.MaxGapSlots != 4 {
		t.Errorf("expected MaxGapSlots=4, got %d", cfg.MaxGapSlots)
	}
	if cfg.ApplyOnSaturday {
		t.Error("expected ApplyOnSaturday to default to false")
	}

	gapConfig = defaultGapConfig()
}
