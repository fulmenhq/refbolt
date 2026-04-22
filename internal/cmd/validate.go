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
		fmt.Printf("Valid config: %d %s, %d %s (%d enabled, %d disabled)\n",
			len(topics), pluralize(len(topics), "topic", "topics"),
			totalProviders, pluralize(totalProviders, "provider", "providers"),
			enabledProviders, disabledProviders)
		fmt.Printf("Config source: %s\n", config.ConfigUsed())

		// Advisory warnings for missing credential env vars (exit 0).
		// When the env var has a known "get a key" URL, surface it so
		// CLI users don't need to hunt the README for it (FA-111 item #3).
		creds := config.CredentialRequirements(topics)
		for _, c := range creds {
			if os.Getenv(c.EnvVar) == "" {
				if url := config.CredentialURL(c.EnvVar); url != "" {
					fmt.Fprintf(os.Stderr, "warning: %s not set — %s (%s). Get a key: %s\n",
						c.EnvVar, joinSlugs(c.Providers), c.Reason, url)
				} else {
					fmt.Fprintf(os.Stderr, "warning: %s not set — %s (%s)\n",
						c.EnvVar, joinSlugs(c.Providers), c.Reason)
				}
			}
		}

		// Zero-config path: the embedded catalog is fine to sync against,
		// but users with no providers.yaml should know how to customize
		// (FA-111 item #11). Only emit when no user config was loaded —
		// `config.ConfigUsed()` already shows which source is in play.
		if config.ConfigUsed() == "(embedded catalog)" {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, "Tip: run 'refbolt init --all --output providers.yaml' to customize")
			fmt.Fprintln(os.Stderr, "     which providers sync.")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}
