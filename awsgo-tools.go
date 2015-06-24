package main

import (
	"fmt"
	"os"

	"github.com/mitchellh/cli"
)

const (
	RCOK  = 0
	RCERR = 1
)

func main() {

	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	c := cli.NewCLI("awsgo-tools", "0.0.5")
	c.Args = os.Args[1:]

	c.Commands = map[string]cli.CommandFactory{
		"asgservers": func() (cli.Command, error) {
			return &ASGServersCommand{
				Ui: &cli.ColoredUi{
					Ui: ui,
				},
			}, nil
		},
		"iamssl": func() (cli.Command, error) {
			return &IAMsslCommand{
				Ui: &cli.ColoredUi{
					Ui: ui,
				},
			}, nil
		},
		"autostop": func() (cli.Command, error) {
			return &ASCommand{
				Ui: &cli.ColoredUi{
					Ui: ui,
				},
			}, nil
		},
		"snapshot": func() (cli.Command, error) {
			return &SSCommand{
				Ui: &cli.ColoredUi{
					Ui: ui,
				},
			}, nil
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
	}

	os.Exit(exitStatus)
}

/*


*/
