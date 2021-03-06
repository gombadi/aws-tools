package main

import (
	"flag"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mitchellh/cli"
)

type ASCommand struct {
	dryrun bool
	quiet  bool
	Ui     cli.Ui
}

// Help function displays detailed help for ths autostop sub command
func (c *ASCommand) Help() string {
	return `
	Description:
	Search the account for any EC2 instances with a tag key of autostop
	and in state running and stop the instance.

	Usage:
		awsgo-tools autostop [flags]
	
	Flags:
	-n - Dry Run to show which instances would be stopped but not make any changes
	-q to suppress the no instances found message
	`
}

// Synopsis function returns a string with concise details of the sub command
func (c *ASCommand) Synopsis() string {
	return "Auto stop tagged instances"
}

// Run function is the function called by the cli library to run the actual sub command code.
func (c *ASCommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("autostop", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.BoolVar(&c.dryrun, "n", false, "Dry Run")
	cmdFlags.BoolVar(&c.quiet, "q", false, "Suppress no instances found message")
	if err := cmdFlags.Parse(args); err != nil {
		return RCERR
	}

	instanceSlice := []*string{}

	// Create an EC2 service object
	// config values keys, sercet key & region read from environment
	svc := ec2.New(session.New(), &aws.Config{MaxRetries: aws.Int(10)})

	resp, err := svc.DescribeInstances(nil)

	if err != nil {
		fmt.Printf("DescribeInstances fatal error: %s\n", err)
		return RCERR
	}

	// extract the instanceId with autostop tags and state running
	for reservation := range resp.Reservations {
		for instance := range resp.Reservations[reservation].Instances {
			for tag := range resp.Reservations[reservation].Instances[instance].Tags {
				if *resp.Reservations[reservation].Instances[instance].Tags[tag].Key == "autostop" &&
					*resp.Reservations[reservation].Instances[instance].State.Name == "running" {
					// Found an instance that needs stopping
					instanceSlice = append(instanceSlice, resp.Reservations[reservation].Instances[instance].InstanceId)
				}
			}
		}
	}

	// make sure we don't stop everything on the account
	if len(instanceSlice) < 1 {
		if !c.quiet {
			fmt.Printf("No autostop instances found\n")
		}
		return RCOK
	}

	if c.dryrun == true {
		for _, i := range instanceSlice {
			fmt.Printf("Dry Run - Would have stopped instance %s\n", *i)
		}
		return RCOK
	}

	ec2sii := ec2.StopInstancesInput{InstanceIds: instanceSlice}

	stopinstanceResp, err := svc.StopInstances(&ec2sii)

	if err != nil {
		fmt.Printf("StopInstances fatal error: %s\n", err)
		return RCERR
	}

	for statechange := range stopinstanceResp.StoppingInstances {
		fmt.Printf("InstanceId: %s\t\tPrevious state: %s\t\tNew State: %s\n",
			*stopinstanceResp.StoppingInstances[statechange].InstanceId,
			*stopinstanceResp.StoppingInstances[statechange].PreviousState.Name,
			*stopinstanceResp.StoppingInstances[statechange].CurrentState.Name)
	}
	return RCOK
}
