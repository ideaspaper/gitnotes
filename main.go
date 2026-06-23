// Command gitnotes manages git notes as reviewable, CSV-backed comments on
// HEAD, and posts them as pull/merge-request review comments.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gitnotes/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cli.Run(ctx, os.Stdout, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
