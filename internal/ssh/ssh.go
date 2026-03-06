package ssh

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultTimeout = 5 * time.Second
const tmuxLSFormat = "#{session_name}|#{session_windows}|#{session_created}|#{session_attached}"

func isLocal(host string) bool {
	return host == "local" || host == "localhost"
}

func sshArgs(host, keyFile string) []string {
	args := []string{"-o", "ConnectTimeout=5"}
	if keyFile != "" {
		args = append(args, "-i", keyFile)
	}
	args = append(args, host)
	return args
}

func runCommand(ctx context.Context, host, keyFile, remoteCmd string) *exec.Cmd {
	if isLocal(host) {
		return exec.CommandContext(ctx, "sh", "-c", remoteCmd)
	}
	args := append(sshArgs(host, keyFile), remoteCmd)
	return exec.CommandContext(ctx, "ssh", args...)
}

func ListSessions(ctx context.Context, host, keyFile string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	remoteCmd := fmt.Sprintf("tmux ls -F '%s'", tmuxLSFormat)
	cmd := runCommand(ctx, host, keyFile, remoteCmd)
	out, err := cmd.CombinedOutput()
	outStr := strings.TrimSpace(string(out))
	if err != nil {
		if strings.Contains(outStr, "no server running") ||
			strings.Contains(outStr, "no sessions") ||
			strings.Contains(outStr, "error connecting") {
			return "", nil
		}
		if outStr != "" {
			return "", fmt.Errorf("%s", outStr)
		}
		return "", err
	}
	return outStr, nil
}

func NewSession(ctx context.Context, host, keyFile, name string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	remoteCmd := fmt.Sprintf("tmux new-session -d -s '%s'", name)
	cmd := runCommand(ctx, host, keyFile, remoteCmd)
	return cmd.Run()
}

func KillSession(ctx context.Context, host, keyFile, name string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	remoteCmd := fmt.Sprintf("tmux kill-session -t '%s'", name)
	cmd := runCommand(ctx, host, keyFile, remoteCmd)
	return cmd.Run()
}

func DetachSession(ctx context.Context, host, keyFile, name string) error {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	remoteCmd := fmt.Sprintf("tmux detach-client -s '%s'", name)
	cmd := runCommand(ctx, host, keyFile, remoteCmd)
	return cmd.Run()
}

func AttachCommand(host, keyFile, session string) *exec.Cmd {
	// Bind Ctrl-q to detach, set status hint, then attach — single shell command.
	// Using ; so bind/set run even if they're redundant on repeat attaches.
	attachCmd := fmt.Sprintf(
		"tmux bind -n C-q detach-client; "+
			"tmux set -t '%s' status-right '[C-q] detach'; "+
			"exec tmux attach -t '%s'",
		session, session,
	)

	if isLocal(host) {
		return exec.Command("sh", "-c", attachCmd)
	}

	args := []string{"-t"}
	if keyFile != "" {
		args = append(args, "-i", keyFile)
	}
	args = append(args, host, attachCmd)
	return exec.Command("ssh", args...)
}
