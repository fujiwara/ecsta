package main

import (
	"context"
	"flag"
	"os"

	"github.com/fujiwara/ecsta"
	"github.com/google/subcommands"
)

func main() {
	subcommands.Register(subcommands.FlagsCommand(), "")

	ctx := context.Background()
	var region, cluster string
	flag.StringVar(&cluster, "cluster", os.Getenv("ECS_CLUSTER"), "ECS cluster name")
	flag.StringVar(&region, "region", os.Getenv("AWS_REGION"), "AWS region")
	flag.Parse()

	app, err := ecsta.New(ctx, region, cluster)
	if err != nil {
		panic(err)
	}
	subcommands.Register(ecsta.NewListCmd(app), "")
	subcommands.Register(ecsta.NewDescribeCmd(app), "")
	subcommands.Register(ecsta.NewExecCmd(app), "")
	subcommands.Register(ecsta.NewPortforwardCmd(app), "")
	subcommands.Register(ecsta.NewStopCmd(app), "")
	subcommands.Register(ecsta.NewConfigureCmd(app), "")

	os.Exit(int(subcommands.Execute(ctx)))
}
