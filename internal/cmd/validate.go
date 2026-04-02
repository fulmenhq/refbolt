package cmd

import (
	"fmt"
	"os"

	"github.com/fulmenhq/refbolt/internal/config"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a providers config file against the schema",
	Long: `Validate the providers config file against the embedded JSON Schema.

Checks YAML syntax, required fields, strategy-specific requirements,
and provider slug validity. Exits 1 on errors, 0 on success.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Config loading with strict validation is handled by PersistentPreRunE.
		// If we get here, validation passed.

		topics := config.Topics()
		totalProviders := 0
		enabledProviders := 0
		for _, t := range topics {
			for _, p := range t.Providers {
				totalProviders++
				if p.IsEnabled() {
					enabledProviders++
				}
			}
		}

		disabledProviders := totalProviders - enabledProviders
		fmt.Printf("Valid config: %d topics, %d providers (%d enabled, %d disabled)\n",
			len(topics), totalProviders, enabledProviders, disabledProviders)
		fmt.Printf("Config source: %s\n", config.ConfigUsed())

		// Advisory warnings for missing credential env vars (exit 0).
		creds := config.CredentialRequirements(topics)
		for _, c := range creds {
			if os.Getenv(c.EnvVar) == "" {
				fmt.Fprintf(os.Stderr, "warning: %s not set — %s (%s)\n",
					c.EnvVar, joinSlugs(c.Providers), c.Reason)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
