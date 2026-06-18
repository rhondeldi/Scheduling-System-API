package StorageResources

import (
	"os"
	"path"
	"testing"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Const"
	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Instructors"
	"github.com/mrdcvlsc/scheduling-system-backend/Utils"
)

// TestInstructorEmploymentPersistenceRoundTrip proves the JSON store now writes
// and reads back EmploymentType / MaxUnits — the fields that were previously
// dropped, causing part-time instructors to be treated as regular.
//
// It seeds a temporary instructors.json in the active data dir (backing up any
// existing file), creates a part-time instructor, then reads it back and
// asserts the employment fields survived the round trip and that
// EffectiveMaxUnits reflects the part-time cap.
func TestInstructorEmploymentPersistenceRoundTrip(t *testing.T) {
	project_root, err := Utils.FindProjectRoot()
	if err != nil {
		t.Fatalf("cannot find project root: %v", err)
	}

	data_dir := path.Join(project_root, "scheduling-system-temporary-data")
	if err := os.MkdirAll(data_dir, 0755); err != nil {
		t.Fatalf("cannot ensure data dir: %v", err)
	}

	instructors_file := path.Join(data_dir, "instructors.json")

	// back up any existing instructors.json and restore it afterwards.
	original, had_original := os.ReadFile(instructors_file)
	t.Cleanup(func() {
		if had_original == nil {
			_ = os.WriteFile(instructors_file, original, 0644)
		} else {
			_ = os.Remove(instructors_file)
		}
	})

	// start from an empty list so CreateInstructor has a file to read.
	if err := os.WriteFile(instructors_file, []byte("[]"), 0644); err != nil {
		t.Fatalf("cannot seed instructors.json: %v", err)
	}

	writer := &JsonWriter{}
	reader := &JsonReader{}

	// create a part-time instructor capped at 15 units.
	err = writer.CreateInstructor(Instructors.Instructor{
		DepartmentID:   1,
		FirstName:      "PART",
		MiddleInitial:  "T",
		LastName:       "TIMER",
		EmploymentType: Instructors.EMPLOYMENT_TYPE_PART_TIME,
		MaxUnits:       15,
	})
	if err != nil {
		t.Fatalf("CreateInstructor failed: %v", err)
	}

	all, err := reader.ReadAllInstructors()
	if err != nil {
		t.Fatalf("ReadAllInstructors failed: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 instructor, got %d", len(all))
	}

	got := all[0]
	if got.EmploymentType != Instructors.EMPLOYMENT_TYPE_PART_TIME {
		t.Errorf("EmploymentType not persisted: got %q, want %q", got.EmploymentType, Instructors.EMPLOYMENT_TYPE_PART_TIME)
	}
	if got.MaxUnits != 15 {
		t.Errorf("MaxUnits not persisted: got %d, want 15", got.MaxUnits)
	}
	if eff := got.EffectiveMaxUnits(); eff != 15 {
		t.Errorf("EffectiveMaxUnits: got %d, want 15 (the part-time cap the GA enforces)", eff)
	}

	// flip back to regular via update and confirm the cap becomes the fixed
	// regular cap.
	got.EmploymentType = Instructors.EMPLOYMENT_TYPE_REGULAR
	got.MaxUnits = Const.REGULAR_INSTRUCTOR_MAX_UNITS
	if err := writer.UpdateInstructor(got); err != nil {
		t.Fatalf("UpdateInstructor failed: %v", err)
	}

	updated, err := reader.ReadInstructor(got.InstructorID)
	if err != nil || updated == nil {
		t.Fatalf("ReadInstructor failed: %v", err)
	}
	if updated.NormalizedEmploymentType() != Instructors.EMPLOYMENT_TYPE_REGULAR {
		t.Errorf("regular type not persisted: got %q", updated.EmploymentType)
	}
	if eff := updated.EffectiveMaxUnits(); eff != Const.REGULAR_INSTRUCTOR_MAX_UNITS {
		t.Errorf("regular EffectiveMaxUnits: got %d, want %d", eff, Const.REGULAR_INSTRUCTOR_MAX_UNITS)
	}
}
