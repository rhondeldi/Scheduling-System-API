package GeneticAlgorithm

import (
	"testing"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
	"github.com/mrdcvlsc/scheduling-system-backend/Schedule"
)

func TestIsNSTP1Or2Subject(t *testing.T) {
	tests := []struct {
		name    string
		subject Curriculum.Subject
		want    bool
	}{
		{
			name: "Code NSTP 1",
			subject: Curriculum.Subject{
				Code: "NSTP 1",
				Name: "National Service Training Program 1",
			},
			want: true,
		},
		{
			name: "Code NSTP2 compact",
			subject: Curriculum.Subject{
				Code: "NSTP2",
				Name: "Civic Welfare",
			},
			want: true,
		},
		{
			name: "Name NSTP-2",
			subject: Curriculum.Subject{
				Code: "CWTS",
				Name: "NSTP-2",
			},
			want: true,
		},
		{
			name: "NSTP 3 is not included",
			subject: Curriculum.Subject{
				Code: "NSTP 3",
				Name: "National Service Training Program 3",
			},
			want: false,
		},
		{
			name: "Regular subject",
			subject: Curriculum.Subject{
				Code: "MATH 101",
				Name: "Calculus 1",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		got := isNSTP1Or2Subject(tt.subject)
		if got != tt.want {
			t.Fatalf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestBuildNSTP1Or2SubjectIDSetAndWeekDayCheck(t *testing.T) {
	curriculums := []Curriculum.Curriculum{
		{
			YearLevels: []Curriculum.YearLevel{
				{
					IsActive: true,
					Semesters: []Curriculum.Semester{
						{
							Subjects: []Curriculum.Subject{
								{ID: 11, Code: "NSTP 1", Name: "NSTP 1"},
								{ID: 12, Code: "NSTP 2", Name: "NSTP 2"},
								{ID: 99, Code: "MATH", Name: "Math"},
							},
						},
					},
				},
			},
		},
	}

	nstpSubjectIDs := buildNSTP1Or2SubjectIDSet(curriculums)

	if !nstpSubjectIDs[11] || !nstpSubjectIDs[12] {
		t.Fatalf("expected NSTP subject IDs to be detected")
	}

	if nstpSubjectIDs[99] {
		t.Fatalf("did not expect non-NSTP subject ID in NSTP set")
	}

	week := &Schedule.WeekTimeTable{}
	week[saturdayDayIndex()][0].SetSubjectID(11)
	week[0][0].SetSubjectID(99)

	if !weekDayHasNSTP1Or2(week, saturdayDayIndex(), nstpSubjectIDs) {
		t.Fatalf("expected Saturday to contain NSTP subject")
	}

	if weekDayHasNSTP1Or2(week, 0, nstpSubjectIDs) {
		t.Fatalf("did not expect Monday to contain NSTP subject")
	}
}
