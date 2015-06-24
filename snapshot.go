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
	Snapshot an instance and create an AMI

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
		return RCOK
	}

	// Create an EC2 service object
	// config values keys, sercet key & region read from environment
	svc := ec2.New(&aws.Config{})

	// load the struct that has details on all instances to be snapshotted
	bkupInstances, err := getBkupInstances(svc, c.instanceId)

	if err != nil {
		fmt.Printf("Fatal error: %s\n", err)
		return RCERR
	}

	// now we have the slice of instances to be backed up we can create the AMI then tag them

	for _, abkupInstance := range bkupInstances {
		// snapshot the instance.
		if newAMI := ssInstance(svc, abkupInstance); newAMI != "" {
			if c.verbose {
				fmt.Printf("Backing up instance Id: %s named %s completed. New AMI: %s\n",
					*abkupInstance.InstanceID,
					*abkupInstance.Name,
					newAMI)
			}

		} else {
			fmt.Printf("Error creating AWS AMI for instance %s\n", abkupInstance.InstanceID)
		}

	}

	if c.verbose {
		fmt.Printf("All done.\n")

	}
	return RCOK
}

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

func ssInstance(svc *ec2.EC2, abkupInstance *ec2.CreateImageInput) (newAMI string) {

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
		return ""
	}

	// Having some issues with AMI not valid when we try and tag it so give it some time to become ready
	// FIXME - takes too long. Need to reduce or change to multiple go routines to tag
	time.Sleep(47 * time.Second)
	// store the creation time in the tag so it can be checked during auto cleanup
	ec2cti := ec2.CreateTagsInput{
		Resources: []*string{createImageResp.ImageID},
		Tags: []*ec2.Tag{
			&ec2.Tag{
				Key:   aws.String("autocleanup"),
				Value: aws.String(strconv.FormatInt(time.Now().Unix(), 10))},
			&ec2.Tag{
				Key:   aws.String("Name"),
				Value: aws.String("Autobkup-" + *abkupInstance.InstanceID)}}}

	td = 499
LOOPCT:

	_, err = svc.CreateTags(&ec2cti)

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
	if err != nil {
		fmt.Printf("non-fatal error adding autocleanup tag to image: %v\n", err)
	}

	return *createImageResp.ImageID
}
