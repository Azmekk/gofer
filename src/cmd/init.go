package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Azmekk/gofer/schema"
	"github.com/spf13/cobra"
)

var (
	noSchema     bool
	remoteSchema bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a starter gofer.json",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().BoolVar(&noSchema, "no-schema", false, "omit $schema field from generated config")
	initCmd.Flags().BoolVar(&remoteSchema, "remote-schema", false, "use remote GitHub URL for $schema instead of a local file")
}

const remoteSchemaURL = "https://raw.githubusercontent.com/Azmekk/gofer/main/src/schema/gofer_schema.json"

var starterConfigBase = map[string]interface{}{
	"env_file": ".env.gofer",
	"tasks": map[string]interface{}{
		"hello": map[string]interface{}{
			"desc": "Prints a greeting",
			"params": []interface{}{
				map[string]interface{}{"name": "name", "default": "Gofer"},
			},
			"steps": []interface{}{
				map[string]interface{}{"name": "greet", "cmd": "echo 'Hello from {{.name}}!'"},
			},
		},
	},
}

func runInit(cmd *cobra.Command, args []string) error {
	if noSchema && remoteSchema {
		return fmt.Errorf("--no-schema and --remote-schema are mutually exclusive")
	}

	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("%s already exists (refusing to overwrite)", configPath)
	}

	cfg := make(map[string]interface{})
	for k, v := range starterConfigBase {
		cfg[k] = v
	}

	schemaFilePath := filepath.Join(filepath.Dir(configPath), "gofer_schema.json")

	switch {
	case noSchema:
		// no $schema, no schema file
	case remoteSchema:
		cfg["$schema"] = remoteSchemaURL
	default:
		cfg["$schema"] = "./gofer_schema.json"
		if err := os.WriteFile(schemaFilePath, schema.SchemaJSON, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", schemaFilePath, err)
		}
		fmt.Printf("Created %s\n", schemaFilePath)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", configPath, err)
	}

	fmt.Printf("Created %s\n", configPath)
	return nil
}
