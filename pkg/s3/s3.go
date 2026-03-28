package s3

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3API defines the interface for S3 operations used by this package.
type S3API interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

func GetFromS3(ctx context.Context, svc S3API, bucket, key string) (io.ReadCloser, error) {
	output, err := svc.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return output.Body, nil
}

func PutToS3(ctx context.Context, svc S3API, bucket, key string, body io.Reader) error {
	_, err := svc.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   body,
	})
	return err
}

func HandleS3(w http.ResponseWriter, r *http.Request, svc S3API) {
	bucket := r.URL.Query().Get("bucket")
	key := r.URL.Query().Get("key")
	if bucket == "" || key == "" {
		http.Error(w, "Parameters 'bucket' and 'key' are required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleGetS3(w, r, svc, bucket, key)
	case http.MethodPost:
		handlePostS3(w, r, svc, bucket, key)
	default:
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
	}
}

func handleGetS3(w http.ResponseWriter, r *http.Request, svc S3API, bucket, key string) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	body, err := GetFromS3(ctx, svc, bucket, key)
	if err != nil {
		http.Error(w, "Error fetching file from S3", http.StatusInternalServerError)
		slog.Error("failed to fetch file from S3", "error", err)
		return
	}
	defer body.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := io.Copy(w, body); err != nil {
		slog.Error("failed to send file", "error", err)
	}
}

func handlePostS3(w http.ResponseWriter, r *http.Request, svc S3API, bucket, key string) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error reading uploaded file", http.StatusBadRequest)
		slog.Error("failed to read uploaded file", "error", err)
		return
	}
	defer file.Close()

	tempFile, err := os.CreateTemp("", "upload-*.tmp")
	if err != nil {
		http.Error(w, "Error creating temporary file", http.StatusInternalServerError)
		slog.Error("failed to create temp file", "error", err)
		return
	}
	defer tempFile.Close()
	defer os.Remove(tempFile.Name())

	if _, err := io.Copy(tempFile, file); err != nil {
		http.Error(w, "Error saving file", http.StatusInternalServerError)
		slog.Error("failed to save file", "error", err)
		return
	}

	tempFile.Seek(0, 0)

	if err := PutToS3(ctx, svc, bucket, key, tempFile); err != nil {
		http.Error(w, "Error uploading file to S3", http.StatusInternalServerError)
		slog.Error("failed to upload file to S3", "error", err)
		return
	}

	slog.Info("file uploaded", "bucket", bucket, "key", key)
	w.WriteHeader(http.StatusOK)
}
