package creator

import "testing"

func TestDisabledDeliveryStopsBeforeScheduling(t *testing.T) {
	input := Delivery{CreatorID: "creator-7", AssetID: "asset-19", UserID: "subscriber-3"}
	if input.CreatorID == "" || input.AssetID == "" || input.UserID == "" {
		t.Fatal("test input must describe a delivery row")
	}
	plan := DecideDelivery(false)
	if plan.Deliver {
		t.Fatal("disabled creator rollout must not deliver")
	}
	if plan.NextJobID != "" {
		t.Fatal("disabled creator rollout must not schedule a job")
	}
}
