package main

import (
	"flag"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/mitchellh/cli"
)

type IAMsslCommand struct {
	header     bool
	printEmpty bool
	account    string
	Ui         cli.Ui
}

// Help function displays detailed help for ths iamssl sub command
func (c *IAMsslCommand) Help() string {
	return `
	Description:
	Produce CSV output of all SSL certificates stored in IAM

	Usage:
		awsgo-tools iamssl [flags]

	Flags:
	-a <account name> - Account name to add to CSV output to identify the
	-h - Produce CSV Headers only and exit
	-e - Print empty csv line id no certificates found for the account
	`
}

// Synopsis function returns a string with concise details of the sub command
func (c *IAMsslCommand) Synopsis() string {
	return "IAM SSL CSV Output"
}

// Run function is the function called by the cli library to run the actual sub command code.
func (c *IAMsslCommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("iamssl", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.BoolVar(&c.header, "h", false, "Produce CSV Headers and exit")
	cmdFlags.BoolVar(&c.printEmpty, "e", false, "Print empty line if no SSL Certs found")
	cmdFlags.StringVar(&c.account, "a", "unknown", "AWS Account Name to use")
	if err := cmdFlags.Parse(args); err != nil {
		return RCERR
	}

	if c.header {
		fmt.Printf("Account Name, Expiry Date, Certificate Name, Certificate ID, Upload Date\n")
		return RCOK
	}

	// Create an IAM service object
	// Config details Keys, secret keys and region will be read from environment
	svc := iam.New(&aws.Config{MaxRetries: 10})

	resp, err := svc.ListServerCertificates(nil)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// process SDK error
			fmt.Printf("AWS Error: %s - %s", awsErr.Code, awsErr.Message)
		}
		fmt.Printf("Fatal error: %s\n", err)
		return RCERR
	}

	// extract the private ip address from the instance struct stored in the reservation
	for _, scml := range resp.ServerCertificateMetadataList {

		fmt.Printf("%s,%s,%s,%s,%s\n",
			c.account,
			fmt.Sprintf("%d-%d-%d", scml.Expiration.Year(), scml.Expiration.Month(), scml.Expiration.Day()),
			*(chkStringValue(scml.ServerCertificateName)),
			*(chkStringValue(scml.ServerCertificateID)),
			fmt.Sprintf("%d-%d-%d", scml.UploadDate.Year(), scml.UploadDate.Month(), scml.UploadDate.Day()))

	}

	if c.printEmpty && len(resp.ServerCertificateMetadataList) == 0 {
		fmt.Printf("%s,,,\n", c.account)
	}
	return RCOK
}

/*

*/
