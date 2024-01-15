package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

var (
	projectID  = os.Getenv("PROJECT_ID")
	collection = os.Getenv("COLLECTION_ID")
)

type Client struct {
	*firestore.Client
}

type Entry struct {
	User        string
	Date        time.Time
	CreatedDate time.Time
	Presence    string
	Reason      string
}

func NewFirestoreClient() *Client {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create firestore client: %v", err)
	}
	return &Client{client}
}

func (c *Client) SaveEntry(e Entry) (string, error) {
	ctx := context.Background()
	doc, _, err := c.Collection(collection).Add(ctx, e)
	if err != nil {
		return "", fmt.Errorf("failed to save entry: %v", err)
	}
	id := doc.ID
	return id, nil
}

func (c *Client) GetEntries(userId string) ([]Entry, error) {
	ctx := context.Background()
	iter := c.Collection(collection).
		Where("User", "==", userId).
		OrderBy("Date", firestore.Asc).
		OrderBy("CreatedDate", firestore.Asc).
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
	allEntries, err := c.GetEntries(userId)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	lastDate := time.Time{}
	for _, e := range allEntries {
		if e.Date != lastDate {
			entries = append(entries, e)
			lastDate = e.Date
		} else {
			entries[len(entries)-1] = e
		}
	}

	return entries, nil
}
