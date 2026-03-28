package ecr

import (
	"context"
	"encoding/base64"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

// ECRAPI defines the interface for ECR operations used by this package.
type ECRAPI interface {
	GetAuthorizationToken(ctx context.Context, params *ecr.GetAuthorizationTokenInput, optFns ...func(*ecr.Options)) (*ecr.GetAuthorizationTokenOutput, error)
}

func GetECRCredentials(ctx context.Context, svc ECRAPI) (*ecr.GetAuthorizationTokenOutput, error) {
	return svc.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
}

func HandleECRLogin(w http.ResponseWriter, r *http.Request, svc ECRAPI) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	results, err := GetECRCredentials(ctx, svc)
	if err != nil {
		slog.Error("failed to fetch ECR credentials", "error", err)
		http.Error(w, "Error fetching ECR credentials", http.StatusInternalServerError)
		return
	}

	if len(results.AuthorizationData) == 0 {
		http.Error(w, "No authorization data found", http.StatusInternalServerError)
		return
	}

	authData := results.AuthorizationData[0]
	decodedToken, err := base64.StdEncoding.DecodeString(*authData.AuthorizationToken)
	if err != nil {
		slog.Error("failed to decode authorization token", "error", err)
		http.Error(w, "Error decoding authorization token", http.StatusInternalServerError)
		return
	}

	tokenParts := strings.SplitN(string(decodedToken), ":", 2)
	if len(tokenParts) != 2 {
		http.Error(w, "Invalid authorization token format", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(tokenParts[1])); err != nil {
		slog.Error("failed to write response", "error", err)
	}
}
