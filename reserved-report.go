package main

import (
	"flag"
	"fmt"
	"sync"
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
	-a <account name> - account name to use in CSV output
	-e - produce an empty line if no reserved instances found
	-h - print headers and exit
	`
}

// Synopsis function returns a string with concise details of the sub command
func (c *RRCommand) Synopsis() string {
	return "EC2 & RDS reserved Instance report CSV Output"
}

// Run function is the function called by the cli library to run the actual sub command code.
func (c *RRCommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("reserved-report", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.BoolVar(&c.header, "h", false, "Produce CSV Headers and exit")
	cmdFlags.BoolVar(&c.printEmpty, "e", false, "Print empty line if no reserved instances found")
	cmdFlags.StringVar(&c.account, "a", "unknown", "AWS Account Name to use")
	if err := cmdFlags.Parse(args); err != nil {
		fmt.Printf("Error processing commandline flags\n")
		return RCERR
	}

	if c.header {
		fmt.Printf("Account Name, State, Reservation Type, Expiry Date, Item Count, AV Zone, Instance Type, Offering Type, Reserved Instance ID \n")
		return RCOK
	}

	var wg sync.WaitGroup
	var ec2resp *ec2.DescribeReservedInstancesOutput
	var rdsresp *rds.DescribeReservedDBInstancesOutput
	var ec2err, rdserr error

	// use concurrency to ask AWS multiple questions at once
	wg.Add(1)
	go func() {

		defer wg.Done()
		// setup ec2 filter
		ec2Filter := ec2.Filter{}
		ec2Filter.Name = aws.String("state")
		ec2Filter.Values = []*string{aws.String("active")}
		ec2drii := ec2.DescribeReservedInstancesInput{Filters: []*ec2.Filter{&ec2Filter}}

		// Create an EC2 service object
		// Config details Keys, secret keys and region will be read from environment
		ec2svc := ec2.New(&aws.Config{MaxRetries: 10})

		// Call the DescribeInstances Operation
		ec2resp, ec2err = ec2svc.DescribeReservedInstances(&ec2drii)

	}()

	wg.Add(1)
	go func() {

		defer wg.Done()
		// Config details Keys, secret keys and region will be read from environment
		rdssvc := rds.New(&aws.Config{MaxRetries: 10})

		// Call the DescribeInstances Operation. Note Filters are not currently supported
		rdsresp, rdserr := rdssvc.DescribeReservedDBInstances(nil)

	}()

	// wait until both goroutines have completed talking to AWS
	wg.Wait()

	if ec2err != nil {
		fmt.Printf("AWS error: %s\n", ec2err)
		return RCERR
	}

	if rdserr != nil {
		fmt.Printf("AWS error: %s\n", rdserr)
		return RCERR
	}

	// extract the reserved instance details for ec2
	for _, ri := range ec2resp.ReservedInstances {

		// compute the expiry date from start + duration
		endDate := ri.Start.Add(time.Duration(*ri.Duration) * time.Second)

		fmt.Printf("%s,%s,%s,%s,%d,%s,%s,%s,%s\n",
			safeString(&c.account),
			safeString(ri.State),
			safeString(aws.String("ec2")),
			fmt.Sprintf("%d-%d-%d", endDate.Year(), endDate.Month(), endDate.Day()),
			*ri.InstanceCount,
			safeString(ri.AvailabilityZone),
			safeString(ri.InstanceType),
			safeString(ri.OfferingType),
			safeString(ri.ReservedInstancesID))

	}

	// extract the rds reserved instance details for rds
	for _, ri := range rdsresp.ReservedDBInstances {

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
			safeString(&c.account),
			safeString(ri.State),
			safeString(aws.String("rds")),
			fmt.Sprintf("%d-%d-%d", endDate.Year(), endDate.Month(), endDate.Day()),
			*ri.DBInstanceCount,
			avZone,
			safeString(ri.DBInstanceClass),
			safeString(ri.OfferingType),
			safeString(ri.ReservedDBInstanceID))

	}

	if c.printEmpty && (len(ec2resp.ReservedInstances)+len(rdsresp.ReservedDBInstances)) == 0 {
		fmt.Printf("%s,,,,,,,,\n", c.account)
	}

	return RCOK
}

/*

 */
