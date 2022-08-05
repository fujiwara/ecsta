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

	ctx := context.Background()
	var region, cluster string
	flag.StringVar(&cluster, "cluster", "", "ECS cluster name")
	flag.StringVar(&region, "region", os.Getenv("AWS_REGION"), "AWS region")
	flag.Parse()

	app, err := ecsta.New(ctx, region, cluster)
	if err != nil {
		panic(err)
	}
	subcommands.Register(ecsta.NewListCmd(app), "")
	subcommands.Register(ecsta.NewDescribeCmd(app), "")
	subcommands.Register(ecsta.NewStopCmd(app), "")

	os.Exit(int(subcommands.Execute(ctx)))
}
