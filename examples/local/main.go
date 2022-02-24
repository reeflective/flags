package main

import (
	"fmt"
	"log"

	"github.com/reeflective/flags"
)

func main() {
	// The main function is used to demonstrate how to:
	// - Create and set up a command Parser (app)
	// - Bind commands to it.
	// - Further customize/adjust either commands or parser
	// - Run the application

	// 1 - Creating the parser
	client := flags.NewClient("local", flags.Default)

	// 2 - Binding commands
	client.AddCommand("list",
		"A local command demonstrating a few reflags features",
		"A longer help string used in detail help/usage output",
		"base command", // We don't use a group for this first example. See later.
		&Command{},
	)

	fmt.Println("bound commands !")

	// 4 - Running the application
	if err := client.Run(); err != nil {
		log.Fatal(err)
	}
}
