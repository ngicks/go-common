package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func requireCleanWorkTree(ctx context.Context) error {
	out, err := exec.CommandContext(ctx, "git", "status", "--porcelain").Output()
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}
	if len(out) > 0 {
		return errors.New("working tree is dirty; commit or stash first")
	}
	return nil
}

func requireTagAbsent(ctx context.Context, tag string) error {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", "refs/tags/"+tag)
	cmd.Stderr = nil // suppress "fatal: Needed a single revision"
	if err := cmd.Run(); err == nil {
		return fmt.Errorf("tag already exists: %s", tag)
	}
	return nil
}

func git(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return nil
}
