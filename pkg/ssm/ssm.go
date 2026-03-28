package ssm

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// SSMAPI defines the interface for SSM operations used by this package.
type SSMAPI interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
	PutParameter(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error)
}

func GetParameter(ctx context.Context, svc SSMAPI, name string) (*ssm.GetParameterOutput, error) {
	return svc.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(true),
	})
}

func PutParameter(ctx context.Context, svc SSMAPI, name, value string, paramType types.ParameterType) (*ssm.PutParameterOutput, error) {
	return svc.PutParameter(ctx, &ssm.PutParameterInput{
		Name:  aws.String(name),
		Value: aws.String(value),
		Type:  paramType,
	})
}

func HandleSSM(w http.ResponseWriter, r *http.Request, svc SSMAPI) {
	switch r.Method {
	case http.MethodGet:
		handleGetSSM(w, r, svc)
	case http.MethodPost:
		HandlePostSSM(w, r, svc)
	default:
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
	}
}

func handleGetSSM(w http.ResponseWriter, r *http.Request, svc SSMAPI) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Parameter 'name' is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	results, err := GetParameter(ctx, svc, name)
	if err != nil {
		slog.Error("failed to fetch parameter", "error", err)
		http.Error(w, "Error fetching parameter", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(*results.Parameter.Value))
}

func HandlePostSSM(w http.ResponseWriter, r *http.Request, svc SSMAPI) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		slog.Error("invalid form data", "error", err)
		return
	}

	name := r.FormValue("name")
	value := r.FormValue("value")
	typeStr := r.FormValue("type")

	if name == "" || value == "" || typeStr == "" {
		http.Error(w, "Parameters 'name', 'value', and 'type' are required", http.StatusBadRequest)
		return
	}

	var paramType types.ParameterType
	switch typeStr {
	case "String":
		paramType = types.ParameterTypeString
	case "SecureString":
		paramType = types.ParameterTypeSecureString
	default:
		http.Error(w, "Parameter 'type' must be 'String' or 'SecureString'", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	_, err := PutParameter(ctx, svc, name, value, paramType)
	if err != nil {
		http.Error(w, "Error putting parameter", http.StatusInternalServerError)
		slog.Error("failed to put parameter", "error", err)
		return
	}

	slog.Info("parameter uploaded", "name", name)
	w.WriteHeader(http.StatusOK)
}
