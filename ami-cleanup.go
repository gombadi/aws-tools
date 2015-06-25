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

type AMICommand struct {
	verbose  bool
	autoDays int
	amiId    string
	Ui       cli.Ui
}

func (c *AMICommand) Help() string {
	return `
	Description:
	Delete AMI and associated snapshots

	Usage:
		awsgo-tools ami-cleanup [flags]

	Flags:
	-i <AMI Id> - Delete single AMI & snapshots
	-a <days> - Auto cleanup AMI & snapshots that have create date more then <days> ago
	-v - Produce verbose output
	`
}

func (c *AMICommand) Synopsis() string {
	return "Delete AMI & snapshots"
}

func (c *AMICommand) Run(args []string) int {

	cmdFlags := flag.NewFlagSet("amicleanup", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }

	cmdFlags.BoolVar(&c.verbose, "v", false, "Produce verbose output")
	cmdFlags.IntVar(&c.autoDays, "a", 0, "In auto cleanup mode, cleanup any AMI's older than this number of days")
	cmdFlags.StringVar(&c.amiId, "i", "", "AMI to be deeted")
	if err := cmdFlags.Parse(args); err != nil {
		return RCERR
	}

	// make sure we are in auto mode or an ami id has been provided
	if c.autoDays == 0 && len(c.amiId) == 0 {
		fmt.Printf("No ami details provided. Please provide an ami-id to cleanup\nor enable auto cleanup mode and specify a number of days.\n")
		return RCERR
	}

	// Create an EC2 service object
	// config values keys, sercet key & region read from environment
	svc := ec2.New(&aws.Config{})

	ec2Filter := ec2.Filter{}

	var ec2dii ec2.DescribeImagesInput

	// config for auto mode
	if c.autoDays > 0 {

		// auto mode search for ami's to cleanup
		ec2Filter.Name = aws.String("tag-key")
		ec2Filter.Values = []*string{aws.String("autocleanup")}
		owners := []*string{aws.String("self")}

		ec2dii = ec2.DescribeImagesInput{Owners: owners, Filters: []*ec2.Filter{&ec2Filter}}

	} else {
		// single ami manual mode
		ec2dii = ec2.DescribeImagesInput{ImageIDs: []*string{aws.String(c.amiId)}}
	}

	td := 499
LOOP:

	imagesResp, err := svc.DescribeImages(&ec2dii)

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
		fmt.Printf("Fatal error: %s\n", err)
		return RCERR
	}

	// AWS response is ok to work with

	// sanity check to make sure we don't remove all images from account
	if len(imagesResp.Images) == 0 {
		if c.verbose {
			fmt.Printf("No images found to cleanup. Exiting\n")
		}
		return RCOK
	}

	// contains a list of all snapshotID's that need to be deleted from all deregistered AMI's
	var snapshots []string

	//
	for image := range imagesResp.Images {

		// The returned Images from AWS should only be the ones with autocleanup but lets check anyway
		// and only delete if the days have passed

		for tag := range imagesResp.Images[image].Tags {
			if *imagesResp.Images[image].Tags[tag].Key == "autocleanup" {
				// check if time is up for this AMI

				// extract the time this AMI was created
				amiCreation, _ := strconv.ParseInt(*imagesResp.Images[image].Tags[tag].Value, 10, 64)
				amiLifeSpan := time.Now().Unix() - amiCreation

				if int64(c.autoDays*86000) < amiLifeSpan {

					if c.verbose {
						fmt.Printf("Info - Deregistering AMI: %s\n", *imagesResp.Images[image].ImageID)
					}

					err := cleanupAMI(svc, imagesResp.Images[image])

					if err != nil {
						fmt.Printf("error deregistering AMI %s. Image and snapshots not cleaned up. Error details\n%si\n",
							*imagesResp.Images[image].ImageID,
							err)
						// continue with next tag
						continue
					}

					for blockDM := range imagesResp.Images[image].BlockDeviceMappings {
						if len(*imagesResp.Images[image].BlockDeviceMappings[blockDM].EBS.SnapshotID) > 0 {
							if c.verbose {
								fmt.Printf("Info - Will delete associated snapshot: %s from ami: %s\n",
									*imagesResp.Images[image].BlockDeviceMappings[blockDM].EBS.SnapshotID,
									*imagesResp.Images[image].ImageID)
							}
							snapshots = append(snapshots, *imagesResp.Images[image].BlockDeviceMappings[blockDM].EBS.SnapshotID)
						}
					}
					if c.verbose {
						fmt.Printf("Info - AMI: %s deregistered\n", *imagesResp.Images[image].ImageID)
					}
				} else {
					if c.verbose {
						fmt.Printf("Info - Not deregistering AMI: %s as expire time not reached\n", *imagesResp.Images[image].ImageID)
					}
				}
			}
		}
	}

	if len(snapshots) > 0 {
		if c.verbose {
			fmt.Printf("Waiting for AWS to break linkage between AMI & snapshot so snapshots can be deleted...\n")
		}
		// pause a while to make sure AWS has broken link between AMI and snapshots so the snapshots can be deleted
		time.Sleep(12 * time.Second)
	}

	for _, snapshot := range snapshots {
		if c.verbose {
			fmt.Printf("Info - Deleting snapshot: %s.\n", snapshot)
		}
		err := deleteSnapshot(svc, snapshot)

		if err != nil {
			fmt.Printf("error deleting snapshot %s. Snapshot has not been removed\n", snapshot)
		}
	}

	if c.verbose {
		fmt.Printf("All done.\n")
	}

	return RCOK
}

func cleanupAMI(svc *ec2.EC2, cleanupImage *ec2.Image) (err error) {

	ec2dii := &ec2.DeregisterImageInput{
		ImageID: cleanupImage.ImageID, // Required
	}

	td := 499
LOOPDI:

	_, err = svc.DeregisterImage(ec2dii)

	// AWS retry logic
	if err != nil {
		if reqErr, ok := err.(awserr.RequestFailure); ok {
			if scErr := reqErr.StatusCode(); scErr >= 500 && scErr < 600 {
				// if retryable then double the delay for the next run
				// if time delay > 64 seconds then give up on this request & move on
				if td = td + td; td < 64000 {
					time.Sleep(time.Duration(td) * time.Millisecond)
					// loop around and try again
					goto LOOPDI
				}
			}
		}
	}

	return
}

func deleteSnapshot(svc *ec2.EC2, snapshotID string) (err error) {

	ec2dsi := ec2.DeleteSnapshotInput{SnapshotID: aws.String(snapshotID)}

	td := 499
LOOPDSS:

	_, err = svc.DeleteSnapshot(&ec2dsi)

	// AWS retry logic
	if err != nil {
		if reqErr, ok := err.(awserr.RequestFailure); ok {
			if scErr := reqErr.StatusCode(); scErr >= 500 && scErr < 600 {
				// if retryable then double the delay for the next run
				// if time delay > 64 seconds then give up on this request & move on
				if td = td + td; td < 64000 {
					time.Sleep(time.Duration(td) * time.Millisecond)
					// loop around and try again
					goto LOOPDSS
				}
			}
		}
	}
	return
}

/*

*/
