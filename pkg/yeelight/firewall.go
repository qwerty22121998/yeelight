package yeelight

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
)

// EnsureFirewallPort best-effort opens the given UDP port for inbound SSDP
// discovery replies. It currently supports ufw only.
//
// It is non-fatal: any failure (no ufw, no privilege, command error) is logged
// and returned, never blocking discovery. When not running as root it escalates
// via pkexec, which raises a graphical auth prompt — so call it once, not per
// scan. The ufw rule is idempotent; re-adding the same rule is harmless.
func EnsureFirewallPort(ctx context.Context, port int) error {
	ufw, err := exec.LookPath("ufw")
	if err != nil {
		slog.WarnContext(ctx, "ufw not found, skipping firewall setup", "error", err)
		return err
	}

	args := []string{ufw, "allow", fmt.Sprintf("%d/udp", port), "comment", "yeelight SSDP discovery replies"}
	if os.Geteuid() != 0 {
		pkexec, err := exec.LookPath("pkexec")
		if err != nil {
			slog.WarnContext(ctx, "not root and pkexec not found, skipping firewall setup", "error", err)
			return err
		}
		args = append([]string{pkexec}, args...)
	}

	out, err := exec.CommandContext(ctx, args[0], args[1:]...).CombinedOutput()
	if err != nil {
		slog.ErrorContext(ctx, "failed to open firewall port", "port", port, "output", string(out), "error", err)
		return err
	}
	slog.InfoContext(ctx, "firewall port ensured", "port", port)
	return nil
}
