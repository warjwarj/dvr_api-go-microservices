package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	config "dvr_api-go-microservices/pkg/config"
	utils "dvr_api-go-microservices/pkg/utils"

	zap "go.uber.org/zap"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/smithy-go"
)

// PresignClient, a client that is used to presign requests to Amazon S3.
// Presigned requests contain temporary credentials and can be made from any HTTP client.
type AwsConnection struct { // static
	logger        *zap.Logger
	s3Client      *s3.Client
	presignClient *s3.PresignClient
}

func NewAwsConnection(logger *zap.Logger) (*AwsConnection, error) {
	// this loads access keys from env vars
	cfg, err := awsConfig.LoadDefaultConfig(context.TODO(), awsConfig.WithRegion("eu-west-2"))
	if err != nil {
		return nil, fmt.Errorf("error loading AWS config: ", err)
	}
	// create s3 and presign client objects
	client := s3.NewFromConfig(cfg)
	presignClient := s3.NewPresignClient(client)
	// return our wrapper
	return &AwsConnection{
		logger:        logger,
		s3Client:      client,
		presignClient: presignClient,
	}, nil
}

// upload video files to S3 and return links to them.
func (au *AwsConnection) UploadVideoToS3(folderPath string, receivedVideoLog map[string]utils.VideoDescription) error {
	// list files in dir
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		log.Fatal("Couldn't open file: ", err)
	}

	// for every file in the folder upload to the folder
	for _, e := range entries {
		// parse the channel num from the file name
		chanNum := strings.Split(strings.Split(e.Name(), "_")[3], ".")[0]

		// // upload the video to the cloud and get a link to it
		// filepath := folderPath + "/" + e.Name()
		// if err := au.uploadFile(context.TODO(), config.VIDEO_STORAGE_BUCKET, e.Name(), filepath); err != nil {
		// 	au.logger.Error("Couldn't upload file to AWS", zap.Error(err))
		// 	continue
		// }

		// get a presigned url
		presigned, err := au.getPresignedUrl(context.TODO(), config.VIDEO_STORAGE_BUCKET, e.Name(), 3600)
		if err != nil {
			au.logger.Error("Couldn't get presigned URL from AWS", zap.Error(err))
			continue
		}

		// put the video link into the video description
		if vidDesc, exists := receivedVideoLog[chanNum]; exists {
			vidDesc.VideoLink = presigned.URL
			receivedVideoLog[chanNum] = vidDesc
		}
	}
	return nil
}

// upload a given file to AWS
func (au *AwsConnection) uploadFile(
	ctx context.Context,
	bucketName string,
	objectKey string,
	filePathToUpload string,
) error {
	file, err := os.Open(filePathToUpload)
	if err != nil {
		return fmt.Errorf("Couldn't open file %v to upload. Here's why: %v\n", filePathToUpload, err)
	} else {
		defer file.Close()
		_, err = au.s3Client.PutObject(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(objectKey),
			Body:   file,
		})
		if err != nil {
			var apiErr smithy.APIError
			if errors.As(err, &apiErr) && apiErr.ErrorCode() == "EntityTooLarge" {
				log.Printf("Error while uploading object to %s. The object is too large.\n"+
					"To upload objects larger than 5GB, use the S3 console (160GB max)\n"+
					"or the multipart upload API (5TB max).", bucketName)
			} else {
				log.Printf("Couldn't upload file %v to %v:%v. Here's why: %v\n",
					filePathToUpload, bucketName, objectKey, err)
			}
		} else {
			err = s3.NewObjectExistsWaiter(au.s3Client).Wait(
				ctx, &s3.HeadObjectInput{Bucket: aws.String(bucketName), Key: aws.String(objectKey)}, time.Minute)
			if err != nil {
				log.Printf("Failed attempt to wait for object %s to exist.\n", objectKey)
			}
		}
	}
	return err
}

// getPresignedUrl makes a presigned request that can be used to get an object from a bucket.
// The presigned request is valid for the specified number of seconds.
func (au *AwsConnection) getPresignedUrl(
	ctx context.Context,
	bucketName string,
	objectKey string,
	lifetimeSecs int64,
) (*v4.PresignedHTTPRequest, error) {
	request, err := au.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
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
