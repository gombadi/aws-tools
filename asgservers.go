package main

import (
	"flag"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mitchellh/cli"
)

type ASGServersCommand struct {
	ASGName string
	Ui      cli.Ui
}

// Help function displays detailed help for the asgservers sub command
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

// Synopsis function returns a string with concise details of the sub command
func (c *ASGServersCommand) Synopsis() string {
	return "Display auto scale server internal ip addresses"
}

// Run function is the function called by the cli library to run the actual sub command code.
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
	svcAs := autoscaling.New(session.New(), &aws.Config{MaxRetries: aws.Int(10)})
	asgi := autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: asgNames}

	resp, err := svcAs.DescribeAutoScalingGroups(&asgi)

	if err != nil {
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
			instanceSlice = append(instanceSlice, resp.AutoScalingGroups[asGroup].Instances[instance].InstanceId)
		}
	}

	if len(instanceSlice) < 1 {
		fmt.Printf("No instances in auto scale group %s\n", *asgNames[0])
		return RCOK
	}

	ec2i := ec2.DescribeInstancesInput{InstanceIds: instanceSlice}

	// Create an EC2 service object
	// config values keys, sercet key & region read from environment
	svcEc2 := ec2.New(session.New(), &aws.Config{MaxRetries: aws.Int(10)})

	respEc2, err := svcEc2.DescribeInstances(&ec2i)

	if err != nil {
		fmt.Printf("Fatal error: DescribeAutoScalingGroups - %s\n", err)
		return RCERR
	}

	// extract the private ip address from the instance struct stored in the reservation
	for reservation := range respEc2.Reservations {
		for instance := range respEc2.Reservations[reservation].Instances {
			fmt.Printf("%s\n",
				*respEc2.Reservations[reservation].Instances[instance].PrivateIpAddress)
		}
	}

	return RCOK
}

/*

 */
