package config

import "github.com/fulmenhq/refbolt/internal/provider"

// CredentialRequirement maps an env var to the providers that need it.
type CredentialRequirement struct {
	EnvVar    string
	Providers []string
	Reason    string // e.g., "jina strategy, rate-limited without key"
}

// CredentialRequirements scans a list of topics for providers that need
// API keys or tokens, grouped by env var.
func CredentialRequirements(topics []Topic) []CredentialRequirement {
	byVar := map[string]*CredentialRequirement{}

	for _, t := range topics {
		for _, p := range t.Providers {
			if !p.IsEnabled() {
				continue
			}

			// Jina strategy requires JINA_API_KEY.
			if p.FetchStrategy == provider.StrategyJina || p.FetchStrategy == provider.StrategyAuto {
				req, ok := byVar["JINA_API_KEY"]
				if !ok {
					req = &CredentialRequirement{
						EnvVar: "JINA_API_KEY",
						Reason: "jina strategy, rate-limited without key",
					}
					byVar["JINA_API_KEY"] = req
				}
				req.Providers = append(req.Providers, p.Slug)
			}

			// Providers with explicit auth_env_var.
			if p.AuthEnvVar != "" && p.AuthEnvVar != "JINA_API_KEY" {
				req, ok := byVar[p.AuthEnvVar]
				if !ok {
					req = &CredentialRequirement{
						EnvVar: p.AuthEnvVar,
						Reason: "rate-limited without token",
					}
					byVar[p.AuthEnvVar] = req
				}
				req.Providers = append(req.Providers, p.Slug)
			}
		}
	}

	// Stable order: JINA_API_KEY first, then GITHUB_TOKEN, then others.
	var result []CredentialRequirement
	for _, key := range []string{"JINA_API_KEY", "GITHUB_TOKEN"} {
		if req, ok := byVar[key]; ok {
			result = append(result, *req)
			delete(byVar, key)
		}
	}
	for _, req := range byVar {
		result = append(result, *req)
	}

	return result
}
