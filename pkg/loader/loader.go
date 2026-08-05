// Package loader converts YAML scenario files into validated scheduler input.
package loader

import (
	"fmt"
	"os"

	"github.com/zhouyingxiao/volcano-sim/pkg/api"
	"gopkg.in/yaml.v3"
)

type nodesDocument struct {
	Nodes []api.Node `yaml:"nodes"`
}

type jobsDocument struct {
	Jobs []api.Job `yaml:"jobs"`
}

type queuesDocument struct {
	Queues []api.Queue `yaml:"queues"`
}

// LoadQueues reads queues and initializes their in-memory allocation vectors.
func LoadQueues(path string) ([]api.Queue, error) {
	var document queuesDocument
	if err := decode(path, &document); err != nil {
		return nil, fmt.Errorf("load queues: %w", err)
	}
	seen := make(map[string]struct{}, len(document.Queues))
	for index := range document.Queues {
		queue := &document.Queues[index]
		if queue.Name == "" || queue.Weight <= 0 {
			return nil, fmt.Errorf("queue has invalid name or weight")
		}
		if _, exists := seen[queue.Name]; exists {
			return nil, fmt.Errorf("queue %q is duplicated", queue.Name)
		}
		seen[queue.Name] = struct{}{}
		if err := validateResource(queue.Name, "queue", queue.Capability); err != nil {
			return nil, err
		}
		queue.Allocated = api.NewResource(nil)
	}
	return document.Queues, nil
}

// Load reads two scenario files and returns nodes with initialized idle capacity.
func Load(nodesPath, jobsPath string) ([]api.Node, []api.Job, error) {
	var nodeDocument nodesDocument
	if err := decode(nodesPath, &nodeDocument); err != nil {
		return nil, nil, fmt.Errorf("load nodes: %w", err)
	}
	for index := range nodeDocument.Nodes {
		node := &nodeDocument.Nodes[index]
		if err := validateResource(node.Name, "node", node.Capacity); err != nil {
			return nil, nil, err
		}
		node.Idle = api.NewResource(node.Capacity)
	}

	var jobDocument jobsDocument
	if err := decode(jobsPath, &jobDocument); err != nil {
		return nil, nil, fmt.Errorf("load jobs: %w", err)
	}
	seenJobs := make(map[string]struct{}, len(jobDocument.Jobs))
	for index := range jobDocument.Jobs {
		job := &jobDocument.Jobs[index]
		if job.Queue == "" {
			job.Queue = "default"
		}
		if job.Name == "" {
			return nil, nil, fmt.Errorf("job has an empty name")
		}
		if _, exists := seenJobs[job.Name]; exists {
			return nil, nil, fmt.Errorf("job %q is duplicated", job.Name)
		}
		seenJobs[job.Name] = struct{}{}
		if job.MinAvailable <= 0 || job.MinAvailable > job.Replicas {
			return nil, nil, fmt.Errorf("job %q has invalid minAvailable %d for %d replicas", job.Name, job.MinAvailable, job.Replicas)
		}
		if job.BatchSize == 0 {
			job.BatchSize = job.Replicas
		}
		if err := validateResource(job.Name, "job", job.Request); err != nil {
			return nil, nil, err
		}
		if err := validateTopology(job); err != nil {
			return nil, nil, err
		}
		job.Allocated = api.NewResource(nil)
	}

	return nodeDocument.Nodes, jobDocument.Jobs, nil
}

func validateTopology(job *api.Job) error {
	if job.Topology == nil || job.Topology.SameFabric == "" {
		return nil
	}
	if job.Topology.SameFabric != "Required" {
		return fmt.Errorf("job %q has unsupported sameFabric value %q", job.Name, job.Topology.SameFabric)
	}
	if job.Topology.GPUModel == "" {
		return fmt.Errorf("job %q requires gpuModel when sameFabric is Required", job.Name)
	}
	return nil
}

func decode(path string, target any) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(contents, target); err != nil {
		return err
	}
	return nil
}

func validateResource(name, kind string, resource api.Resource) error {
	if name == "" {
		return fmt.Errorf("%s has an empty name", kind)
	}
	if len(resource) == 0 {
		return fmt.Errorf("%s %q has no resources", kind, name)
	}
	for resourceName, quantity := range resource {
		if quantity <= 0 {
			return fmt.Errorf("%s %q has non-positive %s quantity %v", kind, name, resourceName, quantity)
		}
	}
	return nil
}
