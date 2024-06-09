package v1

import "github.com/baely/officetracker/pkg/model"

func (i *implementation) GetDay(req model.GetDayRequest) (model.GetDayResponse, error) {
	state, err := i.db.GetDay(req.Meta.UserID, req.Meta.Day, req.Meta.Month, req.Meta.Year)
	if err != nil {
		return model.GetDayResponse{}, err
	}

	return model.GetDayResponse{
		Data: state,
	}, nil
}

func (i *implementation) PutDay(req model.PutDayRequest) (model.PutDayResponse, error) {
	err := i.db.SaveDay(req.Meta.UserID, req.Meta.Day, req.Meta.Month, req.Meta.Year, req.Data)
	if err != nil {
		return model.PutDayResponse{}, err
	}

	return model.PutDayResponse{}, nil
}

func (i *implementation) GetMonth(req model.GetMonthRequest) (model.GetMonthResponse, error) {
	state, err := i.db.GetMonth(req.Meta.UserID, req.Meta.Month, req.Meta.Year)
	if err != nil {
		return model.GetMonthResponse{}, err
	}

	return model.GetMonthResponse{
		Data: state,
	}, nil
}

func (i *implementation) PutMonth(req model.PutMonthRequest) (model.PutMonthResponse, error) {
	err := i.db.SaveMonth(req.Meta.UserID, req.Meta.Month, req.Meta.Year, req.Data)
	if err != nil {
		return model.PutMonthResponse{}, err
	}

	return model.PutMonthResponse{}, nil
}

func (i *implementation) GetYear(req model.GetYearRequest) (model.GetYearResponse, error) {
	state, err := i.db.GetYear(req.Meta.UserID, req.Meta.Year)
	if err != nil {
		return model.GetYearResponse{}, err
	}

	return model.GetYearResponse{
		Data: state,
	}, nil
}

func (i *implementation) GetNote(req model.GetNoteRequest) (model.GetNoteResponse, error) {
	note, err := i.db.GetNote(req.Meta.UserID, req.Meta.Month, req.Meta.Year)
	if err != nil {
		return model.GetNoteResponse{}, err
	}

	return model.GetNoteResponse{
		Data: model.Note{
			Note: note,
		},
	}, nil
}

func (i *implementation) PutNote(req model.PutNoteRequest) (model.PutNoteResponse, error) {
	err := i.db.SaveNote(req.Meta.UserID, req.Meta.Month, req.Meta.Year, req.Data.Note)
	if err != nil {
		return model.PutNoteResponse{}, err
	}

	return model.PutNoteResponse{}, nil
}
