package main

import (
	"context"
	"log"
	"os"

	"github.com/fujiwara/ecsta"
)

var version = "current"

func main() {
	ctx := context.TODO()
	cliApp := ecsta.NewCLI()
	cliApp.Version = version
	if err := cliApp.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
