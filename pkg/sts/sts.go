package sts

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// STSAPI defines the interface for STS operations used by this package.
type STSAPI interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

func GetCallerIdentity(ctx context.Context, svc STSAPI) (*sts.GetCallerIdentityOutput, error) {
	return svc.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
}

func HandleSTS(w http.ResponseWriter, r *http.Request, svc STSAPI) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	results, err := GetCallerIdentity(ctx, svc)
	if err != nil {
		slog.Error("failed to fetch caller identity", "error", err)
		http.Error(w, "Error fetching caller identity", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(results); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}
