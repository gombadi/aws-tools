// The awsgo-tools package brings together a number of cli apps that can be used
// to manage and monitor Amazon Web Services resources
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

	c := cli.NewCLI("awsgo-tools", "0.0.9")
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
		"reserved-report": func() (cli.Command, error) {
			return &RRCommand{
				Ui: &cli.ColoredUi{
					Ui: ui,
				},
			}, nil
		},
		"ami-cleanup": func() (cli.Command, error) {
			return &AMICommand{
				Ui: &cli.ColoredUi{
					Ui: ui,
				},
			}, nil
		},
		"audit": func() (cli.Command, error) {
			return &AuditCommand{
				Ui: &cli.ColoredUi{
					Ui: ui,
				},
			}, nil
		},
		"s3info": func() (cli.Command, error) {
			return &S3infoCommand{
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
