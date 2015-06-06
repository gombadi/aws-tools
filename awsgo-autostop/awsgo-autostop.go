/*
This application will stop any instance on the account if it has a tag autostop
and if it is running.

Command line options
-q Suppress no instances found message


*/

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func main() {

	instanceSlice := []*string{}

	// storage for commandline args
	var quiet bool

	flag.BoolVar(&quiet, "q", false, "Suppress no instances found message")
	flag.Parse()

	// Create an EC2 service object
	// config values keys, sercet key & region read from environment
	svc := ec2.New(&aws.Config{})
	resp, err := svc.DescribeInstances(nil)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// process SDK error
			fmt.Printf("AWS Error: %s - %s", awsErr.Code, awsErr.Message)
		}
		log.Fatalf("Fatal error: %s\n", err)
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
		if !quiet {
			fmt.Printf("No autostop instances found\n")
		}
		os.Exit(0)
	}

	ec2sii := ec2.StopInstancesInput{InstanceIDs: instanceSlice}

	stopinstanceResp, err := svc.StopInstances(&ec2sii)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// process SDK error
			fmt.Printf("AWS Error: %s - %s", awsErr.Code, awsErr.Message)
		}
		log.Fatalf("Fatal error: %s\n", err)
	}

	for statechange := range stopinstanceResp.StoppingInstances {
		fmt.Printf("InstanceId: %s\t\tPrevious state: %s\t\tNew State: %s\n",
			*stopinstanceResp.StoppingInstances[statechange].InstanceID,
			*stopinstanceResp.StoppingInstances[statechange].PreviousState.Name,
			*stopinstanceResp.StoppingInstances[statechange].CurrentState.Name)
	}

}
