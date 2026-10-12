package main

import airline "github.com/relai-ai/airline-customer-support-agent-go/airline"

func main() {
 // Resolve the selected entrypoint without constructing a credentialed client.
 _ = airline.NewAgent
 _ = (*airline.Agent).StreamResponse
}
