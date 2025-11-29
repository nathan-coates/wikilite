package main

import "wikilite/cmd/commands"

func main() {
	cli := commands.NewRootCmd()

	err := cli.Execute()
	if err != nil {
		panic(err)
	}
}
