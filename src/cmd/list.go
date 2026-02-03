package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Azmekk/gofer/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available tasks",
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, _, err := config.LoadAuto(configPath)
	if err != nil {
		return err
	}

	// Partition tasks by group
	ungrouped := make(map[string]config.Task)
	grouped := make(map[string]map[string]config.Task)

	for name, task := range cfg.Tasks {
		if task.Group == "" {
			ungrouped[name] = task
		} else {
			if grouped[task.Group] == nil {
				grouped[task.Group] = make(map[string]config.Task)
			}
			grouped[task.Group][name] = task
		}
	}

	// Print ungrouped tasks first
	if len(ungrouped) > 0 {
		names := sortedKeys(ungrouped)
		for _, name := range names {
			fmt.Printf("  %s%s - %s\n", name, formatParams(ungrouped[name].Params), ungrouped[name].Desc)
		}
		if len(grouped) > 0 {
			fmt.Println()
		}
	}

	// Print grouped tasks
	groupNames := make([]string, 0, len(grouped))
	for g := range grouped {
		groupNames = append(groupNames, g)
	}
	sort.Strings(groupNames)

	for i, gName := range groupNames {
		fmt.Printf("%s:\n", gName)
		names := sortedKeys(grouped[gName])
		for _, name := range names {
			task := grouped[gName][name]
			fmt.Printf("  %s%s - %s\n", name, formatParams(task.Params), task.Desc)
		}
		if i < len(groupNames)-1 {
			fmt.Println()
		}
	}

	return nil
}

func formatParams(params []config.Param) string {
	var hints []string
	for _, p := range params {
		if p.Default != nil {
			hints = append(hints, fmt.Sprintf("%s=%s", p.Name, *p.Default))
		} else {
			hints = append(hints, fmt.Sprintf("<%s>", p.Name))
		}
	}
	if len(hints) > 0 {
		return " [" + strings.Join(hints, ", ") + "]"
	}
	return ""
}

func sortedKeys(m map[string]config.Task) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
