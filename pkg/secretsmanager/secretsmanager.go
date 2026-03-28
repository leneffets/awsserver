package secretsmanager

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// SecretsManagerAPI defines the interface for Secrets Manager operations.
type SecretsManagerAPI interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

func GetSecret(ctx context.Context, svc SecretsManagerAPI, name string) (*secretsmanager.GetSecretValueOutput, error) {
	return svc.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(name),
	})
}

func HandleSecrets(w http.ResponseWriter, r *http.Request, svc SecretsManagerAPI) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Parameter 'name' is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result, err := GetSecret(ctx, svc, name)
	if err != nil {
		slog.Error("failed to fetch secret", "error", err)
		http.Error(w, "Error fetching secret", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	if result.SecretString != nil {
		w.Write([]byte(*result.SecretString))
	} else if result.SecretBinary != nil {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(result.SecretBinary)
	} else {
		http.Error(w, "Secret has no value", http.StatusInternalServerError)
	}
}
