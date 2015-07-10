package main

import (
	"flag"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/mitchellh/cli"
)

type AuditCommand struct {
	verbose    bool
	csv        bool
	all        bool
	public_ami bool
	users      bool
	Ui         cli.Ui
}

func (c *AuditCommand) Help() string {
	return `
	Description:
	Audit various AWS settings/configurations/usage and report results

	Usage:
		awsgo-tools audit [flags]

	Flags:
	-v - produce verbose output
	--csv - produce output in csv format if possible
	--all - run all the audit checks
	--public_ami - check for AMI's owned by account but with public visibility
	--users - show password & access key last used details
	`
}

func (c *AuditCommand) Synopsis() string {
	return "Audit various AWS services"
}

func (c *AuditCommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("audit", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.BoolVar(&c.verbose, "v", false, "Produce verbose output")
	cmdFlags.BoolVar(&c.csv, "csv", false, "Produce output in csv format")
	cmdFlags.BoolVar(&c.all, "all", false, "Select all Audit options")
	cmdFlags.BoolVar(&c.public_ami, "public_ami", false, "Audit AMI's for public launch permissions")
	cmdFlags.BoolVar(&c.users, "users", false, "Audit Users password & AccessKey last used")
	if err := cmdFlags.Parse(args); err != nil {
		return RCERR
	}

	if c.public_ami == true || c.all == true {
		public_ami(c.verbose, c.csv)
	}

	if c.users == true || c.all == true {
		users(c.verbose, c.csv)
	}

	return RCOK
}

// public_ami function displays any AMI that has public launch permissions
func public_ami(verbose bool, csv bool) {

	if verbose == true {
		fmt.Printf("#### Begin Audit of Public AMI Launch Permissions ####\n")
	}

	// Create an EC2 service object
	// config values keys, sercet key & region read from environment
	svc := ec2.New(&aws.Config{MaxRetries: 10})

	ec2dii := ec2.DescribeImagesInput{Owners: []*string{aws.String("self")}}

	imagesResp, err := svc.DescribeImages(&ec2dii)

	if err != nil {
		fmt.Printf("AWS Error: %s\n", err)
		return
	}

	for _, image := range imagesResp.Images {
		if *image.Public == true {
			fmt.Printf("AMI %s has Public launch permissions\n", *image.ImageID)
		}

	}

	if verbose == true {
		fmt.Printf("#### Audit Complete ####\n")
	}

	return

}

// users function will display details on all users and last used info on passwords
// and access keys
func users(verbose bool, csv bool) {

	// ListUsers to get a list of all users on the account. Check truncated
	// Create an IAM service object
	// Config details Keys, secret keys and region will be read from environment
	svc := iam.New(&aws.Config{MaxRetries: 10})

	iamlui := &iam.ListUsersInput{
		Marker:     nil,
		MaxItems:   nil,
		PathPrefix: nil,
	}

	var dowhile bool = true

	// for each user returned record password last used time
	for dowhile == true {

		iamluo, err := svc.ListUsers(iamlui)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				// process SDK error
				fmt.Printf("AWS Error: %s - %s", awsErr.Code(), awsErr.Message())
			}
			return
		}

		if *iamluo.IsTruncated == false {
			dowhile = false
		}
		iamlui.Marker = iamluo.Marker

		// loop over each user account in IAM
		for _, user := range iamluo.Users {

			iamlaki := &iam.ListAccessKeysInput{
				UserName: user.UserName,
			}

			fmt.Printf("\nUsername: %s\nPassword Last Used: %s\n",
				*user.UserName,
				safeDate(user.PasswordLastUsed))

			iamlako, err := svc.ListAccessKeys(iamlaki)
			if err != nil {
				fmt.Printf("AWS Error: %s\n", err)
				return
			}

			// loop over each access key for the user
			for _, accesskey := range iamlako.AccessKeyMetadata {

				iamgaklui := &iam.GetAccessKeyLastUsedInput{
					AccessKeyID: accesskey.AccessKeyID,
				}

				iamgakluo, err := svc.GetAccessKeyLastUsed(iamgaklui)

				if err != nil {
					fmt.Printf("AWS Error: %s\n", err)
					return
				}

				fmt.Printf("AccessKey: %s\nStatus: %s\nDate Last Used: %s\nRegion: %s\nService: %s\n",
					*accesskey.AccessKeyID,
					*accesskey.Status,
					safeDate(iamgakluo.AccessKeyLastUsed.LastUsedDate),
					*(chkStringValue(iamgakluo.AccessKeyLastUsed.Region)),
					*(chkStringValue(iamgakluo.AccessKeyLastUsed.ServiceName)))
			}
		}

	}

	return
}

/*

*/
