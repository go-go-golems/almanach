package app

import (
	almanachdoc "github.com/go-go-golems/almanach/internal/app/doc"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/spf13/cobra"
)

func NewRootCommand(version string) (*cobra.Command, error) {
	rootCmd := &cobra.Command{
		Use:     "almanach-render-service",
		Short:   "Render and print Almanach Studio thermal pages",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.InitLoggerFromCobra(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunServe(cmd.Context(), LoadConfig())
		},
	}

	if err := logging.AddLoggingSectionToRootCommand(rootCmd, "almanach-render-service"); err != nil {
		return nil, err
	}

	helpSystem := help.NewHelpSystem()
	if err := almanachdoc.AddDocToHelpSystem(helpSystem); err != nil {
		return nil, err
	}
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)

	rootCmd.AddCommand(NewServeCommand())

	renderCmd, err := newRenderCommand()
	if err != nil {
		return nil, err
	}
	if err := addGlazedCommand(rootCmd, renderCmd); err != nil {
		return nil, err
	}

	inspectCmd, err := newInspectCommand()
	if err != nil {
		return nil, err
	}
	if err := addGlazedCommand(rootCmd, inspectCmd); err != nil {
		return nil, err
	}

	printCmd, err := newPrintCommand()
	if err != nil {
		return nil, err
	}
	if err := addGlazedCommand(rootCmd, printCmd); err != nil {
		return nil, err
	}

	bleProvisionCmd, err := newBLEProvisionCommand()
	if err != nil {
		return nil, err
	}
	if err := addGlazedCommand(rootCmd, bleProvisionCmd); err != nil {
		return nil, err
	}

	return rootCmd, nil
}

func addGlazedCommand(root *cobra.Command, command cmds.Command) error {
	cobraCmd, err := cli.BuildCobraCommandFromCommand(command,
		cli.WithParserConfig(cli.CobraParserConfig{
			AppName:           "almanach-render-service",
			ShortHelpSections: []string{schema.DefaultSlug},
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	)
	if err != nil {
		return err
	}
	root.AddCommand(cobraCmd)
	return nil
}
