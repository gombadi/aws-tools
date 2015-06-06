/*
This application will snapshot an amazon instance and also give it
tags of Key:Date, Value:todays date and key:Name, Value:same as existing
Name Tag or use instanceId

The resulting AMI can be used to recover the instance if needed.

In auto mode it will find all current instances with a Tag name of autobkup
and create an AMI of them with a Tag of autocleanup

Command line options -
-a <true|false> Auto snapshot mode
-i Instance ID to be backed up
-v verbose mode


*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/gombadi/go-rate"
)

// cmdline flag if we want verbose output
var verbose bool

func getBkupInstances(svc *ec2.EC2, bkupId string) (bkupInstances []*ec2.CreateImageInput) {

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
	resp, err := svc.DescribeInstances(&ec2dii)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// process SDK error
			fmt.Printf("AWS Error: %s - %s", awsErr.Code, awsErr.Message)
		}
		log.Fatalf("Fatal error: %s\n", err)
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
	return
}

func ssInstance(svc *ec2.EC2, abkupInstance *ec2.CreateImageInput) {

	createImageResp, err := svc.CreateImage(abkupInstance)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			// process SDK error
			fmt.Printf("AWS Error: %s - %s", awsErr.Code, awsErr.Message)
		}
		log.Fatalf("Fatal error: %s\n", err)
	}

	// Having some issues with AMI not valid when we try and tag it so give it some time to become ready
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

	_, err = svc.CreateTags(&ec2cti)

	if err != nil {
		log.Printf("non-fatal error adding autocleanup tag to image: %v\n", err)
	}

	if verbose {
		time.Sleep(1 * time.Second)
		fmt.Printf("Backing up instance Id: %s named %s completed. New AMI: %s\n", *abkupInstance.InstanceID, *abkupInstance.Name, *createImageResp.ImageID)
	}

}

func main() {

	// storage for commandline args
	var autoFlag bool
	var bkupId string

	flag.BoolVar(&verbose, "v", false, "Produce verbose output")
	flag.BoolVar(&autoFlag, "a", false, "In auto mode snapshot any instance with an autobkup tag")
	flag.StringVar(&bkupId, "i", "", "Instance id to be backed up")
	flag.Parse()

	// make sure we are in auto mode or an ami id has been provided
	if !autoFlag && len(bkupId) == 0 {
		fmt.Printf("No instance details provided. Please provide an instance id to snapshot\nor enable auto mode to snapshot all tagged instances.\n\n")
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Create an EC2 service object
	// config values keys, sercet key & region read from environment
	svc := ec2.New(&aws.Config{})

	// load the struct that has details on all instances to be snapshotted
	bkupInstances := getBkupInstances(svc, bkupId)

	// now we have the slice of instances to be backed up we can create the AMI then tag them

	var wg sync.WaitGroup

	// rate limit the AWS requests to max 3 per second
	rl := rate.New(3, time.Second)

	for instance := range bkupInstances {

		rl.Wait()

		// Increment the WaitGroup counter.
		wg.Add(1)

		// Launch a goroutine to fetch the URL.
		go func(svc *ec2.EC2, abkupInstance ec2.CreateImageInput) {
			// Decrement the counter when the goroutine completes.
			defer wg.Done()
			// snapshot the instance.
			ssInstance(svc, &abkupInstance)
		}(svc, *bkupInstances[instance])

	}

	// Wait for all Amazon requests to complete.
	wg.Wait()

	if verbose {
		fmt.Printf("All done.\n")

	}

}
