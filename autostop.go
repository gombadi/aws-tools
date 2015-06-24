package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mitchellh/cli"
)

type ASCommand struct {
	quiet bool
	Ui    cli.Ui
}

func (c *ASCommand) Help() string {
	return `
	Description:
	Search the account for any EC2 instances with a tag key of autostop
	and in state running and stop the instance.

	Usage:
		awsgo-tools autostop [flags]
	
	Flags:
	-q to suppress the no instances found message
	`
}

func (c *ASCommand) Synopsis() string {
	return "Auto stop tagged instances"
}

func (c *ASCommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("autostop", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.BoolVar(&c.quiet, "q", false, "Suppress no instances found message")
	if err := cmdFlags.Parse(args); err != nil {
		return RCERR
	}

	instanceSlice := []*string{}

	// Create an EC2 service object
	// config values keys, sercet key & region read from environment
	svc := ec2.New(&aws.Config{})

	td := 499
LOOPDI:

	resp, err := svc.DescribeInstances(nil)

	// AWS retry logic
	if err != nil {
		if reqErr, ok := err.(awserr.RequestFailure); ok {
			if scErr := reqErr.StatusCode(); scErr >= 500 && scErr < 600 {
				// if retryable then double the delay for the next run
				// if time delay > 64 seconds then give up on this request & move on
				if td = td + td; td < 64000 {
					time.Sleep(time.Duration(td) * time.Millisecond)
					// loop around and try again
					goto LOOPDI
				}
			}
		}
	}

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// process SDK error
			fmt.Printf("AWS Error: %s - %s", awsErr.Code, awsErr.Message)
		}
		fmt.Printf("Fatal error: %s\n", err)
		return RCERR
	}

	// extract the instanceId with autostop tags and state running
	for reservation := range resp.Reservations {
		for instance := range resp.Reservations[reservation].Instances {
			for tag := range resp.Reservations[reservation].Instances[instance].Tags {
				if *resp.Reservations[reservation].Instances[instance].Tags[tag].Key == "autostop" &&
					*resp.Reservations[reservation].Instances[instance].State.Name == "running" {
					// Found an instance that needs stopping
					instanceSlice = append(instanceSlice, resp.Reservations[reservation].Instances[instance].InstanceID)
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

	ec2sii := ec2.StopInstancesInput{InstanceIDs: instanceSlice}

	td = 499
LOOPSI:

	stopinstanceResp, err := svc.StopInstances(&ec2sii)

	// AWS retry logic
	if err != nil {
		if reqErr, ok := err.(awserr.RequestFailure); ok {
			if scErr := reqErr.StatusCode(); scErr >= 500 && scErr < 600 {
				// if retryable then double the delay for the next run
				// if time delay > 64 seconds then give up on this request & move on
				if td = td + td; td < 64000 {
					time.Sleep(time.Duration(td) * time.Millisecond)
					// loop around and try again
					goto LOOPSI
				}
			}
		}
	}

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// process SDK error
			fmt.Printf("AWS Error: %s - %s", awsErr.Code, awsErr.Message)
		}
		fmt.Printf("Fatal error: %s\n", err)
		return RCERR
	}

	for statechange := range stopinstanceResp.StoppingInstances {
		fmt.Printf("InstanceId: %s\t\tPrevious state: %s\t\tNew State: %s\n",
			*stopinstanceResp.StoppingInstances[statechange].InstanceID,
			*stopinstanceResp.StoppingInstances[statechange].PreviousState.Name,
			*stopinstanceResp.StoppingInstances[statechange].CurrentState.Name)
	}
	return RCOK
}
