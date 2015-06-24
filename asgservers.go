package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mitchellh/cli"
)

type ASGServersCommand struct {
	ASGName string
	Ui      cli.Ui
}

func (c *ASGServersCommand) Help() string {
	return `
	Description:
	Display the internal IP addresses of auto scaled servers

	Usage:
		awsgo-tools asgservers [flags]

	Flags:
	--asg-name <auto scale group> to display ip addresses for that group
	No flags to display a list of auto scale group names
	`
}

func (c *ASGServersCommand) Synopsis() string {
	return "Display auto scale ip addresses"
}

func (c *ASGServersCommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("asgservers", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.StringVar(&c.ASGName, "asg-name", "", "Auto scale group name or blank to list all groups")
	if err := cmdFlags.Parse(args); err != nil {
		return RCERR
	}

	// Get details of current Auto Scale Groups
	// Create empty auto scale group list and append any group name on the command line
	asgNames := []*string{}
	if len(c.ASGName) > 0 {
		asgNames = append(asgNames, &c.ASGName)
	}

	// Create an Autoscaling service object
	// config values keys, sercet key & region read from environment
	svcAs := autoscaling.New(&aws.Config{})
	asgi := autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: asgNames}

	td := 499
LOOPDASG:

	resp, err := svcAs.DescribeAutoScalingGroups(&asgi)

	// AWS retry logic
	if err != nil {
		if reqErr, ok := err.(awserr.RequestFailure); ok {
			if scErr := reqErr.StatusCode(); scErr >= 500 && scErr < 600 {
				// if retryable then double the delay for the next run
				// if time delay > 64 seconds then give up on this request & move on
				if td = td + td; td < 64000 {
					time.Sleep(time.Duration(td) * time.Millisecond)
					// loop around and try again
					goto LOOPDASG
				}
			}
		}
	}

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// process SDK error
			fmt.Printf("AWS Error: %s - %s", awsErr.Code, awsErr.Message)
		}
		fmt.Printf("Fatal error: DescribeAutoScalingGroups - %s\n", err)
		return RCERR
	}

	// if no asg name provided then display current asg names and exit
	if len(c.ASGName) == 0 {
		fmt.Println("No Autoscaling Group Name provided. Current Groups:")

		for asGroup := range resp.AutoScalingGroups {
			fmt.Printf("%s\n", *resp.AutoScalingGroups[asGroup].AutoScalingGroupName)
		}
		fmt.Println("")
		return RCOK
	}

	if len(resp.AutoScalingGroups) < 1 {
		fmt.Printf("No Auto Scale Group info found for %s\n", *asgNames[0])
		return RCOK
	}

	instanceSlice := []*string{}

	// extract the instanceid's from the auto scale details and append to a slice
	for asGroup := range resp.AutoScalingGroups {
		for instance := range resp.AutoScalingGroups[asGroup].Instances {
			instanceSlice = append(instanceSlice, resp.AutoScalingGroups[asGroup].Instances[instance].InstanceID)
		}
	}

	if len(instanceSlice) < 1 {
		fmt.Printf("No instances in auto scale group %s\n", *asgNames[0])
		return RCOK
	}

	ec2i := ec2.DescribeInstancesInput{InstanceIDs: instanceSlice}

	// Create an EC2 service object
	// config values keys, sercet key & region read from environment
	svcEc2 := ec2.New(&aws.Config{})

	td = 499
LOOPDI:

	respEc2, err := svcEc2.DescribeInstances(&ec2i)

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
		fmt.Printf("Fatal error: DescribeAutoScalingGroups - %s\n", err)
		return RCERR
	}

	// extract the private ip address from the instance struct stored in the reservation
	for reservation := range respEc2.Reservations {
		for instance := range respEc2.Reservations[reservation].Instances {
			fmt.Printf("%s\n",
				*respEc2.Reservations[reservation].Instances[instance].PrivateIPAddress)
		}
	}

	return RCOK
}

/*

*/
