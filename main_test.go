package main

import (
    "bytes"
    "context"
    "io/ioutil"
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
    "github.com/aws/aws-sdk-go/service/s3/s3iface"
    "github.com/aws/aws-sdk-go/service/ssm"
    "github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

// MockSSMAPI for testing
type MockSSMAPI struct {
    ssmiface.SSMAPI
    Response ssm.GetParameterOutput
    Err      error
}

func (m *MockSSMAPI) GetParameterWithContext(ctx context.Context, input *ssm.GetParameterInput, opts ...func(*ssm.Request)) (*ssm.GetParameterOutput, error) {
    return &m.Response, m.Err
}

// MockS3API for testing
type MockS3API struct {
    s3iface.S3API
    Response s3.GetObjectOutput
    Err      error
}

func (m *MockS3API) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
    return &m.Response, m.Err
}

func TestGetParameter(t *testing.T) {
    mockSvc := &MockSSMAPI{
        Response: ssm.GetParameterOutput{
            Parameter: &ssm.Parameter{
                Value: aws.String("mock_value"),
            },
        },
    }

    req, err := http.NewRequest("GET", "/ssm?name=mock_name", nil)
    if err != nil {
        t.Fatal(err)
    }

    rr := httptest.NewRecorder()
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := r.URL.Query().Get("name")
        if id == "" {
            http.Error(w, "Parameter 'name' is required", http.StatusBadRequest)
            return
        }

        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        results, err := GetParameter(ctx, mockSvc, &id)
        if err != nil {
            http.Error(w, "Error fetching parameter", http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "text/plain")
        w.Write([]byte(*results.Parameter.Value))
    })

    handler.ServeHTTP(rr, req)

    if status := rr.Code; status != http.StatusOK {
        t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
    }

    expected := "mock_value"
    if rr.Body.String() != expected {
        t.Errorf("Handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
    }
}

func TestGetFromS3(t *testing.T) {
    mockS3 := &MockS3API{
        Response: s3.GetObjectOutput{
            Body: ioutil.NopCloser(bytes.NewReader([]byte("mock_file_content"))),
        },
    }

    sess := session.Must(session.NewSessionWithOptions(session.Options{
        SharedConfigState: session.SharedConfigEnable,
    }))

    req, err := http.NewRequest("GET", "/s3?bucket=mock_bucket&key=mock_key", nil)
    if err != nil {
        t.Fatal(err)
    }

    rr := httptest.NewRecorder()
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        bucket := r.URL.Query().Get("bucket")
        key := r.URL.Query().Get("key")
        if bucket == "" || key == "" {
            http.Error(w, "Parameters 'bucket' and 'key' are required", http.StatusBadRequest)
            return
        }

        body, err := GetFromS3(sess, bucket, key)
        if err != nil {
            http.Error(w, "Error fetching file from S3", http.StatusInternalServerError)
            return
        }
        defer body.Close()

        w.Header().Set("Content-Type", "application/octet-stream")
        if _, err := io.Copy(w, body); err != nil {
            http.Error(w, "Error sending file", http.StatusInternalServerError)
            return
        }
    })

    handler.ServeHTTP(rr, req)

    if status := rr.Code; status != http.StatusOK {
        t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
    }

    expected := "mock_file_content"
    if rr.Body.String() != expected {
        t.Errorf("Handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
    }
}
