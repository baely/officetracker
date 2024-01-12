package main

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/firestore"
)

var (
	projectID  = os.Getenv("PROJECT_ID")
	collection = os.Getenv("COLLECTION_ID")
)

func newFirestoreClient() *firestore.Client {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create firestore client: %v", err)
	}
	return client
}
