package commands

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	appConfig  Config
	logger     *zap.Logger
	configPath string
	verbose    bool
)

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "inspectgo",
		Short: "InspectGo CLI",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := LoadConfig(configPath)
			if err != nil {
				return err
			}
			appConfig = cfg

			if verbose {
				logger, _ = zap.NewDevelopment()
			} else {
				logger, _ = zap.NewProduction()
			}
			return nil
		},
	}

	root.PersistentFlags().StringVar(&configPath, "config", "", "config file path")
	root.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable verbose logging")

	root.AddCommand(newEvalCommand())
	root.AddCommand(newListCommand())

	return root
}
