package main

import (
	"flag"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mitchellh/cli"
)

type AuditCommand struct {
	verbose    bool
	csv        bool
	all        bool
	public_ami bool
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
	if err := cmdFlags.Parse(args); err != nil {
		return RCERR
	}

	/*
		Need to switch to each func depending on what flags present.
		Also need to support --all and multiple --audit_a --audit_b --audit_c situations
		solution - long list of if, one for each option then only need to set all true if
		--all is given

		Also need to support csv output. Where is that done? Here or in the function

	*/

	if c.public_ami == true || c.all == true {
		public_ami(c.verbose, c.csv)
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
	svc := ec2.New(&aws.Config{})

	ec2dii := ec2.DescribeImagesInput{Owners: []*string{aws.String("self")}}

	imagesResp, err := describeImages(svc, &ec2dii)

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

/*

*/
