package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Azmekk/gofer/schema"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate gofer.json",
	RunE:  runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	var data []byte
	var err error
	if strings.HasPrefix(configPath, "http://") || strings.HasPrefix(configPath, "https://") {
		resp, fetchErr := http.Get(configPath)
		if fetchErr != nil {
			return fmt.Errorf("failed to fetch remote config: %w", fetchErr)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to fetch remote config: %s", resp.Status)
		}
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read remote config: %w", err)
		}
	} else {
		data, err = os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", configPath, err)
		}
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
