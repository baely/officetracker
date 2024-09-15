package testutils

import "github.com/baely/officetracker/pkg/model"

type UnimplementedDatabaser struct {
}

func (u UnimplementedDatabaser) SaveDay(userID int, day int, month int, year int, state model.DayState) error {
	//TODO implement me
	panic("implement me")
}

func (u UnimplementedDatabaser) GetDay(userID int, day int, month int, year int) (model.DayState, error) {
	//TODO implement me
	panic("implement me")
}

func (u UnimplementedDatabaser) SaveMonth(userID int, month int, year int, state model.MonthState) error {
	//TODO implement me
	panic("implement me")
}

func (u UnimplementedDatabaser) GetMonth(userID int, month int, year int) (model.MonthState, error) {
	//TODO implement me
	panic("implement me")
}

func (u UnimplementedDatabaser) GetYear(userID int, year int) (model.YearState, error) {
	//TODO implement me
	panic("implement me")
}

func (u UnimplementedDatabaser) SaveNote(userID int, month int, year int, note string) error {
	//TODO implement me
	panic("implement me")
}

func (u UnimplementedDatabaser) GetNote(userID int, month int, year int) (model.Note, error) {
	//TODO implement me
	panic("implement me")
}

func (u UnimplementedDatabaser) GetNotes(userID int, year int) (map[int]model.Note, error) {
	//TODO implement me
	panic("implement me")
}

func (u UnimplementedDatabaser) GetUserByGHID(ghID string) (int, error) {
	//TODO implement me
	panic("implement me")
}

func (u UnimplementedDatabaser) GetUserBySecret(secret string) (int, error) {
	//TODO implement me
	panic("implement me")
}

func (u UnimplementedDatabaser) GetUser(userID int) (int, error) {
	//TODO implement me
	panic("implement me")
}

func (u UnimplementedDatabaser) SaveUserByGHID(ghID string) (int, error) {
	//TODO implement me
	panic("implement me")
}

func (u UnimplementedDatabaser) SaveSecret(userID int, secret string) error {
	//TODO implement me
	panic("implement me")
}
