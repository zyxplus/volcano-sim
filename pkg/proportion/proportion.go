// Package proportion computes weighted Queue resource shares.
package proportion

import "github.com/zhouyingxiao/volcano-sim/pkg/api"

// ComputeDeserved distributes each cluster resource by Queue weight, then
// clamps each Queue's result between its guarantee and capability.
func ComputeDeserved(queues []api.Queue, total api.Resource) map[string]api.Resource {
	weightTotal := 0
	for _, queue := range queues {
		weightTotal += queue.Weight
	}
	result := make(map[string]api.Resource, len(queues))
	for _, queue := range queues {
		deserved := api.NewResource(nil)
		for name, quantity := range total {
			value := quantity * float64(queue.Weight) / float64(weightTotal)
			if value < queue.Guarantee[name] {
				value = queue.Guarantee[name]
			}
			if queue.Capability[name] > 0 && value > queue.Capability[name] {
				value = queue.Capability[name]
			}
			deserved[name] = value
		}
		result[queue.Name] = deserved
	}
	return result
}

// IsOverused reports whether allocated resource exceeds deserved resource.
func IsOverused(queue api.Queue, deserved api.Resource) bool { return !queue.Allocated.Fits(deserved) }
