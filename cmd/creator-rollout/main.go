package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"creator-rollout-pipeline/internal/creator"
	"creator-rollout-pipeline/internal/infrai"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		fmt.Fprintln(os.Stderr, "set INFRAI_API_KEY")
		os.Exit(1)
	}
	userID := os.Getenv("CREATOR_USER_ID")
	if userID == "" {
		fmt.Fprintln(os.Stderr, "set CREATOR_USER_ID to an existing flag user")
		os.Exit(1)
	}
	client := &infrai.Client{Key: key, HTTPClient: &http.Client{Timeout: 20 * time.Second}, Sleep: time.Sleep}
	plan, err := creator.ProcessDelivery(client, creator.Delivery{
		CreatorID: "creator-7",
		AssetID:   "asset-19",
		UserID:    userID,
	}, "https://example.com/internal/subscriber-update")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("digital asset delivery=%t next_job_id=%s\n", plan.Deliver, plan.NextJobID)
}
