package main

import (
	"log"
	"os"

	"github.com/fujiwara/ecsta"
)

func main() {
	cliApp := ecsta.NewCLI()
	if err := cliApp.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
