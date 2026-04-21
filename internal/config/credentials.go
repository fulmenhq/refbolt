package config

import "github.com/fulmenhq/refbolt/internal/provider"

// CredentialRequirement maps an env var to the providers that need it.
type CredentialRequirement struct {
	EnvVar    string
	Providers []string
	Reason    string // e.g., "jina strategy, rate-limited without key"
}

// ProviderCredentials returns the env var names a single provider will consult
// at runtime. Order is stable: JINA_API_KEY (if applicable) first, then any
// explicit auth_env_var. Shared by CredentialRequirements (sync/init) and the
// catalog browse command (FA-101) so the heuristic lives in one place.
func ProviderCredentials(p provider.ProviderConfig) []string {
	var out []string
	if p.FetchStrategy == provider.StrategyJina || p.FetchStrategy == provider.StrategyAuto {
		out = append(out, "JINA_API_KEY")
	}
	if p.AuthEnvVar != "" && p.AuthEnvVar != "JINA_API_KEY" {
		out = append(out, p.AuthEnvVar)
	}
	return out
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

			for _, envVar := range ProviderCredentials(p) {
				req, ok := byVar[envVar]
				if !ok {
					reason := "rate-limited without token"
					if envVar == "JINA_API_KEY" {
						reason = "jina strategy, rate-limited without key"
					}
					req = &CredentialRequirement{
						EnvVar: envVar,
						Reason: reason,
					}
					byVar[envVar] = req
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
