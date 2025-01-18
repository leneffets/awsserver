package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecr/ecriface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	ssmpkg "github.com/leneffets/ssmserver/pkg/ssm"
)

// MockSSMAPI for testing
type MockSSMAPI struct {
	ssmiface.SSMAPI
	Response    ssm.GetParameterOutput
	PutResponse ssm.PutParameterOutput
	Err         error
}

func (m *MockSSMAPI) GetParameterWithContext(ctx context.Context, input *ssm.GetParameterInput, opts ...request.Option) (*ssm.GetParameterOutput, error) {
	return &m.Response, m.Err
}

func (m *MockSSMAPI) PutParameterWithContext(ctx context.Context, input *ssm.PutParameterInput, opts ...request.Option) (*ssm.PutParameterOutput, error) {
	return &m.PutResponse, m.Err
}

// MockS3API for testing
type MockS3API struct {
	s3iface.S3API
	Response s3.GetObjectOutput
	Err      error
}

func (m *MockS3API) GetObjectWithContext(ctx context.Context, input *s3.GetObjectInput, opts ...request.Option) (*s3.GetObjectOutput, error) {
	return &m.Response, m.Err
}

func (m *MockS3API) PutObjectWithContext(ctx context.Context, input *s3.PutObjectInput, opts ...request.Option) (*s3.PutObjectOutput, error) {
	return &s3.PutObjectOutput{}, m.Err
}

// MockECRAPI for testing
type MockECRAPI struct {
	ecriface.ECRAPI
	Response ecr.GetAuthorizationTokenOutput
	Err      error
}

func (m *MockECRAPI) GetAuthorizationTokenWithContext(ctx context.Context, input *ecr.GetAuthorizationTokenInput, opts ...request.Option) (*ecr.GetAuthorizationTokenOutput, error) {
	return &m.Response, m.Err
}

// MockSTSAPI for testing
type MockSTSAPI struct {
	stsiface.STSAPI
	Response sts.GetCallerIdentityOutput
	Err      error
}

func (m *MockSTSAPI) GetCallerIdentityWithContext(ctx context.Context, input *sts.GetCallerIdentityInput, opts ...request.Option) (*sts.GetCallerIdentityOutput, error) {
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

		results, err := ssmpkg.GetParameter(ctx, mockSvc, &id)
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

func TestPutParameter(t *testing.T) {
	mockSvc := &MockSSMAPI{}

	form := url.Values{}
	form.Add("name", "mock_name")
	form.Add("value", "mock_value")
	form.Add("type", "String")

	req, err := http.NewRequest("POST", "/ssm", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ssmpkg.HandlePostSSM(w, r, mockSvc)
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestGetFromS3(t *testing.T) {
	mockS3 := &MockS3API{
		Response: s3.GetObjectOutput{
			Body: ioutil.NopCloser(bytes.NewReader([]byte("mock_file_content"))),
		},
	}

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

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		body, err := mockS3.GetObjectWithContext(ctx, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			http.Error(w, "Error fetching file from S3", http.StatusInternalServerError)
			return
		}
		defer body.Body.Close()

		w.Header().Set("Content-Type", "application/octet-stream")
		if _, err := io.Copy(w, body.Body); err != nil {
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

func TestPutToS3(t *testing.T) {
	mockS3 := &MockS3API{}

	fileContent := "mock_file_content"
	file := ioutil.NopCloser(bytes.NewReader([]byte(fileContent)))

	req, err := http.NewRequest("POST", "/s3?bucket=mock_bucket&key=mock_key", file)
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

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Convert io.ReadCloser to io.ReadSeeker
		readSeeker := bytes.NewReader([]byte(fileContent))

		_, err = mockS3.PutObjectWithContext(ctx, &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Body:   readSeeker,
		})
		if err != nil {
			http.Error(w, "Error uploading file to S3", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestGetECRLogin(t *testing.T) {
	mockECR := &MockECRAPI{
		Response: ecr.GetAuthorizationTokenOutput{
			AuthorizationData: []*ecr.AuthorizationData{
				{
					AuthorizationToken: aws.String("mock_token"),
					ProxyEndpoint:      aws.String("https://mock_endpoint"),
				},
			},
		},
	}

	req, err := http.NewRequest("GET", "/ecr/login", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := mockECR.GetAuthorizationTokenWithContext(ctx, &ecr.GetAuthorizationTokenInput{})
		if err != nil {
			http.Error(w, "Error fetching ECR authorization token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := ecr.GetAuthorizationTokenOutput{
		AuthorizationData: []*ecr.AuthorizationData{
			{
				AuthorizationToken: aws.String("mock_token"),
				ProxyEndpoint:      aws.String("https://mock_endpoint"),
				ExpiresAt:          nil,
			},
		},
	}

	var actual ecr.GetAuthorizationTokenOutput
	if err := json.Unmarshal(rr.Body.Bytes(), &actual); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Handler returned unexpected body: got %v want %v", actual, expected)
	}
}

func TestGetCallerIdentity(t *testing.T) {
	mockSTS := &MockSTSAPI{
		Response: sts.GetCallerIdentityOutput{
			Account: aws.String("mock_account"),
			Arn:     aws.String("mock_arn"),
			UserId:  aws.String("mock_user_id"),
		},
	}

	req, err := http.NewRequest("GET", "/sts", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := mockSTS.GetCallerIdentityWithContext(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			http.Error(w, "Error fetching caller identity", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := sts.GetCallerIdentityOutput{
		Account: aws.String("mock_account"),
		Arn:     aws.String("mock_arn"),
		UserId:  aws.String("mock_user_id"),
	}

	var actual sts.GetCallerIdentityOutput
	if err := json.Unmarshal(rr.Body.Bytes(), &actual); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Handler returned unexpected body: got %v want %v", actual, expected)
	}
}

func TestMain(m *testing.M) {
	os.Setenv("PORT", "3000")
	code := m.Run()
	os.Exit(code)
}
