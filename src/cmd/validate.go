package cmd

import (
	"fmt"
	"os"

	"github.com/Azmekk/gofer/schema"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate gofer.json",
	RunE:  runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", configPath, err)
	}

	errs := schema.Validate(data)
	if len(errs) == 0 {
		fmt.Println("Configuration is valid.")
		return nil
	}

	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "  - %s\n", e)
	}
	return fmt.Errorf("found %d validation error(s)", len(errs))
}
