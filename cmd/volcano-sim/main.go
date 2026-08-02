package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/zhouyingxiao/volcano-sim/pkg/loader"
	"github.com/zhouyingxiao/volcano-sim/pkg/scheduler"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("volcano-sim", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	nodesPath := flags.String("nodes", "", "path to nodes YAML")
	jobsPath := flags.String("jobs", "", "path to jobs YAML")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *nodesPath == "" || *jobsPath == "" {
		return fmt.Errorf("both -nodes and -jobs are required")
	}

	nodes, jobs, err := loader.Load(*nodesPath, *jobsPath)
	if err != nil {
		return err
	}
	plan := scheduler.New(nodes).Run(jobs)
	return json.NewEncoder(output).Encode(plan)
}
