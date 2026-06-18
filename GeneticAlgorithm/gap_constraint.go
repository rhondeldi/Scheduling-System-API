// Inter-subject gap constraint.
//
// Requirement: between any two *different* subject blocks scheduled on the
// same day for the same section there must be a gap of 1 to 2 hours
// (2 to 4 time slots, since one slot = 30 minutes).
//
//	VALID   : A 7:00-9:00 | gap 2 slots | B 10:00-12:00   (1 hr gap)
//	VALID   : A 7:00-9:00 | gap 4 slots | B 11:00-13:00   (2 hr gap)
//	INVALID : A 7:00-9:00 | gap 0 slots | B 9:00-11:00    (no gap)
//	INVALID : A 7:00-9:00 | gap 6 slots | B 12:00-14:00   (gap too large)
//
// The lunch-break window (slots 8-11) counts as part of the gap, so a
// subject ending at slot 8 and another starting at slot 12 is a valid
// 2-hour gap.
//
// Design summary (see also the task spec):
//   - The MINIMUM gap is steered in encoding (EncodeIndividualGenome) as a
//     soft preference and penalised in fitness.
//   - The MAXIMUM gap is enforced in fitness ONLY — enforcing it in encoding
//     is too restrictive for dense schedules.
//   - The whole constraint is SOFT: it never makes the GA fail. Violations
//     only reduce the fitness score.
//   - Single-subject days are exempt (no gap to check).
//   - The gap between the last subject and the end of the day is NOT checked.
//   - Setting GA_MIN_GAP_HOURS=0 disables the constraint entirely, restoring
//     the previous behaviour exactly (backward compatible).
package GeneticAlgorithm

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

// ─── default gap-constraint parameters ──────────────────────────────────────
// These are the compiled-in defaults. They can each be overridden at runtime
// via the environment variables read by LoadGapConfigFromEnv (Part 5), so no
// recompilation is needed to tune the constraint.
const (
	MIN_GAP_SLOTS         int     = 2   // 1 hour minimum  (2 × 30-minute slots)
	MAX_GAP_SLOTS         int     = 4   // 2 hours maximum (4 × 30-minute slots)
	GAP_VIOLATION_PENALTY float64 = 3.0 // fitness penalty per gap violation
	GAP_SATISFIED_REWARD  float64 = 1.5 // fitness reward when a day's gaps are all valid
)

// GapConfig holds the runtime-tunable gap-constraint parameters. A single
// package-level instance (gapConfig) is consulted by the fitness function,
// the encoder and the horizontal validation. RunGeneticAlgorithm refreshes it
// from the environment via LoadGapConfigFromEnv before a run starts.
type GapConfig struct {
	MinGapSlots     int     // minimum gap in slots (steered in encoding, penalised in fitness)
	MaxGapSlots     int     // maximum gap in slots (penalised in fitness only)
	Penalty         float64 // fitness penalty per gap violation
	Reward          float64 // fitness reward when a multi-subject day has only valid gaps
	ApplyOnSaturday bool    // whether the gap constraint applies to Saturday (day index 5)
	Enabled         bool    // false when GA_MIN_GAP_HOURS <= 0 → constraint is a no-op
}

// defaultGapConfig returns the compiled-in defaults expressed as a GapConfig.
func defaultGapConfig() GapConfig {
	return GapConfig{
		MinGapSlots:     MIN_GAP_SLOTS,
		MaxGapSlots:     MAX_GAP_SLOTS,
		Penalty:         GAP_VIOLATION_PENALTY,
		Reward:          GAP_SATISFIED_REWARD,
		ApplyOnSaturday: false,
		Enabled:         true,
	}
}

// gapConfig is the package-wide active configuration. It defaults to the
// compiled-in values so unit tests and any code path that runs without first
// calling LoadGapConfigFromEnv still get sensible behaviour.
var gapConfig = defaultGapConfig()

// LoadGapConfigFromEnv builds a GapConfig from the GA_* environment variables,
// falling back to the compiled-in defaults when a variable is unset or
// malformed. It also installs the result as the package-wide gapConfig and
// returns it for logging.
//
//	GA_MIN_GAP_HOURS         (default 1.0)  minimum gap in hours; 0 disables the constraint
//	GA_MAX_GAP_HOURS         (default 2.0)  maximum gap in hours
//	GA_GAP_PENALTY           (default 3.0)  fitness penalty per violation
//	GA_GAP_REWARD            (default 1.5)  fitness reward when satisfied
//	GA_APPLY_GAP_ON_SATURDAY (default false) apply gap checks to Saturday too
func LoadGapConfigFromEnv() GapConfig {
	minHours := envFloat("GA_MIN_GAP_HOURS", 1.0)
	maxHours := envFloat("GA_MAX_GAP_HOURS", 2.0)

	cfg := GapConfig{
		MinGapSlots:     hoursToSlots(minHours),
		MaxGapSlots:     hoursToSlots(maxHours),
		Penalty:         envFloat("GA_GAP_PENALTY", GAP_VIOLATION_PENALTY),
		Reward:          envFloat("GA_GAP_REWARD", GAP_SATISFIED_REWARD),
		ApplyOnSaturday: envBool("GA_APPLY_GAP_ON_SATURDAY", false),
		// Disabled when the minimum gap is non-positive. With the constraint
		// disabled the fitness gap block is skipped and the encoder performs
		// no gap steering, so behaviour matches the pre-constraint system.
		Enabled: minHours > 0,
	}

	// Guard against an inverted range (max below min): clamp max up to min so
	// CheckGapsBetweenSubjects never reports both a "too small" and a
	// "too large" violation for the same gap.
	if cfg.MaxGapSlots < cfg.MinGapSlots {
		cfg.MaxGapSlots = cfg.MinGapSlots
	}

	gapConfig = cfg
	return cfg
}

// hoursToSlots converts a duration in hours to the nearest whole number of
// time slots (one slot = 1/N_HOUR_TIME_SLOTS of an hour).
func hoursToSlots(hours float64) int {
	slots := int(math.Round(hours * float64(Const.N_HOUR_TIME_SLOTS)))
	if slots < 0 {
		slots = 0
	}
	return slots
}

// gapShouldApplyToDay reports whether the gap constraint is active for the
// given day index, honouring the Saturday opt-out (CHALLENGE 3).
func gapShouldApplyToDay(day int) bool {
	if !gapConfig.Enabled {
		return false
	}
	if day == saturdayDayIndex() && !gapConfig.ApplyOnSaturday {
		return false
	}
	return true
}

// envBool parses a boolean environment variable, accepting the common
// true/false spellings. Unset or unrecognised values fall back to fallback.
func envBool(name string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "":
		return fallback
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

// SubjectBlock represents a continuous block of slots assigned to one subject
// in one day.
type SubjectBlock struct {
	SubjectID uint16
	StartSlot int
	EndSlot   int // exclusive (first empty slot after the block)
	SlotCount int
}

// ExtractSubjectBlocks finds all continuous subject blocks in one day's
// schedule. Consecutive slots holding the same subject id are merged into a
// single block; empty slots (subject id 0) separate blocks.
func ExtractSubjectBlocks(day Schedule.DayTimeTable) []SubjectBlock {
	blocks := make([]SubjectBlock, 0)

	i := 0
	for i < Const.N_DAILY_TIME_SLOTS {
		subjectID := day[i].GetSubjectID()
		if subjectID == 0 {
			i++
			continue
		}

		// found the start of a block
		start := i
		for i < Const.N_DAILY_TIME_SLOTS && day[i].GetSubjectID() == subjectID {
			i++
		}
		// i is now the first slot AFTER this block

		blocks = append(blocks, SubjectBlock{
			SubjectID: subjectID,
			StartSlot: start,
			EndSlot:   i,
			SlotCount: i - start,
		})
	}

	return blocks
}

// CheckGapsBetweenSubjects validates that the gaps between consecutive subject
// blocks are between minGapSlots and maxGapSlots (inclusive). It returns a list
// of human-readable violation descriptions — empty when every gap is valid (or
// when the day has fewer than two blocks, i.e. single-subject days are exempt).
func CheckGapsBetweenSubjects(day Schedule.DayTimeTable, minGapSlots, maxGapSlots int) []string {
	blocks := ExtractSubjectBlocks(day)
	violations := make([]string, 0)

	for i := 0; i < len(blocks)-1; i++ {
		current := blocks[i]
		next := blocks[i+1]

		// gap = start of next block - end of current block
		gapSlots := next.StartSlot - current.EndSlot
		gapHours := float64(gapSlots) / float64(Const.N_HOUR_TIME_SLOTS)

		if gapSlots < minGapSlots {
			violations = append(violations, fmt.Sprintf(
				"gap between subject %d and %d is %.1f hours (minimum %.1f hour required)",
				current.SubjectID, next.SubjectID, gapHours,
				float64(minGapSlots)/float64(Const.N_HOUR_TIME_SLOTS),
			))
		}
		if gapSlots > maxGapSlots {
			violations = append(violations, fmt.Sprintf(
				"gap between subject %d and %d is %.1f hours (maximum %.1f hours allowed)",
				current.SubjectID, next.SubjectID, gapHours,
				float64(maxGapSlots)/float64(Const.N_HOUR_TIME_SLOTS),
			))
		}
	}

	return violations
}

// TotalExcessGapSlots returns, for a single day, the sum of slots by which
// inter-subject gaps exceed maxGapSlots (0 when no gap is too long). It is used
// by the fitness function to grade the long-gap penalty: the further a gap runs
// beyond the allowed maximum, the harder the schedule is penalised, steering the
// GA toward compact days with short gaps between subjects.
func TotalExcessGapSlots(day Schedule.DayTimeTable, maxGapSlots int) int {
	blocks := ExtractSubjectBlocks(day)
	excess := 0

	for i := 0; i < len(blocks)-1; i++ {
		gapSlots := blocks[i+1].StartSlot - blocks[i].EndSlot
		if gapSlots > maxGapSlots {
			excess += gapSlots - maxGapSlots
		}
	}

	return excess
}

// CountGapViolations returns the total number of gap violations across all days
// in a week schedule.
func CountGapViolations(week Schedule.WeekTimeTable, minGapSlots, maxGapSlots int) int {
	total := 0
	for day := 0; day < Const.N_WEEKLY_SCHOOL_DAYS; day++ {
		violations := CheckGapsBetweenSubjects(week[day], minGapSlots, maxGapSlots)
		total += len(violations)
	}
	return total
}

// hasMinimumGapFromPreviousSubject reports whether placing a block of
// proposedSlotCount slots starting at proposedStartSlot would keep at least
// minGapSlots of empty space between it and the nearest neighbouring subject on
// both sides. It is used by the encoder as a soft steer toward valid minimum
// gaps; the maximum gap is intentionally NOT enforced here (see file header).
func hasMinimumGapFromPreviousSubject(
	day Schedule.DayTimeTable,
	proposedStartSlot int,
	proposedSlotCount int,
	minGapSlots int,
) bool {
	// check the gap FROM the previous subject TO the proposed slot
	for s := proposedStartSlot - 1; s >= 0; s-- {
		if day[s].GetSubjectID() > 0 {
			// found the end of the previous subject
			gapSlots := proposedStartSlot - (s + 1)
			if gapSlots < minGapSlots {
				return false // gap too small
			}
			break
		}
	}

	// check the gap FROM the proposed slot TO the next subject
	proposedEnd := proposedStartSlot + proposedSlotCount
	for s := proposedEnd; s < Const.N_DAILY_TIME_SLOTS; s++ {
		if day[s].GetSubjectID() > 0 {
			gapSlots := s - proposedEnd
			if gapSlots < minGapSlots {
				return false // gap too small
			}
			break
		}
	}

	return true // gap is acceptable
}
