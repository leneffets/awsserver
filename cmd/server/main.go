package main

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

// Funktion, um SSM-Parameter abzurufen
func GetParameter(ctx context.Context, svc ssmiface.SSMAPI, name *string) (*ssm.GetParameterOutput, error) {
	results, err := svc.GetParameterWithContext(ctx, &ssm.GetParameterInput{
		Name:           name,
		WithDecryption: aws.Bool(true),
	})
	return results, err
}

// Funktion, um ein Objekt aus S3 zu holen
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

// Funktion, um ein Objekt in S3 zu legen
func PutToS3(ctx context.Context, sess *session.Session, bucket, key string, body io.ReadSeeker) error {
	svc := s3.New(sess)
	_, err := svc.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   body,
	})
	return err
}

func main() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := ssm.New(sess)

	creds, err := sess.Config.Credentials.Get()
	if err != nil {
		log.Fatalf("Failed to get credentials: %v", err)
	}
	log.Printf("Using credentials: %s/%s\n", creds.AccessKeyID, creds.SecretAccessKey)

	http.HandleFunc("/ssm", func(w http.ResponseWriter, r *http.Request) {
		// Validate input parameter
		id := r.URL.Query().Get("name")
		if id == "" {
			http.Error(w, "Parameter 'name' is required", http.StatusBadRequest)
			return
		}

		// Set context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Fetch parameter
		results, err := GetParameter(ctx, svc, &id)
		if err != nil {
			log.Printf("Error fetching parameter: %v", err)
			http.Error(w, "Error fetching parameter", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(*results.Parameter.Value))
	})

	http.HandleFunc("/s3", func(w http.ResponseWriter, r *http.Request) {
		bucket := r.URL.Query().Get("bucket")
		key := r.URL.Query().Get("key")
		if bucket == "" || key == "" {
			http.Error(w, "Parameters 'bucket' and 'key' are required", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if r.Method == http.MethodGet {
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
		} else if r.Method == http.MethodPost {
			file, _, err := r.FormFile("file")
			if err != nil {
				http.Error(w, "Error reading uploaded file", http.StatusBadRequest)
				log.Printf("Error reading uploaded file: %v", err)
				return
			}
			defer file.Close()

			tempFile, err := ioutil.TempFile("", "upload-*.tmp")
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

			w.WriteHeader(http.StatusOK)
			log.Printf("File uploaded successfully to bucket %s with key %s", bucket, key)
		} else {
			http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server running on port %s", port)
	if err := http.ListenAndServe(":" + port, nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
