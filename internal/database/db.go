package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

type Client struct {
	*firestore.Client
}

type Entry struct {
	User             string
	CreateDate       time.Time
	Day, Month, Year int
	State            int
}

func buildDocumentId(e Entry) string {
	return fmt.Sprintf("%s-%d-%d-%d", e.User, e.Day, e.Month, e.Year)
}

func generateDocumentId(userID string, day, month, year int) string {
	return fmt.Sprintf("%s-%d-%d-%d", userID, day, month, year)
}

func NewFirestoreClient() (*Client, error) {
	ctx := context.Background()
	projectID := os.Getenv("PROJECT_ID")
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create firestore client: %v", err)
	}
	return &Client{client}, nil
}

func (c *Client) SaveEntry(e Entry) (string, error) {
	ctx := context.Background()
	collection := os.Getenv("COLLECTION_ID")
	docId := buildDocumentId(e)
	_, err := c.Collection(collection).Doc(docId).Set(ctx, e)
	if err != nil {
		return "", fmt.Errorf("failed to save entry: %v", err)
	}
	return docId, nil
}

func (c *Client) GetEntries(userId string, month, year int) ([]Entry, error) {
	ctx := context.Background()
	collection := os.Getenv("COLLECTION_ID")
	iter := c.Collection(collection).
		Where("User", "==", userId).
		Where("Month", "==", month).
		Where("Year", "==", year).
		OrderBy("Day", firestore.Asc).
		Documents(ctx)
	defer iter.Stop()

	var entries []Entry
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return entries, fmt.Errorf("failed to iterate entries: %v", err)
		}

		var e Entry
		if err := doc.DataTo(&e); err != nil {
			return entries, fmt.Errorf("failed to convert entry: %v", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (c *Client) GetAllEntries(userId string) ([]Entry, error) {
	ctx := context.Background()
	collection := os.Getenv("COLLECTION_ID")
	iter := c.Collection(collection).
		Where("User", "==", userId).
		OrderBy("Year", firestore.Asc).
		OrderBy("Month", firestore.Asc).
		OrderBy("Day", firestore.Asc).
		Documents(ctx)
	defer iter.Stop()

	var entries []Entry
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return entries, fmt.Errorf("failed to iterate entries: %v", err)
		}

		var e Entry
		if err := doc.DataTo(&e); err != nil {
			return entries, fmt.Errorf("failed to convert entry: %v", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (c *Client) GetLatestEntries(userId string) ([]Entry, error) {
	allEntries, err := c.GetAllEntries(userId)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, e := range allEntries {
		entries = append(entries, e)
	}

	return entries, nil
}

func (c *Client) GetEntriesForBankYear(userID string, bankYear int) ([]Entry, error) {
	ctx := context.Background()
	collection := os.Getenv("COLLECTION_ID")
	var entries []Entry

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

		var e Entry
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

		var e Entry
		if err := doc.DataTo(&e); err != nil {
			return entries, fmt.Errorf("failed to convert entry: %v", err)
		}
		entries = append(entries, e)
	}

	return entries, nil
}
