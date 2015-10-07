package main

import (
	"flag"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/mitchellh/cli"
)

type S3infoCommand struct {
	all    bool
	csv    bool
	info   bool
	kilo   bool
	mega   bool
	giga   bool
	bucket string
	trend  int
	Ui     cli.Ui
}

// Help function displays detailed help for ths iamssl sub command
func (c *S3infoCommand) Help() string {
	return `
	Description:
	Display info about AWS S3 bucket sizes and object counts

	Usage:
		awsgo-tools s3info [flags]

	Flags:
	-a - Display info on all available buckets
	-b <bucket name> - Display info on one bucket
	-c - Display info in csv format
	-i - Display extra info about the S3 bucket
	-o - Display S3 bucket object count details
	-s - Display S3 bucket size info
	-t <days> - Display size trend over this many days
	-K - Display bucket size info in KiloBytes
	-M - Display bucket size info in MegaBytes
	-G - Display bucket size info in GigaBytes
	`
}

// Synopsis function returns a string with concise details of the sub command
func (c *S3infoCommand) Synopsis() string {
	return "S3 Bucket Size Display"
}

// Run function is the function called by the cli library to run the actual sub command code.
func (c *S3infoCommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("s3info", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.BoolVar(&c.all, "a", false, "Display info for all S3 buckets")
	cmdFlags.BoolVar(&c.info, "i", false, "Display extra info about the buckets")
	cmdFlags.BoolVar(&c.csv, "c", false, "Display info in csv format")
	cmdFlags.BoolVar(&c.kilo, "K", false, "Display info in KiloBytes")
	cmdFlags.BoolVar(&c.mega, "M", false, "Display info in MegaBytes")
	cmdFlags.BoolVar(&c.giga, "G", false, "Display info in GigaBytes")
	cmdFlags.StringVar(&c.bucket, "b", "", "S3 bucket to show details for")
	cmdFlags.IntVar(&c.trend, "t", 14, "Display size trend over this many days")
	if err := cmdFlags.Parse(args); err != nil {
		return RCERR
	}

	// Config details Keys, secret keys and region will be read from environment
	s3svc := s3.New(&aws.Config{MaxRetries: aws.Int(10)})

	params := &s3.GetBucketLocationInput{
		Bucket: aws.String(c.bucket),
	}
	resp, err := s3svc.GetBucketLocation(params)

	if err != nil {
		fmt.Printf("GetBucketLocation fatal error: %s\n", err)
		return RCERR
	}

	if resp.LocationConstraint != nil {
		fmt.Printf("Bucket %s is in region %s\n", c.bucket, *resp.LocationConstraint)
	} else {
		fmt.Printf("Unable to find the location of s3 bucket %s\n", c.bucket)
	}

	return RCOK
}

/*

 */
