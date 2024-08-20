package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/fujiwara/ecsta"
)

var version = "current"

func init() {
	ecsta.Version = version
}

func main() {
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, []os.Signal{os.Interrupt}...)
	defer stop()

	err := ecsta.RunCLI(ctx, os.Args[1:])
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
