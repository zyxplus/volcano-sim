package scheduler

import (
	"sort"

	"github.com/zhouyingxiao/volcano-sim/pkg/api"
)

// Scheduler owns mutable node-idle state for one in-memory scheduling session.
type Scheduler struct {
	nodes []api.Node
	total api.Resource
}

// New creates a scheduler whose node state is independent of the caller.
func New(nodes []api.Node) *Scheduler {
	cloned := make([]api.Node, len(nodes))
	for i, node := range nodes {
		cloned[i] = node
		cloned[i].Capacity = api.NewResource(node.Capacity)
		if node.Idle == nil {
			cloned[i].Idle = api.NewResource(node.Capacity)
		} else {
			cloned[i].Idle = api.NewResource(node.Idle)
		}
	}
	sort.Slice(cloned, func(i, j int) bool { return cloned[i].Name < cloned[j].Name })
	return &Scheduler{nodes: cloned, total: api.ClusterTotal(cloned)}
}

// OrderJobs returns a sorted copy, preserving the caller's input slice.
func (s *Scheduler) OrderJobs(jobs []api.Job) []api.Job {
	ordered := append([]api.Job(nil), jobs...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left := ordered[i].Allocated.DominantShare(s.total)
		right := ordered[j].Allocated.DominantShare(s.total)
		if left == right {
			return ordered[i].Name < ordered[j].Name
		}
		return left < right
	})
	return ordered
}

type trialAllocation struct {
	allocation api.Allocation
	nodeIndex  int
	request    api.Resource
}

// Run schedules each job as one gang transaction. It commits all successful
// trial allocations only when the transaction reaches MinAvailable.
func (s *Scheduler) Run(jobs []api.Job) api.AllocationPlan {
	plan := api.AllocationPlan{Unschedulable: make(map[string]string)}
	for _, job := range s.OrderJobs(jobs) {
		candidates, reason := s.selectCandidateNodes(job)
		if reason != "" {
			plan.Unschedulable[job.Name] = reason
			continue
		}
		journal := make([]trialAllocation, 0, job.Replicas)
		for taskIndex := 0; taskIndex < job.Replicas; taskIndex++ {
			nodeIndex := s.findNode(job.Request, candidates)
			if nodeIndex < 0 {
				break
			}
			s.nodes[nodeIndex].Idle = s.nodes[nodeIndex].Idle.Sub(job.Request)
			journal = append(journal, trialAllocation{
				allocation: api.Allocation{JobName: job.Name, TaskIndex: taskIndex, NodeName: s.nodes[nodeIndex].Name},
				nodeIndex:  nodeIndex,
				request:    job.Request,
			})
		}

		if len(journal) < job.MinAvailable {
			for index := len(journal) - 1; index >= 0; index-- {
				trial := journal[index]
				s.nodes[trial.nodeIndex].Idle = s.nodes[trial.nodeIndex].Idle.Add(trial.request)
			}
			plan.Unschedulable[job.Name] = "insufficient resources for minAvailable"
			continue
		}
		for _, trial := range journal {
			plan.Allocations = append(plan.Allocations, trial.allocation)
		}
	}
	return plan
}

// selectCandidateNodes chooses one complete matching Fabric before any Gang
// transaction changes node idle resources.
func (s *Scheduler) selectCandidateNodes(job api.Job) ([]int, string) {
	if job.Topology == nil || job.Topology.SameFabric == "" {
		candidates := make([]int, len(s.nodes))
		for index := range s.nodes {
			candidates[index] = index
		}
		return candidates, ""
	}

	need := api.NewResource(nil)
	for replica := 0; replica < job.MinAvailable; replica++ {
		need = need.Add(job.Request)
	}
	fabrics := make(map[string][]int)
	for index, node := range s.nodes {
		if node.Labels["gpu-model"] != job.Topology.GPUModel || node.Labels["fabric-id"] == "" {
			continue
		}
		fabricID := node.Labels["fabric-id"]
		fabrics[fabricID] = append(fabrics[fabricID], index)
	}

	fabricIDs := make([]string, 0, len(fabrics))
	for fabricID := range fabrics {
		fabricIDs = append(fabricIDs, fabricID)
	}
	sort.Strings(fabricIDs)
	for _, fabricID := range fabricIDs {
		idle := api.NewResource(nil)
		for _, index := range fabrics[fabricID] {
			idle = idle.Add(s.nodes[index].Idle)
		}
		if need.Fits(idle) {
			return fabrics[fabricID], ""
		}
	}
	return nil, "no single fabric has enough capacity"
}

func (s *Scheduler) findNode(request api.Resource, candidates []int) int {
	for _, index := range candidates {
		node := s.nodes[index]
		if request.Fits(node.Idle) {
			return index
		}
	}
	return -1
}
