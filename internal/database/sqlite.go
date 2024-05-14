package database

import "github.com/baely/officetracker/internal/models"

type sqliteClient struct {
}

func NewSQLiteClient() (Databaser, error) {
	return &sqliteClient{}, nil
}

func (s sqliteClient) SaveEntry(e models.Entry) error {
	//TODO implement me
	panic("implement me")
}

func (s sqliteClient) GetEntries(userID string, month, year int) (models.Entry, error) {
	//TODO implement me
	panic("implement me")
}

func (s sqliteClient) GetAllEntries(userID string) ([]models.Entry, error) {
	//TODO implement me
	panic("implement me")
}

func (s sqliteClient) GetEntriesForBankYear(userID string, bankYear int) ([]models.Entry, error) {
	//TODO implement me
	panic("implement me")
}
