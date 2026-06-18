package Const

// specifies the number of days per week the university holds classes.
const N_WEEKLY_SCHOOL_DAYS = 6

// specifies the total number of teaching hours the university is open per day.
const N_DAILY_SCHOOL_HOURS = 12 //

// indicates the number of time slots available within a single teaching hour.
const N_HOUR_TIME_SLOTS = 2

// represents the total number of time slots available in a single teaching day.
const N_DAILY_TIME_SLOTS = N_DAILY_SCHOOL_HOURS * N_HOUR_TIME_SLOTS

// the total number of time slots available for scheduling in a week.
const N_WEEKLY_TIME_SLOTS = N_WEEKLY_SCHOOL_DAYS * N_DAILY_TIME_SLOTS

// the maximum number of distinct days a single section's non-NSTP classes may
// be spread across. NSTP 1/2 subjects are Saturday-pinned and do NOT count
// against this limit, so a section can still have an NSTP class on a further day.
//
// Set to 4 so each section's week is compact (classes confined to 4 days).
// Naively capping every section at 4 days saturates the first days (all sections
// pack onto Mon-Thu) and made dense departments infeasible. To keep 4 feasible,
// genesis (EncodeIndividualGenome) STAGGERS each section's preferred starting day
// across the week and fills day-major from there, so different sections occupy
// different 4-day windows and the department load spreads over all 6 days while
// every section stays within 4 days. Day-major filling also keeps each day's
// classes compact (no large break-time gaps).
const MAX_NON_NSTP_SCHOOL_DAYS = 5

// the fixed weekly teaching unit cap for a "regular" instructor. It is not
// configurable per-instructor: every regular instructor is capped here. Part-time
// instructors do NOT use this value; their cap is derived from their weekly
// availability (see Instructor.EffectiveMaxUnits) and always stays below it.
// Instructors must never be scheduled beyond their cap (no overload).
const REGULAR_INSTRUCTOR_MAX_UNITS = 32
