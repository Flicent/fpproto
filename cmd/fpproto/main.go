package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/fieldpulse-prototypes/fpproto/internal/cli"
	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags "-X main.version=v1.0.0"
var version = "dev"

func main() {
	// Inject version into the cli package.
	cli.Version = version

	rootCmd := &cobra.Command{
		Use:     "fpproto",
		Short:   "Create, clone, and archive prototyping environments",
		Long:    "fpproto is a CLI tool that lets the product team create, clone, and archive prototyping environments with a single command.",
		Version: version,
		// Silence usage/errors since we handle our own output.
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Add subcommands.
	rootCmd.AddCommand(
		cli.NewSetupCmd(),
		cli.NewCreateCmd(),
		cli.NewCloneCmd(),
		cli.NewDestroyCmd(),
		cli.NewListCmd(),
		cli.NewUpdateCmd(),
		cli.NewSupabaseCmd(),
	)

	// Background version check (skip for update command itself).
	var versionNudge string
	var wg sync.WaitGroup
	if len(os.Args) > 1 && os.Args[1] != "update" && os.Args[1] != "version" && os.Args[1] != "--version" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			versionNudge = cli.CheckForUpdate(version)
		}()
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Wait for version check and print nudge if available.
	wg.Wait()
	if versionNudge != "" {
		fmt.Println()
		fmt.Println(versionNudge)
	}
}
