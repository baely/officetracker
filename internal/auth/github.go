package auth

import (
	"errors"

	"github.com/baely/officetracker/internal/database"
)

func toUserID(db database.Databaser, ghID string) (int, error) {
	userID, err := db.GetUserByGHID(ghID)
	if errors.Is(err, database.ErrNoUser) {
		userID, err = db.SaveUserByGHID(ghID)
	}
	if err != nil {
		return 0, err
	}
	return userID, nil
}
