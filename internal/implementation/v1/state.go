package v1

import (
	"fmt"

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

	return model.GetYearResponse{
		Data: state,
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
