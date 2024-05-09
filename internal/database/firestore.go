package database

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/baely/officetracker/internal/models"
	"google.golang.org/api/iterator"
	"log/slog"
	"os"
)

type Firestore struct {
	*firestore.Client
}

func buildDocumentId(e models.Entry) string {
	return fmt.Sprintf("%s-%d-%d", e.User, e.Month, e.Year)
}

func newFirestoreClient() (*Firestore, error) {
	ctx := context.Background()
	projectID := os.Getenv("PROJECT_ID")
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create firestore client: %v", err)
	}
	return &Firestore{client}, nil
}

func (c *Firestore) SaveEntry(e models.Entry) error {
	ctx := context.Background()
	collection := os.Getenv("COLLECTION_ID")
	docId := buildDocumentId(e)
	_, err := c.Collection(collection).Doc(docId).Set(ctx, e)
	if err != nil {
		return fmt.Errorf("failed to save entry: %v", err)
	}
	slog.Info(fmt.Sprintf("saved entry with id: %s", docId))
	return nil
}

func (c *Firestore) GetEntries(userId string, month, year int) (models.Entry, error) {
	ctx := context.Background()
	collection := os.Getenv("COLLECTION_ID")
	docTitle := buildDocumentId(models.Entry{
		User:  userId,
		Month: month,
		Year:  year,
	})
	doc, err := c.Collection(collection).Doc(docTitle).Get(ctx)
	if !doc.Exists() {
		return models.Entry{}, nil
	}
	if err != nil {
		return models.Entry{}, fmt.Errorf("failed to fetch entry: %v", err)
	}

	var e models.Entry
	if err = doc.DataTo(&e); err != nil {
		return models.Entry{}, fmt.Errorf("failed to fetch entry: %v", err)
	}

	return e, nil
}

func (c *Firestore) GetAllEntries(userId string) ([]models.Entry, error) {
	ctx := context.Background()
	collection := os.Getenv("COLLECTION_ID")
	iter := c.Collection(collection).
		Where("User", "==", userId).
		OrderBy("Year", firestore.Asc).
		OrderBy("Month", firestore.Asc).
		OrderBy("Day", firestore.Asc).
		Documents(ctx)
	defer iter.Stop()

	var entries []models.Entry
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return entries, fmt.Errorf("failed to iterate entries: %v", err)
		}

		var e models.Entry
		if err := doc.DataTo(&e); err != nil {
			return entries, fmt.Errorf("failed to convert entry: %v", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (c *Firestore) GetEntriesForBankYear(userID string, bankYear int) ([]models.Entry, error) {
	ctx := context.Background()
	collection := os.Getenv("COLLECTION_ID")
	var entries []models.Entry

	iterPrev := c.Collection(collection).
		Where("User", "==", userID).
		Where("Year", "==", bankYear-1).
		Where("Month", ">=", 10).
		OrderBy("Month", firestore.Asc).
		Documents(ctx)

	for {
		doc, err := iterPrev.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return entries, fmt.Errorf("failed to iterate entries: %v", err)
		}

		var e models.Entry
		if err := doc.DataTo(&e); err != nil {
			return entries, fmt.Errorf("failed to convert entry: %v", err)
		}
		entries = append(entries, e)
	}

	iter := c.Collection(collection).
		Where("User", "==", userID).
		Where("Year", "==", bankYear).
		Where("Month", "<=", 9).
		OrderBy("Month", firestore.Asc).
		Documents(ctx)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return entries, fmt.Errorf("failed to iterate entries: %v", err)
		}

		var e models.Entry
		if err := doc.DataTo(&e); err != nil {
			return entries, fmt.Errorf("failed to convert entry: %v", err)
		}
		entries = append(entries, e)
	}

	return entries, nil
}
