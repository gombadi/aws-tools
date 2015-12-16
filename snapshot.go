package main

import (
	"flag"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mitchellh/cli"
)

type SSCommand struct {
	verbose    bool
	dryrun     bool
	automode   bool
	reboot     bool
	instanceId string
	Ui         cli.Ui
}

// Help function displays detailed help for ths snapshot sub command
func (c *SSCommand) Help() string {
	return `
	Description:
	Snapshot an instance (no reboot) and create an AMI

	Usage:
		awsgo-tools snapshot [flags]

	Flags:
	-a to use auto mode to snapshot all instances with tag key of autobkup
	-i <instanceid> to snapshot one EC2 instance
	-n - Dry run. Report what would have happened but make no changes
	-f force an instance reboot when making the snapshot
	-v to produce verbose output
	`
}

// Synopsis function returns a string with concise details of the sub command
func (c *SSCommand) Synopsis() string {
	return "Snapshot instance & create AMI"
}

// Run function is the function called by the cli library to run the actual sub command code.
func (c *SSCommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.BoolVar(&c.verbose, "v", false, "Produce verbose output")
	cmdFlags.BoolVar(&c.dryrun, "n", false, "Dry Run")
	cmdFlags.BoolVar(&c.reboot, "f", false, "Reboot instance wehn making snapshot. default: false")
	cmdFlags.BoolVar(&c.automode, "a", false, "auto mode to snapshot any instance with a tag key of autobkup")
	cmdFlags.StringVar(&c.instanceId, "i", "", "instance to be backed up")
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// make sure we are in auto mode or an ami id has been provided
	if !c.automode && len(c.instanceId) == 0 {
		fmt.Printf("No instance details provided. Please provide an instance id to snapshot\nor enable auto mode to snapshot all tagged instances.\n\n")
		return RCERR
	}

	// Create an EC2 service object
	// config values keys, sercet key & region read from environment
	svc := ec2.New(session.New(), &aws.Config{MaxRetries: aws.Int(10)})

	// load the struct that has details on all instances to be snapshotted
	bkupInstances, err := getBkupInstances(svc, c.instanceId, c.reboot)

	if err != nil {
		// AWS DescribeInstances failed
		fmt.Printf("Fatal error: %s\n", err)
		return RCERR
	}

	// amis contains the list of AMI's that have been created and need tagged
	var amis []string

	// now we have the slice of instanceIds to be backed up we can create the AMI then tag them
	for _, abkupInstance := range bkupInstances {

		if c.dryrun == false {
			// snapshot the instance.
			createImageResp, err := svc.CreateImage(abkupInstance)

			if err != nil {
				fmt.Printf("Error creating AWS AMI for instance %s\n", *abkupInstance.InstanceId)
				fmt.Printf("Error details - %s\n", err)
			} else {
				if c.verbose {
					fmt.Printf("Info - Started creating AMI: %s\n", *createImageResp.ImageId)
				}
				// add the amiid to the list of ami's to tag
				amis = append(amis, *createImageResp.ImageId)
			}
		} else {
			fmt.Printf("Dry Run - Would have created AMI for instance %s\n", *abkupInstance.InstanceId)
		}
	}

	// if no AMI's created then lets leave
	if len(amis) == 0 {
		return RCOK
	}

	if c.verbose {
		fmt.Printf("AMI's creation has started. Now waiting for AWS to make AMI's available to tag...\n")
	}
	time.Sleep(47 * time.Second)

	theTags := []*ec2.Tag{
		&ec2.Tag{
			Key:   aws.String("autocleanup"),
			Value: aws.String(strconv.FormatInt(time.Now().Unix(), 10))}}

	for _, ami := range amis {

		ec2cti := ec2.CreateTagsInput{
			Resources: []*string{aws.String(ami)},
			Tags:      theTags}

		// call the create tag func
		_, err = svc.CreateTags(&ec2cti)

		if err != nil {
			fmt.Printf("Warning - problem adding tags to AMI: %s. Error was %s\n", ami, err)
		}
		if c.verbose {
			fmt.Printf("Info - Tagged AMI: %s\n", ami)
		}
	}

	if c.verbose {
		fmt.Printf("All done.\n")
	}
	return RCOK
}

// getBkupInstances will return a slice of CreateImageInput structures for either a single instance
// or all instances in an account that have a tag key of autobkup
func getBkupInstances(svc *ec2.EC2, bkupId string, reboot bool) (bkupInstances []*ec2.CreateImageInput, err error) {

	var instanceSlice []*string
	var ec2Filter ec2.Filter

	// if instance id provided use it else search for tags autobkup
	if len(bkupId) > 0 {
		instanceSlice = append(instanceSlice, &bkupId)
		ec2Filter.Name = nil
		ec2Filter.Values = nil
	} else {
		ec2Filter.Name = aws.String("tag-key")
		ec2Filter.Values = []*string{aws.String("autobkup")}
		instanceSlice = nil
	}

	ec2dii := ec2.DescribeInstancesInput{InstanceIds: instanceSlice, Filters: []*ec2.Filter{&ec2Filter}}

	resp, err := svc.DescribeInstances(&ec2dii)

	if err != nil {
		return nil, err
	}

	// for any instance found extract tag name and instanceid
	for reservation := range resp.Reservations {
		for instance := range resp.Reservations[reservation].Instances {
			// Create a new theInstance variable for each run through the loop
			theInstance := ec2.CreateImageInput{}

			for tag := range resp.Reservations[reservation].Instances[instance].Tags {
				if *resp.Reservations[reservation].Instances[instance].Tags[tag].Key == "Name" {
					// name of the created AMI must be unique so add the Unix Epoch
					theInstance.Name = aws.String(
						*resp.Reservations[reservation].Instances[instance].Tags[tag].Value +
							"-" +
							strconv.FormatInt(time.Now().Unix(), 10))
					break
				} else {
					theInstance.Name = aws.String(
						*resp.Reservations[reservation].Instances[instance].InstanceId +
							"-" +
							strconv.FormatInt(time.Now().Unix(), 10))
				}
			}
			theInstance.Description = aws.String("Auto backup of instance " + *resp.Reservations[reservation].Instances[instance].InstanceId)
			theInstance.InstanceId = resp.Reservations[reservation].Instances[instance].InstanceId
			// swap value as the question is NoReboot?
			theInstance.NoReboot = aws.Bool(!reboot)
			// append details on this instance to the slice
			bkupInstances = append(bkupInstances, &theInstance)
		}
	}
	return bkupInstances, nil
}
