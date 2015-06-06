/*
This application will display the private ip addresses for an
auto scaling group. If no auto scale group name is supplied
then it will display all auto scale group names.

Command line options -
-a Name of the auto scale group. If not provided then all auto scale group names
will be returned.


*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func main() {

	// storage for commandline args
	var asgName string

	flag.StringVar(&asgName, "a", "", "AWS Auto scale group name")
	flag.Parse()

	// Get details of current Auto Scale Groups
	// Create empty auto scale group list and append any group name on the command line
	asgNames := []*string{}
	if len(asgName) > 0 {
		asgNames = append(asgNames, &asgName)
	}

	asgi := autoscaling.DescribeAutoScalingGroupsInput{AutoScalingGroupNames: asgNames}

	// Create an Autoscaling service object
	// config values keys, sercet key & region read from environment
	svcAs := autoscaling.New(&aws.Config{})
	resp, err := svcAs.DescribeAutoScalingGroups(&asgi)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// process SDK error
			fmt.Printf("AWS Error: %s - %s", awsErr.Code, awsErr.Message)
		}
		log.Fatalf("Fatal error: DescribeAutoScalingGroups - %s\n", err)
	}

	// if no asg name provided then display current asg names and exit
	if len(asgName) == 0 {
		fmt.Println("No Autoscaling Group Name provided. Current Groups:")

		for asGroup := range resp.AutoScalingGroups {
			fmt.Printf("%s\n", *resp.AutoScalingGroups[asGroup].AutoScalingGroupName)
		}
		fmt.Println("")

		os.Exit(0)
	}

	if len(resp.AutoScalingGroups) < 1 {
		fmt.Printf("No Auto Scale Group info found for %s.\n", *asgNames[0])
		os.Exit(1)
	}

	instanceSlice := []*string{}

	// extract the instanceid's from the auto scale details and append to a slice
	for asGroup := range resp.AutoScalingGroups {
		for instance := range resp.AutoScalingGroups[asGroup].Instances {
			instanceSlice = append(instanceSlice, resp.AutoScalingGroups[asGroup].Instances[instance].InstanceID)
		}
	}

	if len(instanceSlice) < 1 {
		fmt.Printf("No instances in auto scale group %s.\n", *asgNames[0])
		os.Exit(1)
	}

	ec2i := ec2.DescribeInstancesInput{InstanceIDs: instanceSlice}

	// Create an EC2 service object
	// config values keys, sercet key & region read from environment
	svcEc2 := ec2.New(&aws.Config{})
	respEc2, err := svcEc2.DescribeInstances(&ec2i)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// process SDK error
			fmt.Printf("AWS Error: %s - %s", awsErr.Code, awsErr.Message)
		}
		log.Fatalf("Fatal error: DescribeAutoScalingGroups - %s\n", err)
	}

	// extract the private ip address from the instance struct stored in the reservation
	for reservation := range respEc2.Reservations {
		for instance := range respEc2.Reservations[reservation].Instances {
			fmt.Printf("%s\n",
				*respEc2.Reservations[reservation].Instances[instance].PrivateIPAddress)
		}
	}

}
