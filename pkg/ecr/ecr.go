package ecr

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
)

func GetECRCredentials(ctx context.Context, sess *session.Session) (*ecr.GetAuthorizationTokenOutput, error) {
	svc := ecr.New(sess)
	return svc.GetAuthorizationTokenWithContext(ctx, &ecr.GetAuthorizationTokenInput{})
}

func HandleECRLogin(w http.ResponseWriter, r *http.Request, sess *session.Session) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if r.Method != http.MethodGet {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	results, err := GetECRCredentials(ctx, sess)
	if err != nil {
		log.Printf("Error fetching ECR credentials: %v", err)
		http.Error(w, "Error fetching ECR credentials", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(results); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
	}
}
