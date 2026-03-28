package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type mockS3 struct {
	getResp s3.GetObjectOutput
	err     error
}

func (m *mockS3) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return &m.getResp, m.err
}

func (m *mockS3) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return &s3.PutObjectOutput{}, m.err
}

func TestHandleGetS3(t *testing.T) {
	mock := &mockS3{getResp: s3.GetObjectOutput{
		Body: io.NopCloser(bytes.NewReader([]byte("file_content"))),
	}}

	req := httptest.NewRequest("GET", "/s3?bucket=b&key=k", nil)
	rr := httptest.NewRecorder()
	HandleS3(rr, req, mock)

	if rr.Code != http.StatusOK {
		t.Errorf("got %d want %d", rr.Code, http.StatusOK)
	}
	if rr.Body.String() != "file_content" {
		t.Errorf("got %q want file_content", rr.Body.String())
	}
}

func TestHandleGetS3_MissingParams(t *testing.T) {
	req := httptest.NewRequest("GET", "/s3?bucket=b", nil)
	rr := httptest.NewRecorder()
	HandleS3(rr, req, &mockS3{})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %d want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleGetS3_AWSError(t *testing.T) {
	mock := &mockS3{err: fmt.Errorf("aws error")}

	req := httptest.NewRequest("GET", "/s3?bucket=b&key=k", nil)
	rr := httptest.NewRecorder()
	HandleS3(rr, req, mock)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("got %d want %d", rr.Code, http.StatusInternalServerError)
	}
}

func TestHandlePostS3(t *testing.T) {
	mock := &mockS3{}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "test.txt")
	part.Write([]byte("content"))
	w.Close()

	req := httptest.NewRequest("POST", "/s3?bucket=b&key=k", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()
	HandleS3(rr, req, mock)

	if rr.Code != http.StatusOK {
		t.Errorf("got %d want %d", rr.Code, http.StatusOK)
	}
}

func TestHandlePostS3_NoFile(t *testing.T) {
	req := httptest.NewRequest("POST", "/s3?bucket=b&key=k", nil)
	rr := httptest.NewRecorder()
	HandleS3(rr, req, &mockS3{})

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %d want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleS3_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest("DELETE", "/s3?bucket=b&key=k", nil)
	rr := httptest.NewRecorder()
	HandleS3(rr, req, &mockS3{})

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("got %d want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}
