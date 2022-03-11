package main

import (
	"context"
	"flag"
	"os"

	"github.com/fujiwara/ecsta"
	"github.com/google/subcommands"
)

func main() {
	subcommands.Register(subcommands.HelpCommand(), "")
	subcommands.Register(subcommands.FlagsCommand(), "")
	subcommands.Register(subcommands.CommandsCommand(), "")

	app, err := ecsta.New(context.Background(), os.Getenv("AWS_REGION"))
	if err != nil {
		panic(err)
	}

	subcommands.Register(ecsta.NewTasksCmd(app), "")

	flag.Parse()
	ctx := context.Background()
	os.Exit(int(subcommands.Execute(ctx)))
}
