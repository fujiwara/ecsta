package main

import (
	"context"
	"log"
	"os"

	"github.com/fujiwara/ecsta"
)

var version = "current"

func init() {
	ecsta.Version = version
}

func main() {
	ctx := context.TODO()
	err := ecsta.RunCLI(ctx, os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
}
