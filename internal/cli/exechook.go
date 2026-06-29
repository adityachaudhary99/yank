package cli

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// splitArgs splits a command string into argv, honoring single and double quotes
// (so paths and arguments with spaces survive). It does NOT interpret a shell —
// no pipes, globs, or variable expansion — which keeps --exec predictable and
// free of shell-injection surprises. Use a wrapper script for shell features.
func splitArgs(s string) ([]string, error) {
	var args []string
	var cur strings.Builder
	inWord := false
	var quote rune
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
			inWord = true
		case r == '\'' || r == '"':
			quote = r
			inWord = true
		case r == ' ' || r == '\t':
			if inWord {
				args = append(args, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unbalanced quote in --exec command")
	}
	if inWord {
		args = append(args, cur.String())
	}
	return args, nil
}

// buildExecArgv turns the --exec command and a downloaded file path into argv.
// Each "{}" token is replaced with the path; if no token contains "{}", the path
// is appended as a final argument (so both `--exec 'tar xzf {}'` and
// `--exec sha256sum` work).
func buildExecArgv(command, path string) ([]string, error) {
	parts, err := splitArgs(command)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("--exec command is empty")
	}
	substituted := false
	for i, p := range parts {
		if strings.Contains(p, "{}") {
			parts[i] = strings.ReplaceAll(p, "{}", path)
			substituted = true
		}
	}
	if !substituted {
		parts = append(parts, path)
	}
	return parts, nil
}

// runExecHook runs the --exec command after a successful download, with the
// downloaded file's path substituted in. It's a no-op when no command is set or
// the path is unknown (e.g. stdout streaming, or an auto-named dispatch result).
// The hook's stdout/stderr stream to out. A build error is returned; a non-zero
// exit from the hook is wrapped so the caller can report it.
func runExecHook(ctx context.Context, command, path string, out io.Writer) error {
	if command == "" || path == "" || path == "-" {
		return nil
	}
	argv, err := buildExecArgv(command, path)
	if err != nil {
		return err
	}
	c := exec.CommandContext(ctx, argv[0], argv[1:]...)
	c.Stdout, c.Stderr = out, out
	if err := c.Run(); err != nil {
		return fmt.Errorf("%s: %w", argv[0], err)
	}
	return nil
}
