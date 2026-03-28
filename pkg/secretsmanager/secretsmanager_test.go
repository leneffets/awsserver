package secretsmanager

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type mockSM struct {
	resp secretsmanager.GetSecretValueOutput
	err  error
}

func (m *mockSM) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	return &m.resp, m.err
}

func TestHandleSecrets(t *testing.T) {
	mock := &mockSM{resp: secretsmanager.GetSecretValueOutput{
		SecretString: aws.String(`{"user":"admin","pass":"secret"}`),
	}}

	req := httptest.NewRequest("GET", "/secrets?name=my-secret", nil)
	rr := httptest.NewRecorder()
	HandleSecrets(rr, req, mock)

	if rr.Code != http.StatusOK {
		t.Errorf("got %d want %d", rr.Code, http.StatusOK)
	}
	if rr.Body.String() != `{"user":"admin","pass":"secret"}` {
		t.Errorf("got %q want JSON secret", rr.Body.String())
	}
}

func TestHandleSecrets_Binary(t *testing.T) {
	mock := &mockSM{resp: secretsmanager.GetSecretValueOutput{
		SecretBinary: []byte{0x01, 0x02, 0x03},
	}}

	req := httptest.NewRequest("GET", "/secrets?name=my-binary-secret", nil)
	rr := httptest.NewRecorder()
	HandleSecrets(rr, req, mock)

	if rr.Code != http.StatusOK {
		t.Errorf("got %d want %d", rr.Code, http.StatusOK)
	}
	if rr.Header().Get("Content-Type") != "application/octet-stream" {
		t.Errorf("got Content-Type %q want application/octet-stream", rr.Header().Get("Content-Type"))
	}
}

func TestHandleSecrets_MissingName(t *testing.T) {
	req := httptest.NewRequest("GET", "/secrets", nil)
	rr := httptest.NewRecorder()
	HandleSecrets(rr, req, &mockSM{})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %d want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleSecrets_AWSError(t *testing.T) {
	mock := &mockSM{err: fmt.Errorf("aws error")}

	req := httptest.NewRequest("GET", "/secrets?name=bad", nil)
	rr := httptest.NewRecorder()
	HandleSecrets(rr, req, mock)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("got %d want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandleSecrets_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("POST", "/secrets", nil)
	rr := httptest.NewRecorder()
	HandleSecrets(rr, req, &mockSM{})

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}
