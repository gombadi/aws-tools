package main

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// cleanupAMI takes pointer to the current AWS service object and a pounter to the AMI to be
// deregistered.
func describeImages(svc *ec2.EC2, ec2dii *ec2.DescribeImagesInput) (imagesResp *ec2.DescribeImagesOutput, err error) {

	td := 499
LOOP:

	imagesResp, err = svc.DescribeImages(ec2dii)

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
	return

}

// cleanupAMI takes pointer to the current AWS service object and a pounter to the AMI to be
// deregistered.
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

// deleteSnapshot function takes a pointer to the current AWS service object and
// a string containing the snapshotId to be deleted.
// It returns the AWS error
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

// chkStringValue checks if a pointer to a string is nil and ifso
// it returns a pointer to an empty string
func chkStringValue(s *string) *string {
	if s == nil {
		emptyString := ""
		s = &emptyString
	}
	return s
}
