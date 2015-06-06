/*
This application will output details in CSV format of all active ec2 & RDS reserved instances.
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
)

func main() {

	// storage for commandline args
	var header, printEmpty bool
	var account string

	flag.BoolVar(&header, "h", false, "Produce CSV Headers and exit")
	flag.BoolVar(&printEmpty, "e", false, "Print empty line if no reserved Instances found")
	flag.StringVar(&account, "a", "Unknown", "AWS Account name")
	flag.Parse()

	if header {
		fmt.Printf("Account Name, State, Reservation Type, Expiry Date, Item Count, AV Zone, Instance Type, Offering Type, Reserved Instance ID \n")
		os.Exit(0)
	}

	// Create an EC2 service object
	// Config details Keys, secret keys and region will be read from environment
	ec2svc := ec2.New(&aws.Config{})

	ec2Filter := ec2.Filter{}
	ec2Filter.Name = aws.String("state")
	ec2Filter.Values = []*string{aws.String("active")}

	ec2drii := ec2.DescribeReservedInstancesInput{Filters: []*ec2.Filter{&ec2Filter}}

	// Call the DescribeInstances Operation
	resp, err := ec2svc.DescribeReservedInstances(&ec2drii)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// process SDK error
			fmt.Printf("AWS Error: %s - %s", awsErr.Code, awsErr.Message)
		}
		log.Fatalf("Fatal error: %s\n", err)
	}

	// extract the private ip address from the instance struct stored in the reservation
	for _, ri := range resp.ReservedInstances {

		// compute the expiry date from start + duration
		endDate := ri.Start.Add(time.Duration(*ri.Duration) * time.Second)

		fmt.Printf("%s,%s,%s,%s,%d,%s,%s,%s,%s\n",
			*(chkStringValue(&account)),
			*(chkStringValue(ri.State)),
			*(chkStringValue(aws.String("ec2"))),
			fmt.Sprintf("%d-%d-%d", endDate.Year(), endDate.Month(), endDate.Day()),
			*ri.InstanceCount,
			*(chkStringValue(ri.AvailabilityZone)),
			*(chkStringValue(ri.InstanceType)),
			*(chkStringValue(ri.OfferingType)),
			*(chkStringValue(ri.ReservedInstancesID)))

	}

	// Config details Keys, secret keys and region will be read from environment
	rdssvc := rds.New(&aws.Config{})
	// Call the DescribeInstances Operation. Note Filters are not currently supported
	rdsResp, err := rdssvc.DescribeReservedDBInstances(nil)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// process SDK error
			fmt.Printf("AWS Error: %s - %s", awsErr.Code, awsErr.Message)
		}
		log.Fatalf("Fatal error: %s\n", err)
	}

	// extract the private ip address from the instance struct stored in the reservation
	for _, ri := range rdsResp.ReservedDBInstances {

		// rds does not currently support filters so need to filter at the output end
		if *ri.State != "active" {
			continue
		}

		// compute the expiry date from start + duration
		endDate := ri.StartTime.Add(time.Duration(*ri.Duration) * time.Second)

		var avZone string
		if *ri.MultiAZ {
			avZone = "Multi Zone"
		} else {
			avZone = "Single Zone"
		}

		fmt.Printf("%s,%s,%s,%s,%d,%s,%s,%s,%s\n",
			*(chkStringValue(&account)),
			*(chkStringValue(ri.State)),
			*(chkStringValue(aws.String("rds"))),
			fmt.Sprintf("%d-%d-%d", endDate.Year(), endDate.Month(), endDate.Day()),
			*ri.DBInstanceCount,
			avZone,
			*(chkStringValue(ri.DBInstanceClass)),
			*(chkStringValue(ri.OfferingType)),
			*(chkStringValue(ri.ReservedDBInstanceID)))

	}

	if printEmpty && (len(resp.ReservedInstances)+len(rdsResp.ReservedDBInstances)) == 0 {
		fmt.Printf("%s,,,,,,,,\n", account)
	}

}

func chkStringValue(s *string) *string {
	if s == nil {
		emptyString := ""
		s = &emptyString
	}
	return s
}
