/*
This application will output details in CSV format of all SSL certificates stored in IAM.
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
)

func main() {

	// storage for commandline args
	var header, printEmpty bool
	var account string

	flag.BoolVar(&header, "h", false, "Produce CSV Headers and exit")
	flag.BoolVar(&printEmpty, "e", false, "Print empty line if no SSL Certs found")
	flag.StringVar(&account, "a", "Unknown", "AWS Account name")
	flag.Parse()

	if header {
		fmt.Printf("Account Name, Expiry Date, Certificate Name, Certificate ID, Upload Date\n")
		os.Exit(0)
	}

	// Create an IAM service object
	// Config details Keys, secret keys and region will be read from environment
	svc := iam.New(&aws.Config{})

	// Call the DescribeInstances Operation
	resp, err := svc.ListServerCertificates(nil)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// process SDK error
			fmt.Printf("AWS Error: %s - %s", awsErr.Code, awsErr.Message)
		}
		log.Fatalf("Fatal error: %s\n", err)
	}

	// extract the private ip address from the instance struct stored in the reservation
	for _, scml := range resp.ServerCertificateMetadataList {

		fmt.Printf("%s,%s,%s,%s,%s\n",
			account,
			fmt.Sprintf("%d-%d-%d", scml.Expiration.Year(), scml.Expiration.Month(), scml.Expiration.Day()),
			*(chkStringValue(scml.ServerCertificateName)),
			*(chkStringValue(scml.ServerCertificateID)),
			fmt.Sprintf("%d-%d-%d", scml.UploadDate.Year(), scml.UploadDate.Month(), scml.UploadDate.Day()))

	}

	if printEmpty && len(resp.ServerCertificateMetadataList) == 0 {
		fmt.Printf("%s,,,\n", account)
	}

}

func chkStringValue(s *string) *string {
	if s == nil {
		emptyString := ""
		s = &emptyString
	}
	return s
}
