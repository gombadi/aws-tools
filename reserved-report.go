package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/mitchellh/cli"
)

type RRCommand struct {
	header     bool
	printEmpty bool
	account    string
	Ui         cli.Ui
}

// Help function displays detailed help for ths reserver-report sub command
func (c *RRCommand) Help() string {
	return `
	Description:
	Produce CSV output with details of all active EC2 & RDS reserved instances

	Usage:
		awsgo-tools reserved-report [flags]

	Flags:
	-h - print headers and exit
	-e - produce an empty line if no reserved instances found
	-a <account name> - account name to use in CSV output
	`
}

// Synopsis function returns a string with concise details of the sub command
func (c *RRCommand) Synopsis() string {
	return "Reserved Instance report CSV Output"
}

// Run function is the function called by the cli library to run the actual sub command code.
func (c *RRCommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("reserved-report", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.BoolVar(&c.header, "h", false, "Produce CSV Headers and exit")
	cmdFlags.BoolVar(&c.printEmpty, "e", false, "Print empty line if no reserved instances found")
	cmdFlags.StringVar(&c.account, "a", "unknown", "AWS Account Name to use")
	if err := cmdFlags.Parse(args); err != nil {
		return RCERR
	}

	if c.header {
		fmt.Printf("Account Name, State, Reservation Type, Expiry Date, Item Count, AV Zone, Instance Type, Offering Type, Reserved Instance ID \n")
		return RCOK
	}

	// Create an EC2 service object
	// Config details Keys, secret keys and region will be read from environment
	ec2svc := ec2.New(&aws.Config{MaxRetries: 10})

	ec2Filter := ec2.Filter{}
	ec2Filter.Name = aws.String("state")
	ec2Filter.Values = []*string{aws.String("active")}

	ec2drii := ec2.DescribeReservedInstancesInput{Filters: []*ec2.Filter{&ec2Filter}}

	// Call the DescribeInstances Operation
	resp, err := ec2svc.DescribeReservedInstances(&ec2drii)

	if err != nil {
		fmt.Printf("Fatal error: %s\n", err)
		return RCERR
	}

	// extract the private ip address from the instance struct stored in the reservation
	for _, ri := range resp.ReservedInstances {

		// compute the expiry date from start + duration
		endDate := ri.Start.Add(time.Duration(*ri.Duration) * time.Second)

		fmt.Printf("%s,%s,%s,%s,%d,%s,%s,%s,%s\n",
			*(chkStringValue(&c.account)),
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
	rdssvc := rds.New(&aws.Config{MaxRetries: 10})

	// Call the DescribeInstances Operation. Note Filters are not currently supported
	rdsResp, err := rdssvc.DescribeReservedDBInstances(nil)

	if err != nil {
		fmt.Printf("Fatal error: %s\n", err)
		return RCERR
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
			*(chkStringValue(&c.account)),
			*(chkStringValue(ri.State)),
			*(chkStringValue(aws.String("rds"))),
			fmt.Sprintf("%d-%d-%d", endDate.Year(), endDate.Month(), endDate.Day()),
			*ri.DBInstanceCount,
			avZone,
			*(chkStringValue(ri.DBInstanceClass)),
			*(chkStringValue(ri.OfferingType)),
			*(chkStringValue(ri.ReservedDBInstanceID)))

	}

	if c.printEmpty && (len(resp.ReservedInstances)+len(rdsResp.ReservedDBInstances)) == 0 {
		fmt.Printf("%s,,,,,,,,\n", c.account)
	}

	return RCOK
}

/*

*/
