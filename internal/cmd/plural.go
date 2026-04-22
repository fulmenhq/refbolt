package cmd

// pluralize returns singular for n==1, plural otherwise. Shared across
// `init`, `validate`, and `catalog` so count-dependent output agrees on
// the singular/plural rule (FA-111 item #2). No locale handling — ASCII
// English only, which matches the rest of the CLI's output strings.
func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
