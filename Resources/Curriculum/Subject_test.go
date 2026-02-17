package Curriculum_test

import (
	"testing"

	sub "github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
)

func TestSubjectDbTableItem(t *testing.T) {
	subject := sub.Subject{}

	//////////////

	subject.LecHours = 5
	if subject.LecHours != 5 {
		t.Error("subject.LecHours = 5 - failed")
	}

	if subject.LabHours != 0 {
		t.Error("subject.LecHours = 5 + subject.LabHours != 0 - failed")
	}

	//////////////

	subject.LecHours = 0
	if subject.LecHours != 0 {
		t.Errorf("subject.LecHours = 0 - failed = %d", subject.LecHours)
	}

	if subject.LabHours != 0 {
		t.Error("subject.LecHours = 0 + subject.LabHours != 0 - failed")
	}

	//////////////

	subject.LabHours = 5
	if subject.LabHours != 5 {
		t.Errorf("subject.LabHours = 5  - failed = %d", subject.LabHours)
	}

	if subject.LecHours != 0 {
		t.Error("subject.LabHours = 5  + subject.LecHours != 0 - failed")
	}

	//////////////

	subject.LabHours = 0
	if subject.LecHours != 0 {
		t.Error("subject.LabHours = 0 - failed")
	}

	if subject.LabHours != 0 {
		t.Error("subject.LabHours = 0 + subject.LabHours != 0 - failed")
	}

	//////////////

	if subject.IsGymType() {
		t.Error("subject.IsGymType() - failed - this should not be a gym type yet")
	}
}

func TestSubjectDbRecordSerialization(t *testing.T) {
	subject := sub.Subject{}

	subject.LecHours = 5
	subject.LabHours = 2
	subject.BitFlags = 7

	serialized_subject_hours := subject.Serialize()

	if subject.LecHours != 5 {
		t.Error("Serialization Wrong Lec Hours")
	}

	if subject.LabHours != 2 {
		t.Error("Serialization Wrong Lab Hours")
	}

	if subject.BitFlags != 7 {
		t.Error("Serialization Wrong Gym Hours")
	}

	subject.LecHours = 0
	subject.LabHours = 0
	subject.BitFlags = 0

	if subject.LecHours != 0 {
		t.Error("Serialization Wrong Lec Hours")
	}

	if subject.LabHours != 0 {
		t.Error("Serialization Wrong Lab Hours")
	}

	if subject.BitFlags != 0 {
		t.Error("Serialization Wrong Gym Hours")
	}

	subject.Deserialize(serialized_subject_hours)

	if subject.LecHours != 5 {
		t.Error("Serialization Wrong Lec Hours")
	}

	if subject.LabHours != 2 {
		t.Error("Serialization Wrong Lab Hours")
	}

	if subject.BitFlags != 7 {
		t.Error("Serialization Wrong Gym Hours")
	}
}

func TestSubjectDbRecord(t *testing.T) {
	subject := sub.Subject{}

	subject.LecHours = (6)

	if subject.LecHours != 6 {
		t.Error("subject.LecHours = (6) - failed")
	}

	subject.LabHours = (7)

	if subject.LabHours != 7 {
		t.Error("subject.LabHours = (7) - failed")
	}

	if subject.LecHours != 6 {
		t.Errorf("subject.LecHours = (6) = (%d) - failed 2nd ", subject.LecHours)
	}

	subject.BitFlags = (1)

	if subject.BitFlags != 1 {
		t.Error("subject.BitFlags != 0 : failed")
	}
}
