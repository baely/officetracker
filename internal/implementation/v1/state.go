package v1

import (
	"fmt"
	"time"

	"github.com/baely/officetracker/pkg/model"
)

func (i *Service) GetDay(req model.GetDayRequest) (model.GetDayResponse, error) {
	state, err := i.db.GetDay(req.Meta.UserID, req.Meta.Day, req.Meta.Month, req.Meta.Year)
	if err != nil {
		err = fmt.Errorf("failed to get day: %w", err)
		return model.GetDayResponse{}, err
	}

	return model.GetDayResponse{
		Data: state,
	}, nil
}

func (i *Service) PutDay(req model.PutDayRequest) (model.PutDayResponse, error) {
	err := i.db.SaveDay(req.Meta.UserID, req.Meta.Day, req.Meta.Month, req.Meta.Year, req.Data)
	if err != nil {
		err = fmt.Errorf("failed to save day: %w", err)
		return model.PutDayResponse{}, err
	}

	return model.PutDayResponse{}, nil
}

func (i *Service) GetMonth(req model.GetMonthRequest) (model.GetMonthResponse, error) {
	state, err := i.db.GetMonth(req.Meta.UserID, req.Meta.Month, req.Meta.Year)
	if err != nil {
		err = fmt.Errorf("failed to get month: %w", err)
		return model.GetMonthResponse{}, err
	}

	return model.GetMonthResponse{
		Data: state,
	}, nil
}

func (i *Service) PutMonth(req model.PutMonthRequest) (model.PutMonthResponse, error) {
	err := i.db.SaveMonth(req.Meta.UserID, req.Meta.Month, req.Meta.Year, req.Data)
	if err != nil {
		err = fmt.Errorf("failed to save month: %w", err)
		return model.PutMonthResponse{}, err
	}

	return model.PutMonthResponse{}, nil
}

func (i *Service) GetYear(req model.GetYearRequest) (model.GetYearResponse, error) {
	state, err := i.db.GetYear(req.Meta.UserID, req.Meta.Year)
	if err != nil {
		err = fmt.Errorf("failed to get year: %w", err)
		return model.GetYearResponse{}, err
	}

	// Get schedule preferences to merge with actual state
	schedulePrefs, err := i.db.GetSchedulePreferences(req.Meta.UserID)
	if err != nil {
		err = fmt.Errorf("failed to get schedule preferences: %w", err)
		return model.GetYearResponse{}, err
	}

	// Merge schedule preferences with actual state
	mergedState := i.mergeScheduleWithYear(state, schedulePrefs, req.Meta.Year)

	return model.GetYearResponse{
		Data: mergedState,
	}, nil
}

func (i *Service) GetNote(req model.GetNoteRequest) (model.GetNoteResponse, error) {
	note, err := i.db.GetNote(req.Meta.UserID, req.Meta.Month, req.Meta.Year)
	if err != nil {
		err = fmt.Errorf("failed to get note: %w", err)
		return model.GetNoteResponse{}, err
	}

	return model.GetNoteResponse{
		Data: note,
	}, nil
}

func (i *Service) PutNote(req model.PutNoteRequest) (model.PutNoteResponse, error) {
	err := i.db.SaveNote(req.Meta.UserID, req.Meta.Month, req.Meta.Year, req.Data.Note)
	if err != nil {
		err = fmt.Errorf("failed to save note: %w", err)
		return model.PutNoteResponse{}, err
	}

	return model.PutNoteResponse{}, nil
}

func (i *Service) GetNotes(req model.GetNotesRequest) (model.GetNotesResponse, error) {
	notes, err := i.db.GetNotes(req.Meta.UserID, req.Meta.Year)
	if err != nil {
		err = fmt.Errorf("failed to get notes: %w", err)
		return model.GetNotesResponse{}, err
	}

	return model.GetNotesResponse{
		Data: notes,
	}, nil
}

// mergeScheduleWithYear merges schedule preferences with actual state data for a year
func (i *Service) mergeScheduleWithYear(yearState model.YearState, schedulePrefs model.SchedulePreferences, year int) model.YearState {
	// Create a map for day of week to schedule state
	dayOfWeekToState := map[time.Weekday]model.State{
		time.Sunday:    schedulePrefs.Sunday,
		time.Monday:    schedulePrefs.Monday,
		time.Tuesday:   schedulePrefs.Tuesday,
		time.Wednesday: schedulePrefs.Wednesday,
		time.Thursday:  schedulePrefs.Thursday,
		time.Friday:    schedulePrefs.Friday,
		time.Saturday:  schedulePrefs.Saturday,
	}

	// Process each month
	for month := 1; month <= 12; month++ {
		// Determine which year this month belongs to (academic year logic)
		var monthYear int
		if month <= 9 {
			monthYear = year
		} else {
			monthYear = year - 1
		}

		// Initialize month if it doesn't exist
		if yearState.Months == nil {
			yearState.Months = make(map[int]model.MonthState)
		}
		if _, exists := yearState.Months[month]; !exists {
			yearState.Months[month] = model.MonthState{
				Days: make(map[int]model.DayState),
			}
		}

		// Get days in this month
		daysInMonth := time.Date(monthYear, time.Month(month+1), 0, 0, 0, 0, 0, time.UTC).Day()

		// Process each day in the month
		for day := 1; day <= daysInMonth; day++ {
			date := time.Date(monthYear, time.Month(month), day, 0, 0, 0, 0, time.UTC)
			dayOfWeek := date.Weekday()
			
			// Check if this day has actual state data
			monthState := yearState.Months[month]
			dayState, hasActualState := monthState.Days[day]
			
			// Show scheduled state if:
			// 1. No actual state is set, OR
			// 2. Actual state is explicitly set to untracked (0)
			shouldShowScheduled := !hasActualState || (hasActualState && dayState.State == model.StateUntracked)
			
			if shouldShowScheduled {
				// Check if there's a schedule for this day
				if scheduledState := dayOfWeekToState[dayOfWeek]; scheduledState != model.StateUntracked {
					// Convert regular state to scheduled state
					var actualScheduledState model.State
					switch scheduledState {
					case model.StateWorkFromHome:
						actualScheduledState = model.StateScheduledWorkFromHome
					case model.StateWorkFromOffice:
						actualScheduledState = model.StateScheduledWorkFromOffice
					case model.StateOther:
						actualScheduledState = model.StateScheduledOther
					default:
						continue // Skip untracked
					}
					// Add/update with scheduled state
					monthState.Days[day] = model.DayState{State: actualScheduledState}
					yearState.Months[month] = monthState
				}
			}
		}
	}

	return yearState
}
