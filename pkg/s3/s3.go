package s3

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func GetFromS3(ctx context.Context, sess *session.Session, bucket, key string) (io.ReadCloser, error) {
	svc := s3.New(sess)
	output, err := svc.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return output.Body, nil
}

func PutToS3(ctx context.Context, sess *session.Session, bucket, key string, body io.ReadSeeker) error {
	svc := s3.New(sess)
	_, err := svc.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   body,
	})
	return err
}

func HandleS3(w http.ResponseWriter, r *http.Request, sess *session.Session) {
	bucket := r.URL.Query().Get("bucket")
	key := r.URL.Query().Get("key")
	if bucket == "" || key == "" {
		http.Error(w, "Parameters 'bucket' and 'key' are required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if r.Method == http.MethodGet {
		handleGetS3(w, r, sess, bucket, key, ctx)
	} else if r.Method == http.MethodPost {
		handlePostS3(w, r, sess, bucket, key, ctx)
	} else {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
	}
}

func handleGetS3(w http.ResponseWriter, r *http.Request, sess *session.Session, bucket, key string, ctx context.Context) {
	body, err := GetFromS3(ctx, sess, bucket, key)
	if err != nil {
		http.Error(w, "Error fetching file from S3", http.StatusInternalServerError)
		log.Printf("Error fetching file from S3: %v", err)
		return
	}
	defer body.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := io.Copy(w, body); err != nil {
		http.Error(w, "Error sending file", http.StatusInternalServerError)
		log.Printf("Error sending file: %v", err)
		return
	}
}

func handlePostS3(w http.ResponseWriter, r *http.Request, sess *session.Session, bucket, key string, ctx context.Context) {
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error reading uploaded file", http.StatusBadRequest)
		log.Printf("Error reading uploaded file: %v", err)
		return
	}
	defer file.Close()

	tempFile, err := os.CreateTemp("", "upload-*.tmp")
	if err != nil {
		http.Error(w, "Error creating temporary file", http.StatusInternalServerError)
		log.Printf("Error creating temporary file: %v", err)
		return
	}
	defer os.Remove(tempFile.Name())

	if _, err := io.Copy(tempFile, file); err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		log.Printf("Error saving file: %v", err)
		return
	}

	tempFile.Seek(0, 0)

	if err := PutToS3(ctx, sess, bucket, key, tempFile); err != nil {
		http.Error(w, "Error uploading file to S3", http.StatusInternalServerError)
		log.Printf("Error uploading file to S3: %v", err)
		return
	}

	log.Printf("File uploaded successfully to bucket %s with key %s", bucket, key)
	w.WriteHeader(http.StatusOK)
}
