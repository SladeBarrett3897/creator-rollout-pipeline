package creator

import (
	"fmt"

	"creator-rollout-pipeline/internal/infrai"
)

type Delivery struct {
	CreatorID string
	AssetID   string
	UserID    string
}

type Plan struct {
	Deliver   bool
	NextJobID string
}

func DecideDelivery(enabled bool) Plan {
	return Plan{Deliver: enabled}
}

func ProcessDelivery(client *infrai.Client, input Delivery, taskURL string) (Plan, error) {
	enabled, err := client.FlagValue("creator-commerce", input.UserID)
	if err != nil {
		return Plan{}, fmt.Errorf("read creator flag: %w", err)
	}
	if !DecideDelivery(enabled).Deliver {
		return DecideDelivery(false), nil
	}
	requestID := input.CreatorID + ":" + input.AssetID + ":rollout"
	if err := client.Rollout("creator-commerce", requestID); err != nil {
		return Plan{}, fmt.Errorf("advance creator rollout: %w", err)
	}
	jobID, err := client.CreateCron("0 * * * *", taskURL, input.CreatorID+":"+input.AssetID+":schedule")
	if err != nil {
		return Plan{}, fmt.Errorf("schedule subscriber update: %w", err)
	}
	return Plan{Deliver: true, NextJobID: jobID}, nil
}
