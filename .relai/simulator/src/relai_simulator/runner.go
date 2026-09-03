package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	relai "github.com/relai-ai/relai-cli/sdk/go"
)

func main() {
	projectRoot := flag.String("project-root", "", "project root")
	resultJSON := flag.String("result-json", "", "result JSON path")
	flag.Parse()

	if *projectRoot == "" {
		fmt.Fprintln(os.Stderr, "--project-root is required")
		os.Exit(2)
	}

	environment := BuildEnvironment()
	globalEvaluators, err := relai.FilterGlobalEvaluatorsForEnvironment(
		*projectRoot,
		loadGlobalEvaluators(),
		environment,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	environment.Evaluators, err = relai.CombineEvaluators(environment.Evaluators, globalEvaluators)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	runtime := relai.NewAdapterRuntime(environment.Mocks)
	adapter := any(BuildAgentAdapter(runtime))
	capabilities, err := relai.ValidateAdapterCapabilities(adapter)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(capabilities) > 1 {
		selfChecker, ok := adapter.(relai.AdapterSelfChecker)
		if !ok {
			fmt.Fprintln(os.Stderr, "adapters with optional capabilities must implement SelfCheck")
			os.Exit(1)
		}
		checks, checkErr := selfChecker.SelfCheck(context.Background(), runtime)
		if checkErr != nil {
			fmt.Fprintln(os.Stderr, checkErr)
			os.Exit(1)
		}
		if checkErr := relai.ValidateAdapterSelfCheck(capabilities, checks); checkErr != nil {
			fmt.Fprintln(os.Stderr, checkErr)
			os.Exit(1)
		}
	}
	var result relai.SimulationResult
	if environment.Target != nil && environment.Target.Type() == "component" {
		componentAdapter, ok := adapter.(relai.ComponentAdapter)
		if !ok {
			fmt.Fprintln(os.Stderr, "component environment requires BuildAgentAdapter to return a relai.ComponentAdapter")
			os.Exit(1)
		}
		result, err = relai.RunComponentEnvironmentWithRuntime(context.Background(), *projectRoot, environment, componentAdapter, runtime)
	} else {
		agentAdapter, ok := adapter.(relai.AgentAdapter)
		if !ok {
			fmt.Fprintln(os.Stderr, "agent environment requires BuildAgentAdapter to return a relai.AgentAdapter")
			os.Exit(1)
		}
		result, err = relai.RunEnvironmentWithRuntime(context.Background(), *projectRoot, environment, agentAdapter, runtime)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *resultJSON != "" {
		if err := relai.WriteSimulationResult(*resultJSON, result); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
