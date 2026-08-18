package provider

// Canonical environment variable names surfaced in CLI hints and error messages.
// These are not secret values — gosec G101 flags string literals that resemble
// credential names; suppress once here and reference these constants elsewhere.
const (
	EnvJinaAPIKey  = "JINA_API_KEY" //#nosec G101 -- env var name, not a secret value
	EnvGitHubToken = "GITHUB_TOKEN" //#nosec G101 -- env var name, not a secret value
)
