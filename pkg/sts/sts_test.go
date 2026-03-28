package sts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type mockSTS struct {
	resp sts.GetCallerIdentityOutput
	err  error
}

func (m *mockSTS) GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error) {
	return &m.resp, m.err
}

func TestHandleSTS(t *testing.T) {
	mock := &mockSTS{resp: sts.GetCallerIdentityOutput{
		Account: aws.String("123456789012"),
		Arn:     aws.String("arn:aws:iam::123456789012:user/test"),
		UserId:  aws.String("AIDEXAMPLE"),
	}}

	req := httptest.NewRequest("GET", "/sts", nil)
	rr := httptest.NewRecorder()
	HandleSTS(rr, req, mock)

	if rr.Code != http.StatusOK {
		t.Errorf("got %d want %d", rr.Code, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if result["Account"] != "123456789012" {
		t.Errorf("got Account=%v want 123456789012", result["Account"])
	}
}

func TestHandleSTS_AWSError(t *testing.T) {
	mock := &mockSTS{err: fmt.Errorf("aws error")}

	req := httptest.NewRequest("GET", "/sts", nil)
	rr := httptest.NewRecorder()
	HandleSTS(rr, req, mock)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("got %d want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleSTS_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("POST", "/sts", nil)
	rr := httptest.NewRecorder()
	HandleSTS(rr, req, &mockSTS{})

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}
