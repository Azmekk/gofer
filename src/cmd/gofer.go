package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/Azmekk/gofer/config"
	goferenv "github.com/Azmekk/gofer/env"
	"github.com/Azmekk/gofer/executor"
	"github.com/Azmekk/gofer/schema"
	"github.com/spf13/cobra"
)

var (
	Version    = "dev"
	configPath string
	paramFlags []string
)

var rootCmd = &cobra.Command{
	Use:     "gofer <task> [args...]",
	Short:   "Gofer - a JSON-based task runner",
	Long:    "Gofer is a simple, cross-platform task runner configured via gofer.json.",
	Version: Version,
	Args:    cobra.ArbitraryArgs,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if updateFlag, _ := cmd.Flags().GetBool("update"); updateFlag {
			if err := selfUpdate(); err != nil {
				return err
			}
			os.Exit(0)
		}
		return nil
	},
	RunE: runTask,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "gofer.json", "path or URL to config file")
	rootCmd.PersistentFlags().Bool("update", false, "update gofer to the latest version")
	rootCmd.Flags().StringArrayVarP(&paramFlags, "param", "p", nil, "task parameter in key=value format")

	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(validateCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runTask(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}

	taskRef := args[0]
	positionalArgs := args[1:]

	cfg, raw, err := config.LoadAuto(configPath)
	if err != nil {
		return err
	}

	if errs := schema.Validate(raw); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "validation error: %s\n", e)
		}
		return fmt.Errorf("config validation failed")
	}

	task, err := cfg.ResolveTask(taskRef)
	if err != nil {
		return err
	}

	params := make(map[string]string)

	for i, arg := range positionalArgs {
		if i < len(task.Params) {
			params[task.Params[i].Name] = arg
		}
	}

	for _, pf := range paramFlags {
		key, value, ok := strings.Cut(pf, "=")
		if !ok {
			return fmt.Errorf("invalid param format %q: expected key=value", pf)
		}
		params[key] = value
	}

	envVars, err := goferenv.LoadEnvFile(cfg.EnvFile)
	if err != nil {
		return fmt.Errorf("failed to load env file: %w", err)
	}
	env := goferenv.BuildEnv(envVars)

	exec := executor.New(cfg, env, params)
	return exec.RunTask(taskRef)
}
