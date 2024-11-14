package main

import (
    "context"
    "io"
    "log"
    "net/http"
    "time"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
    "github.com/aws/aws-sdk-go/service/ssm"
    "github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

func GetParameter(ctx context.Context, svc ssmiface.SSMAPI, name *string) (*ssm.GetParameterOutput, error) {
    results, err := svc.GetParameterWithContext(ctx, &ssm.GetParameterInput{
        Name:  name,
        WithDecryption: aws.Bool(true),
    })

    return results, err
}

func GetFromS3(sess *session.Session, bucket, key string) (io.ReadCloser, error) {
    svc := s3.New(sess)
    output, err := svc.GetObject(&s3.GetObjectInput{
        Bucket: aws.String(bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        return nil, err
    }
    return output.Body, nil
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
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
        // Validate input parameters
        bucket := r.URL.Query().Get("bucket")
        key := r.URL.Query().Get("key")
        if bucket == "" || key == "" {
            http.Error(w, "Parameters 'bucket' and 'key' are required", http.StatusBadRequest)
            return
        }

        // Fetch file from S3
        body, err := GetFromS3(sess, bucket, key)
        if err != nil {
            http.Error(w, "Error fetching file from S3", http.StatusInternalServerError)
            log.Printf("Error fetching file from S3: %v", err)
            return
        }
        defer body.Close()

        // Send file content to client
        w.Header().Set("Content-Type", "application/octet-stream")
        if _, err := io.Copy(w, body); err != nil {
            http.Error(w, "Error sending file", http.StatusInternalServerError)
            log.Printf("Error sending file: %v", err)
            return
        }
    })

    log.Println("Server running on port 3000")
    if err := http.ListenAndServe(":3000", nil); err != nil {
        log.Fatalf("Error starting server: %v", err)
    }
}
