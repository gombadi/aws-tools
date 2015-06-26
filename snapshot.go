package main

import (
	"flag"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/mitchellh/cli"
)

type SSCommand struct {
	verbose    bool
	automode   bool
	instanceId string
	Ui         cli.Ui
}

func (c *SSCommand) Help() string {
	return `
	Description:
	Snapshot an instance (no reboot) and create an AMI

	Usage:
		awsgo-tools snapshot [flags]

	Flags:
	-v to produce verbose output
	-a to use auto mode to snapshot all instances with tag key of autobkup
	-i <instanceid> to snapshot one EC2 instance
	`
}

func (c *SSCommand) Synopsis() string {
	return "Snapshot instance & create AMI"
}

func (c *SSCommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.BoolVar(&c.verbose, "v", false, "Produce verbose output")
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
	svc := ec2.New(&aws.Config{})

	// load the struct that has details on all instances to be snapshotted
	bkupInstances, err := getBkupInstances(svc, c.instanceId)

	if err != nil {
		// AWS DescribeInstances failed
		fmt.Printf("Fatal error: %s\n", err)
		return RCERR
	}

	var amis []string

	// now we have the slice of instanceIds to be backed up we can create the AMI then tag them
	for _, abkupInstance := range bkupInstances {

		// snapshot the instance.
		newAMI, err := ssInstance(svc, abkupInstance)

		if err != nil {
			fmt.Printf("Error creating AWS AMI for instance %s\n", abkupInstance.InstanceID)
			fmt.Printf("Error details - %s\n", err)
		} else {
			if c.verbose {
				fmt.Printf("Info - Started creating AMI: %s\n", newAMI)
			}
			// add the amiid to the list of ami's to tag
			amis = append(amis, newAMI)
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
		err := createTags(svc, &ec2cti)

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
func getBkupInstances(svc *ec2.EC2, bkupId string) (bkupInstances []*ec2.CreateImageInput, err error) {

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

	ec2dii := ec2.DescribeInstancesInput{InstanceIDs: instanceSlice, Filters: []*ec2.Filter{&ec2Filter}}

	td := 499
LOOP:

	resp, err := svc.DescribeInstances(&ec2dii)

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
						*resp.Reservations[reservation].Instances[instance].InstanceID +
							"-" +
							strconv.FormatInt(time.Now().Unix(), 10))
				}
			}
			theInstance.Description = aws.String("Auto backup of instance " + *resp.Reservations[reservation].Instances[instance].InstanceID)
			theInstance.InstanceID = resp.Reservations[reservation].Instances[instance].InstanceID
			theInstance.NoReboot = aws.Boolean(true)
			// append details on this instance to the slice
			bkupInstances = append(bkupInstances, &theInstance)
		}
	}
	return bkupInstances, nil
}

// ssInstance will create an AMI for an instance and return the new AMI reference
func ssInstance(svc *ec2.EC2, abkupInstance *ec2.CreateImageInput) (newAMI string, err error) {

	td := 499
LOOPCI:
	createImageResp, err := svc.CreateImage(abkupInstance)

	// AWS retry logic
	if err != nil {
		if reqErr, ok := err.(awserr.RequestFailure); ok {
			if scErr := reqErr.StatusCode(); scErr >= 500 && scErr < 600 {
				// if retryable then double the delay for the next run
				// if time delay > 64 seconds then give up on this request & move on
				if td = td + td; td < 64000 {
					time.Sleep(time.Duration(td) * time.Millisecond)
					// loop around and try again
					goto LOOPCI
				}
			}
		}
	}
	if err != nil {
		fmt.Printf("Fatal error: %s\n", err)
		return "", err
	}
	return *createImageResp.ImageID, nil
}

// createTags is a helper function that will add tags to a given resource
func createTags(svc *ec2.EC2, ec2cti *ec2.CreateTagsInput) (err error) {

	td := 499
LOOPCT:

	_, err = svc.CreateTags(ec2cti)

	// AWS retry logic
	if err != nil {
		if reqErr, ok := err.(awserr.RequestFailure); ok {
			if scErr := reqErr.StatusCode(); scErr >= 500 && scErr < 600 {
				// if retryable then double the delay for the next run
				// if time delay > 64 seconds then give up on this request & move on
				if td = td + td; td < 64000 {
					time.Sleep(time.Duration(td) * time.Millisecond)
					// loop around and try again
					goto LOOPCT
				}
			}
		}
	}
	return
}
