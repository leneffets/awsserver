package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ecrpkg "github.com/leneffets/awsserver/pkg/ecr"
	s3pkg "github.com/leneffets/awsserver/pkg/s3"
	smpkg "github.com/leneffets/awsserver/pkg/secretsmanager"
	ssmpkg "github.com/leneffets/awsserver/pkg/ssm"
	stspkg "github.com/leneffets/awsserver/pkg/sts"
)

// Mock SSM
type MockSSMAPI struct {
	GetResp ssm.GetParameterOutput
	PutResp ssm.PutParameterOutput
	Err     error
}

func (m *MockSSMAPI) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	return &m.GetResp, m.Err
}

func (m *MockSSMAPI) PutParameter(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
	return &m.PutResp, m.Err
}

// Mock S3
type MockS3API struct {
	GetResp s3.GetObjectOutput
	Err     error
}

func (m *MockS3API) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return &m.GetResp, m.Err
}

func (m *MockS3API) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return &s3.PutObjectOutput{}, m.Err
}

// Mock ECR
type MockECRAPI struct {
	Resp ecr.GetAuthorizationTokenOutput
	Err  error
}

func (m *MockECRAPI) GetAuthorizationToken(ctx context.Context, params *ecr.GetAuthorizationTokenInput, optFns ...func(*ecr.Options)) (*ecr.GetAuthorizationTokenOutput, error) {
	return &m.Resp, m.Err
}

// Mock STS
type MockSTSAPI struct {
	Resp sts.GetCallerIdentityOutput
	Err  error
}

func (m *MockSTSAPI) GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	return &m.Resp, m.Err
}

// Mock Secrets Manager
type MockSMAPI struct {
	Resp secretsmanager.GetSecretValueOutput
	Err  error
}

func (m *MockSMAPI) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	return &m.Resp, m.Err
}

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("got %v want %v", rr.Code, http.StatusOK)
	}
	if rr.Body.String() != "ok" {
		t.Errorf("got %v want ok", rr.Body.String())
	}
}

func TestGetParameter(t *testing.T) {
	mock := &MockSSMAPI{
		GetResp: ssm.GetParameterOutput{
			Parameter: &ssmtypes.Parameter{Value: aws.String("mock_value")},
		},
	}

	req := httptest.NewRequest("GET", "/ssm?name=mock_name", nil)
	rr := httptest.NewRecorder()
	ssmpkg.HandleSSM(rr, req, mock)

	if rr.Code != http.StatusOK {
		t.Errorf("got %v want %v", rr.Code, http.StatusOK)
	}
	if rr.Body.String() != "mock_value" {
		t.Errorf("got %v want mock_value", rr.Body.String())
	}
}

func TestPutParameter(t *testing.T) {
	mock := &MockSSMAPI{}

	form := url.Values{}
	form.Add("name", "mock_name")
	form.Add("value", "mock_value")
	form.Add("type", "String")

	req := httptest.NewRequest("POST", "/ssm", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	ssmpkg.HandleSSM(rr, req, mock)

	if rr.Code != http.StatusOK {
		t.Errorf("got %v want %v", rr.Code, http.StatusOK)
	}
}

func TestPutParameterInvalidType(t *testing.T) {
	mock := &MockSSMAPI{}

	form := url.Values{}
	form.Add("name", "mock_name")
	form.Add("value", "mock_value")
	form.Add("type", "InvalidType")

	req := httptest.NewRequest("POST", "/ssm", strings.NewReader(form.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	ssmpkg.HandleSSM(rr, req, mock)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %v want %v", rr.Code, http.StatusBadRequest)
	}
}

func TestGetFromS3(t *testing.T) {
	mock := &MockS3API{
		GetResp: s3.GetObjectOutput{
			Body: io.NopCloser(bytes.NewReader([]byte("mock_file_content"))),
		},
	}

	req := httptest.NewRequest("GET", "/s3?bucket=mock_bucket&key=mock_key", nil)
	rr := httptest.NewRecorder()
	s3pkg.HandleS3(rr, req, mock)

	if rr.Code != http.StatusOK {
		t.Errorf("got %v want %v", rr.Code, http.StatusOK)
	}
	if rr.Body.String() != "mock_file_content" {
		t.Errorf("got %v want mock_file_content", rr.Body.String())
	}
}

func TestPutToS3(t *testing.T) {
	mock := &MockS3API{}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("mock_file_content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/s3?bucket=mock_bucket&key=mock_key", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()
	s3pkg.HandleS3(rr, req, mock)

	if rr.Code != http.StatusOK {
		t.Errorf("got %v want %v", rr.Code, http.StatusOK)
	}
}

func TestGetECRLogin(t *testing.T) {
	mock := &MockECRAPI{
		Resp: ecr.GetAuthorizationTokenOutput{
			AuthorizationData: []ecrtypes.AuthorizationData{
				{
					AuthorizationToken: aws.String("QVdTOm1vY2tfcGFzc3dvcmQ="),
					ProxyEndpoint:      aws.String("https://mock_endpoint"),
				},
			},
		},
	}

	req := httptest.NewRequest("GET", "/ecr/login", nil)
	rr := httptest.NewRecorder()
	ecrpkg.HandleECRLogin(rr, req, mock)

	if rr.Code != http.StatusOK {
		t.Errorf("got %v want %v", rr.Code, http.StatusOK)
	}
	if rr.Body.String() != "mock_password" {
		t.Errorf("got %v want mock_password", rr.Body.String())
	}
}

func TestGetCallerIdentity(t *testing.T) {
	mock := &MockSTSAPI{
		Resp: sts.GetCallerIdentityOutput{
			Account: aws.String("mock_account"),
			Arn:     aws.String("mock_arn"),
			UserId:  aws.String("mock_user_id"),
		},
	}

	req := httptest.NewRequest("GET", "/sts", nil)
	rr := httptest.NewRecorder()
	stspkg.HandleSTS(rr, req, mock)

	if rr.Code != http.StatusOK {
		t.Errorf("got %v want %v", rr.Code, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if result["Account"] != "mock_account" {
		t.Errorf("got Account=%v want mock_account", result["Account"])
	}
}

func TestGetSecret(t *testing.T) {
	mock := &MockSMAPI{
		Resp: secretsmanager.GetSecretValueOutput{
			SecretString: aws.String("super_secret"),
		},
	}

	req := httptest.NewRequest("GET", "/secrets?name=my-secret", nil)
	rr := httptest.NewRecorder()
	smpkg.HandleSecrets(rr, req, mock)

	if rr.Code != http.StatusOK {
		t.Errorf("got %v want %v", rr.Code, http.StatusOK)
	}
	if rr.Body.String() != "super_secret" {
		t.Errorf("got %v want super_secret", rr.Body.String())
	}
}

func TestMethodNotAllowed(t *testing.T) {
	mock := &MockSSMAPI{}
	req := httptest.NewRequest("DELETE", "/ssm", nil)
	rr := httptest.NewRecorder()
	ssmpkg.HandleSSM(rr, req, mock)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %v want %v", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestMissingParameters(t *testing.T) {
	mock := &MockSSMAPI{}
	req := httptest.NewRequest("GET", "/ssm", nil)
	rr := httptest.NewRecorder()
	ssmpkg.HandleSSM(rr, req, mock)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %v want %v", rr.Code, http.StatusBadRequest)
	}
}

func TestS3MissingParameters(t *testing.T) {
	mock := &MockS3API{}
	req := httptest.NewRequest("GET", "/s3", nil)
	rr := httptest.NewRecorder()
	s3pkg.HandleS3(rr, req, mock)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %v want %v", rr.Code, http.StatusBadRequest)
	}
}

func TestMain(m *testing.M) {
	os.Setenv("PORT", "3000")
	os.Exit(m.Run())
}
