package main

import (
	"log"
	"os"

	"github.com/fujiwara/ecsta"
)

var version = "current"

func main() {
	cliApp := ecsta.NewCLI()
	cliApp.Version = version
	if err := cliApp.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
