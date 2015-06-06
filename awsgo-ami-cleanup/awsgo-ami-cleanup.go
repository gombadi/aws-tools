/*
This application will deregister AMI's and associated snapshots. If run in auto
mode it will cleanup AMI's and snapshots on any AMI tagged with autocleanup that
is older than supplied days old.

Command line options -
-i ami-id to be removed
-v verbose mode
-a <days> autodelete mode enabled. Delete images older that this days.

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
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/gombadi/go-rate"
)

// cmdline flag if we want verbose output
var verbose bool

func cleanupAMI(svc *ec2.EC2, cleanupImage ec2.Image) {
	if verbose {
		fmt.Printf("Info - Deregistering AMI: %s\n", *cleanupImage.ImageID)
	}

	ec2dii := &ec2.DeregisterImageInput{
		ImageID: cleanupImage.ImageID, // Required
	}

	_, err := svc.DeregisterImage(ec2dii)

	if err != nil {
		fmt.Printf("\nNon fatal error deregistering image %s...\n%v\n\n", *cleanupImage.ImageID, err)
		return
	}

	if verbose {
		fmt.Printf("Image has been deregistered and now waiting 12 seconds for AWS to release the snapshots from the AMI\n")
	}

	// after the image is deregistered then you can delete the snapshots it used
	time.Sleep(12 * time.Second)

	for blockDM := range cleanupImage.BlockDeviceMappings {
		if len(*cleanupImage.BlockDeviceMappings[blockDM].EBS.SnapshotID) > 0 {
			if verbose {
				fmt.Printf("Info - Deleting associated snapshot: %s from ami: %s\n",
					*cleanupImage.BlockDeviceMappings[blockDM].EBS.SnapshotID,
					*cleanupImage.ImageID)
			}

			ec2dsi := ec2.DeleteSnapshotInput{SnapshotID: cleanupImage.BlockDeviceMappings[blockDM].EBS.SnapshotID}

			_, err := svc.DeleteSnapshot(&ec2dsi)
			if err != nil {
				fmt.Printf("\nError deleting the snapshots %s.\n%v\n\n", *cleanupImage.BlockDeviceMappings[blockDM].EBS.SnapshotID, err)
			}
		}
	}
	if verbose {
		fmt.Printf("Snapshots have been deleted.\n")
	}
}

func main() {

	// storage for commandline args
	var autoDays int
	var amiId string

	flag.BoolVar(&verbose, "v", false, "Produce verbose output")
	flag.IntVar(&autoDays, "a", 0, "In auto cleanup mode cleanup any AMI's older than this number of days")
	flag.StringVar(&amiId, "i", "", "AMI Id to be deleted")
	flag.Parse()

	// load the AWS credentials from the environment or from the standard file

	// make sure we are in auto mode or an ami id has been provided
	if autoDays == 0 && len(amiId) == 0 {
		fmt.Printf("No ami details provided. Please provide an ami-id to cleanup\nor enable auto cleanup mode and specify a number of days.\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Create an EC2 service object
	// config values keys, sercet key & region read from environment
	svc := ec2.New(&aws.Config{})

	ec2Filter := ec2.Filter{}

	// pointers to objects we use to talk to AWS
	var imagesResp *ec2.DescribeImagesOutput
	var err error

	// config for auto mode
	if autoDays > 0 {

		// auto mode search for ami's to cleanup
		ec2Filter.Name = aws.String("tag-key")
		ec2Filter.Values = []*string{aws.String("autocleanup")}
		owners := []*string{aws.String("self")}

		ec2dii := ec2.DescribeImagesInput{Owners: owners, Filters: []*ec2.Filter{&ec2Filter}}

		imagesResp, err = svc.DescribeImages(&ec2dii)
		if err != nil {
			log.Fatalf("\nError getting the Image details for images.\n%v\n", err)
		}
	} else {
		// single ami manual mode
		ec2dii := ec2.DescribeImagesInput{ImageIDs: []*string{aws.String(amiId)}}

		imagesResp, err = svc.DescribeImages(&ec2dii)
		if err != nil {
			log.Fatalf("\nError getting the Image details for image %s...\n%v\n", amiId, err)
		}
	}

	if len(imagesResp.Images) == 0 {
		if verbose {
			fmt.Printf("No images found to cleanup. Exiting\n")
		}
		os.Exit(0)
	}

	// use go routines to deregister many at once and
	// use a waitgroup to sync it all
	var wg sync.WaitGroup

	// rate limit the AWS requests to max 3 per second
	rl := rate.New(3, time.Second)

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

				if int64(autoDays*86000) < amiLifeSpan {

					anImage := imagesResp.Images[image]

					rl.Wait()

					// Increment the WaitGroup counter.
					wg.Add(1)

					// Launch a goroutine to cleanup the AMI.
					go func(svc *ec2.EC2, anImage ec2.Image) {
						// Decrement the counter when the goroutine completes.
						defer wg.Done()
						// Fetch the URL.
						cleanupAMI(svc, anImage)
					}(svc, *anImage)

					// deregister the AMI and delete associated snapshots
					//cleanupAMI(e, imagesResp.Images[image])
					//fmt.Printf("\nWould have cleaned up AMI: %s\n\n", imagesResp.Images[image].Id)
				} else {
					if verbose {
						fmt.Printf("Info - Not deregistering AMI: %s as expire time not reached\n", *imagesResp.Images[image].ImageID)
					}
				}
			}
		}
	}

	// Wait for all Amazon requests to complete.
	wg.Wait()

	if verbose {
		fmt.Printf("All done.\n")
	}

}
