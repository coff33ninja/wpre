package main

import (
	"os"

	"wpre/internal/orchestrator"
)

func main() {
	os.Exit(orchestrator.Run())
}
