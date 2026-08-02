package api

import "testing"

func TestResourceOperationsSupportArbitraryDimensions(t *testing.T) {
	allocated := NewResource(map[string]float64{"cpu": 2, "rdma": 1})
	request := NewResource(map[string]float64{"cpu": 1, "gpu": 2})

	got := allocated.Add(request)
	capacity := NewResource(map[string]float64{"cpu": 3, "gpu": 2, "rdma": 1})
	if !got.Fits(capacity) {
		t.Fatal("combined resource should fit exact capacity")
	}
	if allocated["gpu"] != 0 {
		t.Fatal("Add must not mutate its receiver")
	}

	remaining := got.Sub(NewResource(map[string]float64{"gpu": 1, "rdma": 1}))
	if remaining["cpu"] != 3 || remaining["gpu"] != 1 || remaining["rdma"] != 0 {
		t.Fatalf("unexpected subtraction result: %#v", remaining)
	}

	share := NewResource(map[string]float64{"cpu": 2, "gpu": 4}).DominantShare(
		NewResource(map[string]float64{"cpu": 20, "gpu": 10}),
	)
	if share != 0.4 {
		t.Fatalf("dominant share = %v, want 0.4", share)
	}
}
