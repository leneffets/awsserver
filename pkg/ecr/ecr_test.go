package ecr

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

type mockECR struct {
	resp ecr.GetAuthorizationTokenOutput
	err  error
}

func (m *mockECR) GetAuthorizationToken(ctx context.Context, params *ecr.GetAuthorizationTokenInput, optFns ...func(*ecr.Options)) (*ecr.GetAuthorizationTokenOutput, error) {
	return &m.resp, m.err
}

func TestHandleECRLogin(t *testing.T) {
	mock := &mockECR{resp: ecr.GetAuthorizationTokenOutput{
		AuthorizationData: []types.AuthorizationData{
			{AuthorizationToken: aws.String("QVdTOnRlc3RfcGFzcw==")}, // base64("AWS:test_pass")
		},
	}}

	req := httptest.NewRequest("GET", "/ecr/login", nil)
	rr := httptest.NewRecorder()
	HandleECRLogin(rr, req, mock)

	if rr.Code != http.StatusOK {
		t.Errorf("got %d want %d", rr.Code, http.StatusOK)
	}
	if rr.Body.String() != "test_pass" {
		t.Errorf("got %q want test_pass", rr.Body.String())
	}
}

func TestHandleECRLogin_AWSError(t *testing.T) {
	mock := &mockECR{err: fmt.Errorf("aws error")}

	req := httptest.NewRequest("GET", "/ecr/login", nil)
	rr := httptest.NewRecorder()
	HandleECRLogin(rr, req, mock)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("got %d want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleECRLogin_NoAuthData(t *testing.T) {
	mock := &mockECR{resp: ecr.GetAuthorizationTokenOutput{}}

	req := httptest.NewRequest("GET", "/ecr/login", nil)
	rr := httptest.NewRecorder()
	HandleECRLogin(rr, req, mock)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("got %d want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleECRLogin_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("POST", "/ecr/login", nil)
	rr := httptest.NewRecorder()
	HandleECRLogin(rr, req, &mockECR{})

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}
