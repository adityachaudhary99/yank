package doctor

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/adityachaudhary99/yank/internal/ui"
)

// Runner runs an install command. It mirrors backend.Runner's Run method so the
// production ExecRunner can be passed directly while tests inject a fake.
type Runner interface {
	Run(ctx context.Context, argv []string) error
}

// InstallOptions controls how Install behaves.
type InstallOptions struct {
	Yes   bool      // run without prompting
	Print bool      // only print the command, never run
	TTY   bool      // is the output an interactive terminal
	In    io.Reader // prompt input (interactive)
	Out   io.Writer // prompt + status output
}

// InstallArgv builds the package-manager command line (as argv) for tools.
// Tool binary names are translated to package names (see PackageName). The
// command is non-interactive: yank gates installation itself (via --yes or its
// own [Y/n] prompt), so the package manager must not prompt a second time.
// Returns nil for an unknown/empty manager.
func InstallArgv(manager string, tools ...string) []string {
	pkgs := make([]string, len(tools))
	for i, t := range tools {
		pkgs[i] = PackageName(t)
	}
	switch manager {
	case "apt":
		return append([]string{"sudo", "apt", "install", "-y"}, pkgs...)
	case "dnf":
		return append([]string{"sudo", "dnf", "install", "-y"}, pkgs...)
	case "pacman":
		return append([]string{"sudo", "pacman", "-S", "--noconfirm"}, pkgs...)
	case "zypper":
		return append([]string{"sudo", "zypper", "--non-interactive", "install"}, pkgs...)
	case "apk":
		// apk add is non-interactive by default.
		return append([]string{"sudo", "apk", "add"}, pkgs...)
	case "brew":
		// brew install is non-interactive by default.
		return append([]string{"brew", "install"}, pkgs...)
	default:
		return nil
	}
}

// Install installs tools via the given package manager. Behavior:
//   - Print: show the command(s) and return nil (run nothing).
//   - Yes: run immediately, no prompt.
//   - interactive TTY: ask [Y/n], run on yes; an explicit "no" returns an error.
//   - non-TTY without Yes: print the command and return an error (never blocks).
func Install(runner Runner, mgr string, tools []string, opt InstallOptions) error {
	out := opt.Out
	if out == nil {
		out = io.Discard
	}
	argv := InstallArgv(mgr, tools...)
	cmdline := strings.Join(argv, " ")

	if opt.Print {
		if argv == nil {
			for _, t := range tools {
				fmt.Fprintln(out, InstallHint(t, mgr))
			}
		} else {
			fmt.Fprintln(out, cmdline)
		}
		return nil
	}

	if argv == nil {
		return fmt.Errorf("no known package manager; install manually: %s", strings.Join(tools, " "))
	}

	if !opt.Yes {
		if !opt.TTY {
			fmt.Fprintln(out, cmdline)
			return fmt.Errorf("missing %s; re-run with --yes to auto-install, or run: %s",
				strings.Join(tools, ", "), cmdline)
		}
		if !ui.Confirm(opt.In, out, "Install "+strings.Join(tools, ", ")+"?", true) {
			return fmt.Errorf("install declined")
		}
	}

	fmt.Fprintf(out, "running: %s\n", cmdline)
	if err := runner.Run(context.Background(), argv); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}
	fmt.Fprintf(out, "+ installed %s\n", strings.Join(tools, ", "))
	return nil
}
