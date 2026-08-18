package config

import "github.com/fulmenhq/refbolt/internal/provider"

// credentialURLs maps the env var names refbolt recognizes to their
// "where do I get one" URLs. Kept as a small map — new entries land
// alongside new credential-needing providers, and unknown env vars fall
// through to "no URL" rather than guessing.
var credentialURLs = map[string]string{
	provider.EnvJinaAPIKey:  "https://jina.ai/reader",
	provider.EnvGitHubToken: "https://github.com/settings/tokens",
}

// CredentialURL returns the "get a key" URL for the given env var, or
// the empty string when we don't have a canonical URL. Used by `init`
// stderr hints, `validate` warnings, and `catalog show` credential
// lines so the CLI surfaces the same URL in every place (FA-111).
func CredentialURL(envVar string) string {
	return credentialURLs[envVar]
}

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
		out = append(out, provider.EnvJinaAPIKey)
	}
	if p.AuthEnvVar != "" && p.AuthEnvVar != provider.EnvJinaAPIKey {
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
					if envVar == provider.EnvJinaAPIKey {
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
	for _, key := range []string{provider.EnvJinaAPIKey, provider.EnvGitHubToken} {
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
