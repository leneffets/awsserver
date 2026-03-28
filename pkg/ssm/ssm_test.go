package ssm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type mockSSM struct {
	getResp ssm.GetParameterOutput
	putResp ssm.PutParameterOutput
	err     error
}

func (m *mockSSM) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	return &m.getResp, m.err
}

func (m *mockSSM) PutParameter(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
	return &m.putResp, m.err
}

func TestHandleGetSSM(t *testing.T) {
	mock := &mockSSM{getResp: ssm.GetParameterOutput{
		Parameter: &types.Parameter{Value: aws.String("test_value")},
	}}

	req := httptest.NewRequest("GET", "/ssm?name=test", nil)
	rr := httptest.NewRecorder()
	HandleSSM(rr, req, mock)

	if rr.Code != http.StatusOK {
		t.Errorf("got %d want %d", rr.Code, http.StatusOK)
	}
	if rr.Body.String() != "test_value" {
		t.Errorf("got %q want test_value", rr.Body.String())
	}
}

func TestHandleGetSSM_MissingName(t *testing.T) {
	req := httptest.NewRequest("GET", "/ssm", nil)
	rr := httptest.NewRecorder()
	HandleSSM(rr, req, &mockSSM{})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %d want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleGetSSM_AWSError(t *testing.T) {
	mock := &mockSSM{err: fmt.Errorf("aws error")}

	req := httptest.NewRequest("GET", "/ssm?name=test", nil)
	rr := httptest.NewRecorder()
	HandleSSM(rr, req, mock)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("got %d want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandlePostSSM(t *testing.T) {
	mock := &mockSSM{}
	form := url.Values{"name": {"p"}, "value": {"v"}, "type": {"SecureString"}}

	req := httptest.NewRequest("POST", "/ssm", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	HandleSSM(rr, req, mock)

	if rr.Code != http.StatusOK {
		t.Errorf("got %d want %d", rr.Code, http.StatusOK)
	}
}

func TestHandlePostSSM_InvalidType(t *testing.T) {
	form := url.Values{"name": {"p"}, "value": {"v"}, "type": {"BadType"}}

	req := httptest.NewRequest("POST", "/ssm", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	HandleSSM(rr, req, &mockSSM{})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %d want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandlePostSSM_MissingFields(t *testing.T) {
	form := url.Values{"name": {"p"}}

	req := httptest.NewRequest("POST", "/ssm", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	HandleSSM(rr, req, &mockSSM{})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %d want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleSSM_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("DELETE", "/ssm", nil)
	rr := httptest.NewRecorder()
	HandleSSM(rr, req, &mockSSM{})

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}
