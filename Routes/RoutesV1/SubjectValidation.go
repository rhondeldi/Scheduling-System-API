package RoutesV1

import (
	"errors"
	"math"
	"strings"

	"github.com/mrdcvlsc/scheduling-system-backend/Resources/Curriculum"
)

type SubjectUpsertPayload struct {
	ID                    uint16   `json:"ID"`
	Code                  string   `json:"Code"`
	Name                  string   `json:"Name"`
	LecHours              uint8    `json:"LecHours"`
	LabHours              uint8    `json:"LabHours"`
	LecUnits              uint8    `json:"LecUnits"`
	LabUnits              uint8    `json:"LabUnits"`
	Units                 uint8    `json:"Units"`
	BitFlags              uint16   `json:"BitFlags"`
	DesignatedInstructors []uint16 `json:"DesignatedInstructorsID"`
	SubjectType           string   `json:"SubjectType"`
	AsynchronousHours     *float64 `json:"AsynchronousHours"`
	SaturdayOnly          *bool    `json:"SaturdayOnly"`

	SubjectTypeSnake       string   `json:"subject_type"`
	AsynchronousHoursSnake *float64 `json:"asynchronous_hours"`
	TotalHoursSnake        *float64 `json:"total_hours"`
}

func buildSubjectFromPayload(payload SubjectUpsertPayload) Curriculum.Subject {
	subject := Curriculum.Subject{
		ID:                    payload.ID,
		Code:                  payload.Code,
		Name:                  payload.Name,
		LecHours:              payload.LecHours,
		LabHours:              payload.LabHours,
		LecUnits:              payload.LecUnits,
		LabUnits:              payload.LabUnits,
		BitFlags:              payload.BitFlags,
		DesignatedInstructors: payload.DesignatedInstructors,
		SubjectType:           payload.SubjectType,
	}

	if subject.SubjectType == "" && payload.SubjectTypeSnake != "" {
		subject.SubjectType = payload.SubjectTypeSnake
	}

	if payload.AsynchronousHours != nil {
		subject.AsynchronousHours = *payload.AsynchronousHours
	}

	if payload.AsynchronousHoursSnake != nil {
		subject.AsynchronousHours = *payload.AsynchronousHoursSnake
	}

	if payload.SaturdayOnly != nil {
		subject.SaturdayOnly = *payload.SaturdayOnly
	}

	// Compatibility alias for external clients that send total_hours.
	if payload.TotalHoursSnake != nil && subject.LecHours == 0 && subject.LabHours == 0 {
		totalHoursRounded := uint8(math.Round(*payload.TotalHoursSnake))
		if strings.EqualFold(subject.SubjectType, Curriculum.SUBJECT_TYPE_LABORATORY) {
			subject.LabHours = totalHoursRounded
		} else {
			subject.LecHours = totalHoursRounded
		}
	}

	return subject
}

func normalizeAndValidateSubjectPayload(subject *Curriculum.Subject) error {
	if subject == nil {
		return errors.New("missing subject payload")
	}

	subject.Code = strings.TrimSpace(subject.Code)
	subject.Name = strings.TrimSpace(subject.Name)

	if subject.Code == "" {
		return errors.New("subject code cannot be empty")
	}

	if subject.Name == "" {
		return errors.New("subject name cannot be empty")
	}

	if subject.AsynchronousHours < 0 {
		return errors.New("asynchronous hours cannot be negative")
	}

	totalHours := subject.TotalHours()
	if totalHours > 0 && subject.AsynchronousHours >= totalHours {
		return errors.New("asynchronous hours must be less than total subject hours")
	}

	if subject.AsynchronousHours > float64(subject.LecHours) {
		return errors.New("asynchronous hours cannot exceed lecture hours")
	}

	// total credit units is always the sum of the lecture and laboratory units.
	subject.Units = subject.LecUnits + subject.LabUnits

	subject.NormalizeAsyncConfig()

	return nil
}
