package api

// Node is a schedulable machine. Idle changes during a scheduling session;
// Capacity remains the machine's total resource vector.
type Node struct {
	Name     string   `yaml:"name" json:"name"`
	Capacity Resource `yaml:"capacity" json:"capacity"`
	Idle     Resource `yaml:"-" json:"idle"`
}

// Job describes identical task replicas that must reach MinAvailable together.
type Job struct {
	Name         string   `yaml:"name" json:"name"`
	MinAvailable int      `yaml:"minAvailable" json:"minAvailable"`
	Replicas     int      `yaml:"replicas" json:"replicas"`
	Request      Resource `yaml:"request" json:"request"`
	Allocated    Resource `yaml:"-" json:"allocated"`
}

// Task identifies one replica of a Job during trial allocation.
type Task struct {
	JobName string   `json:"jobName"`
	Index   int      `json:"index"`
	Request Resource `json:"request"`
}

// Allocation records a committed placement decision.
type Allocation struct {
	JobName   string `json:"jobName"`
	TaskIndex int    `json:"taskIndex"`
	NodeName  string `json:"nodeName"`
}

// AllocationPlan is the scheduler's side-effect-free output.
type AllocationPlan struct {
	Allocations   []Allocation      `json:"allocations"`
	Unschedulable map[string]string `json:"unschedulable"`
}

// ClusterTotal sums the capacity of every node.
func ClusterTotal(nodes []Node) Resource {
	total := NewResource(nil)
	for _, node := range nodes {
		total = total.Add(node.Capacity)
	}
	return total
}
