package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	utils "dvr_api-go-microservices/pkg/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// It contains PresignClient, a client that is used to presign requests to Amazon S3.
// Presigned requests contain temporary credentials and can be made from any HTTP client.
type AwsUtils struct {
	PresignClient *s3.PresignClient
}

// upload video files to S3 and return links to them.
func uploadVideoToS3(folderPath string, receivedVideoLog map[string]utils.VideoDescription) error {
	// list files in dir
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		log.Fatal(err)
	}

	// for every file in the folder upload to the folder
	for _, e := range entries {
		// parse the channel num from the file name
		chanNum := strings.Split(strings.Split(e.Name(), "_")[3], ".")[0]

		// upload the video to the cloud and get a link to it
		awsLink := "soijsoijsoij"

		// put the video link into the video description
		if vidDesc, exists := receivedVideoLog[chanNum]; exists {
			vidDesc.VideoLink = awsLink
			receivedVideoLog[chanNum] = vidDesc
		}
	}
	return nil
}

// GetObject makes a presigned request that can be used to get an object from a bucket.
// The presigned request is valid for the specified number of seconds.
func (a *AwsUtils) GetObject(
	ctx context.Context,
	bucketName string,
	objectKey string,
	lifetimeSecs int64,
) (*v4.PresignedHTTPRequest, error) {
	request, err := a.PresignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(lifetimeSecs * int64(time.Second))
	})
	if err != nil {
		log.Printf("Couldn't get a presigned request to get %v:%v. Here's why: %v\n",
			bucketName, objectKey, err)
	}
	return request, err
}

// PutObject makes a presigned request that can be used to put an object in a bucket.
// The presigned request is valid for the specified number of seconds.
func (a *AwsUtils) PutObject(
	ctx context.Context,
	bucketName string,
	objectKey string,
	lifetimeSecs int64,
) (*v4.PresignedHTTPRequest, error) {
	request, err := a.PresignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(lifetimeSecs * int64(time.Second))
	})
	if err != nil {
		log.Printf("Couldn't get a presigned request to put %v:%v. Here's why: %v\n",
			bucketName, objectKey, err)
	}
	return request, err
}

// DeleteObject makes a presigned request that can be used to delete an object from a bucket.
func (a *AwsUtils) DeleteObject(
	ctx context.Context,
	bucketName string,
	objectKey string,
) (*v4.PresignedHTTPRequest, error) {
	request, err := a.PresignClient.PresignDeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		log.Printf("Couldn't get a presigned request to delete object %v. Here's why: %v\n", objectKey, err)
	}
	return request, err
}

func (a *AwsUtils) PresignPostObject(
	ctx context.Context,
	bucketName string,
	objectKey string,
	lifetimeSecs int64,
) (*s3.PresignedPostRequest, error) {
	request, err := a.PresignClient.PresignPostObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}, func(options *s3.PresignPostOptions) {
		options.Expires = time.Duration(lifetimeSecs) * time.Second
	})
	if err != nil {
		log.Printf("Couldn't get a presigned post request to put %v:%v. Here's why: %v\n", bucketName, objectKey, err)
	}
	return request, nil
}
