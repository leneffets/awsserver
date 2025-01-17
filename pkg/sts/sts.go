package sts

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

func GetCallerIdentity(ctx context.Context, sess *session.Session) (*sts.GetCallerIdentityOutput, error) {
	svc := sts.New(sess)
	return svc.GetCallerIdentityWithContext(ctx, &sts.GetCallerIdentityInput{})
}

func HandleSTS(w http.ResponseWriter, r *http.Request, sess *session.Session) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if r.Method != http.MethodGet {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	results, err := GetCallerIdentity(ctx, sess)
	if err != nil {
		log.Printf("Error fetching caller identity: %v", err)
		http.Error(w, "Error fetching caller identity", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(results); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		log.Printf("Error encoding response: %v", err)
	}
}
