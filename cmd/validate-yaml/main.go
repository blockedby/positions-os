package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("No files to check.")
		os.Exit(0)
	}

	failed := false
	for _, path := range os.Args[1:] {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("❌ Failed to read %s: %v\n", path, err)
			failed = true
			continue
		}

		var temp interface{}
		if err := yaml.Unmarshal(data, &temp); err != nil {
			fmt.Printf("❌ Invalid YAML in %s: %v\n", path, err)
			failed = true
			continue
		}
		fmt.Printf("✅ %s is valid\n", path)
	}

	if failed {
		os.Exit(1)
	}
}
