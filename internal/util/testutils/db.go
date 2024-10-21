package testutils

import "github.com/baely/officetracker/pkg/model"

type MockDatabase struct {
}

func NewMockDatabase() MockDatabase {
	return MockDatabase{}
}

func (m MockDatabase) SaveDay(userID int, day int, month int, year int, state model.DayState) error {
	//TODO implement me
	panic("implement me")
}

func (m MockDatabase) GetDay(userID int, day int, month int, year int) (model.DayState, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockDatabase) SaveMonth(userID int, month int, year int, state model.MonthState) error {
	//TODO implement me
	panic("implement me")
}

func (m MockDatabase) GetMonth(userID int, month int, year int) (model.MonthState, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockDatabase) GetYear(userID int, year int) (model.YearState, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockDatabase) SaveNote(userID int, month int, year int, note string) error {
	//TODO implement me
	panic("implement me")
}

func (m MockDatabase) GetNote(userID int, month int, year int) (model.Note, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockDatabase) GetNotes(userID int, year int) (map[int]model.Note, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockDatabase) GetUserByGHID(ghID string) (int, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockDatabase) GetUserBySecret(secret string) (int, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockDatabase) GetUser(userID int) (int, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockDatabase) SaveUserByGHID(ghID string) (int, error) {
	//TODO implement me
	panic("implement me")
}

func (m MockDatabase) SaveSecret(userID int, secret string) error {
	//TODO implement me
	panic("implement me")
}
