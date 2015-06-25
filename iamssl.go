package main

import (
	"flag"
	"fmt"
	"time"

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

func (c *IAMsslCommand) Synopsis() string {
	return "IAM SSL CSV Output"
}

func (c *IAMsslCommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("iamssl", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.BoolVar(&c.header, "h", false, "Produce CSV Headers and exit")
	cmdFlags.BoolVar(&c.printEmpty, "e", false, "Print empty line if no SSL Certs found")
	cmdFlags.StringVar(&c.account, "a", "unknown", "AWS Account Name to use")
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	if c.header {
		fmt.Printf("Account Name, Expiry Date, Certificate Name, Certificate ID, Upload Date\n")
		return RCOK
	}

	// Create an IAM service object
	// Config details Keys, secret keys and region will be read from environment
	svc := iam.New(&aws.Config{})

	td := 499
LOOP:

	resp, err := svc.ListServerCertificates(nil)

	// AWS retry logic
	if err != nil {
		if reqErr, ok := err.(awserr.RequestFailure); ok {
			if scErr := reqErr.StatusCode(); scErr >= 500 && scErr < 600 {
				// if retryable then double the delay for the next run
				// if time delay > 64 seconds then give up on this request & move on
				if td = td + td; td < 64000 {
					time.Sleep(time.Duration(td) * time.Millisecond)
					// loop around and try again
					goto LOOP
				}
			}
		}
	}

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
