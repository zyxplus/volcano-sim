package api

// Node is a schedulable machine. Idle changes during a scheduling session;
// Capacity remains the machine's total resource vector.
type Node struct {
	Name     string            `yaml:"name" json:"name"`
	Capacity Resource          `yaml:"capacity" json:"capacity"`
	Labels   map[string]string `yaml:"labels" json:"labels"`
	Idle     Resource          `yaml:"-" json:"idle"`
}

// Topology constrains where a Job's task replicas may be placed.
type Topology struct {
	GPUModel   string `yaml:"gpuModel" json:"gpuModel"`
	SameFabric string `yaml:"sameFabric" json:"sameFabric"`
}

// Queue groups jobs under a weighted, resource-capped admission boundary.
type Queue struct {
	Name        string   `yaml:"name" json:"name"`
	Weight      int      `yaml:"weight" json:"weight"`
	Capability  Resource `yaml:"capability" json:"capability"`
	Guarantee   Resource `yaml:"guarantee" json:"guarantee"`
	Reclaimable bool     `yaml:"reclaimable" json:"reclaimable"`
	Allocated   Resource `yaml:"-" json:"allocated"`
}

// Job describes identical task replicas that must reach MinAvailable together.
type Job struct {
	Name         string    `yaml:"name" json:"name"`
	MinAvailable int       `yaml:"minAvailable" json:"minAvailable"`
	Replicas     int       `yaml:"replicas" json:"replicas"`
	Request      Resource  `yaml:"request" json:"request"`
	Queue        string    `yaml:"queue" json:"queue"`
	Topology     *Topology `yaml:"topology" json:"topology,omitempty"`
	Allocated    Resource  `yaml:"-" json:"allocated"`
}

// Task identifies one replica of a Job during trial allocation.
type Task struct {
	JobName string   `json:"jobName"`
	Index   int      `json:"index"`
	Request Resource `json:"request"`
}

// RunningTask is an already placed task that may be selected as a reclaim victim.
type RunningTask struct {
	JobName   string
	Priority  int
	QueueName string
	TaskIndex int
	NodeName  string
	Request   Resource
}

// Allocation records a committed placement decision.
type Allocation struct {
	JobName   string `json:"jobName"`
	TaskIndex int    `json:"taskIndex"`
	NodeName  string `json:"nodeName"`
}

// Eviction records a dry-run decision to release a running task's resources.
type Eviction struct {
	JobName   string `json:"jobName"`
	TaskIndex int    `json:"taskIndex"`
	NodeName  string `json:"nodeName"`
	Reason    string `json:"reason"`
}

// AllocationPlan is the scheduler's side-effect-free output.
type AllocationPlan struct {
	Allocations   []Allocation      `json:"allocations"`
	Evictions     []Eviction        `json:"evictions"`
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
