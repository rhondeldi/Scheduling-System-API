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
