/*
This is a small program to display basic information about AWS instances.
Mainly used to understand the AWS SDK data structures and calls

*/
package main

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var emptyString = ""

func main() {

	// Create an EC2 service object
	// Config details Keys, secret keys and region will be read from environment
	svc := ec2.New(&aws.Config{})

	// Call the DescribeInstances Operation
	resp, err := svc.DescribeInstances(nil)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// process SDK error
			fmt.Printf("AWS Error: %s - %s", awsErr.Code, awsErr.Message)
		}
		log.Fatalf("Fatal error: %s\n", err)
	}

	var instanceName *string

	// extract the private ip address from the instance struct stored in the reservation
	for reservation := range resp.Reservations {
		for instance, inst := range resp.Reservations[reservation].Instances {
			for tag := range resp.Reservations[reservation].Instances[instance].Tags {
				if *resp.Reservations[reservation].Instances[instance].Tags[tag].Key == "Name" {
					instanceName = resp.Reservations[reservation].Instances[instance].Tags[tag].Value
					break
				} else {
					instanceName = aws.String("Unknown")
				}
			}

			fmt.Printf("Instance: %s\tName: %s\tstate: %s\tType: %s\tAVzone: %s\tPublicIP: %s\tPrivateIP: %s\n",
				*(chkStringValue(inst.InstanceID)),
				*(chkStringValue(instanceName)),
				*(chkStringValue(inst.State.Name)),
				*(chkStringValue(inst.InstanceType)),
				*(chkStringValue(inst.Placement.AvailabilityZone)),
				*(chkStringValue(inst.PublicIPAddress)),
				*(chkStringValue(inst.PrivateIPAddress)))

			instanceName = &emptyString
		}
	}

}

func chkStringValue(s *string) *string {
	if s == nil {
		s = &emptyString
	}
	return s
}
