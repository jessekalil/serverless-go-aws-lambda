package main

import (
	"bytes"
	"context"
	"github.com/aws/aws-lambda-go/events"
	runtime "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
	_ "image/png"
	"path/filepath"
	"strings"
)

type LambdaResponse struct {
	status  int
	message string
}

var awsSession *session.Session
var s3Client *s3.S3

func init() {
	awsSession = session.Must(session.NewSession())
	s3Client = s3.New(awsSession)
}

func imageProcess(img image.Image) image.Image {
	return resize.Thumbnail(1280, 720, img, resize.Lanczos3)
}

func getS3Object(bucket, key string) (object *s3.GetObjectOutput) {
	object, err := s3Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		panic(err)
	}
	return
}

func putS3Object(bucket, key, contentType string, reader *bytes.Reader) {
	_, err := s3Client.PutObject(&s3.PutObjectInput{
		Body:        aws.ReadSeekCloser(reader),
		Key:         aws.String(key),
		Bucket:      aws.String(bucket),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		panic(err)
	}
}

func handleRecord(record events.S3EventRecord) error {
	bucket := record.S3.Bucket.Name
	key := record.S3.Object.Key

	object := getS3Object(bucket, key)

	img, _, err := image.Decode(object.Body)
	if err != nil {
		return err
	}

	img = imageProcess(img)

	var buff bytes.Buffer
	err = jpeg.Encode(&buff, img, &jpeg.Options{Quality: 50})
	if err != nil {
		return err
	}

	filename := filepath.Base(key)
	newFilename := strings.TrimSuffix(filename, filepath.Ext(filename)) + ".jpeg"
	newKey := "compressed/" + newFilename

	putS3Object(bucket, newKey, "image/jpeg", bytes.NewReader(buff.Bytes()))

	return nil
}

func handleRequest(_ context.Context, event events.S3Event) (LambdaResponse, error) {
	for _, record := range event.Records {
		err := handleRecord(record)
		if err != nil {
			return LambdaResponse{500, err.Error()}, nil
		}
	}

	return LambdaResponse{200, "Success"}, nil
}

func main() {
	runtime.Start(handleRequest)
}
