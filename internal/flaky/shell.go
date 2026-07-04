package flaky

import "strings"

// QuoteShellArg wraps s in single quotes, escaping any embedded single
// quotes, so it can be safely reassembled into a shell command line without
// the shell re-splitting or reinterpreting it.
func QuoteShellArg(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// JoinShellCommand rebuilds a single `sh -c` command string from a slice of
// argv-style arguments, quoting each one individually. This preserves the
// original argument boundaries (including arguments containing spaces)
// while still running the result through a real shell, so users can embed
// shell syntax (pipes, &&, redirection) by passing it as a single
// pre-quoted argument.
func JoinShellCommand(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = QuoteShellArg(a)
	}
	return strings.Join(quoted, " ")
}
