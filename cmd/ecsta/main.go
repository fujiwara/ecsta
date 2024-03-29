package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/fujiwara/ecsta"
)

var version = "current"

func init() {
	ecsta.Version = version
}

func main() {
	ctx := context.TODO()
	ctx, stop := signal.NotifyContext(ctx, []os.Signal{os.Interrupt}...)
	defer stop()

	err := ecsta.RunCLI(ctx, os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
}
