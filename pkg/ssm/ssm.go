package ssm

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

func GetParameter(ctx context.Context, svc ssmiface.SSMAPI, name *string) (*ssm.GetParameterOutput, error) {
	results, err := svc.GetParameterWithContext(ctx, &ssm.GetParameterInput{
		Name:           name,
		WithDecryption: aws.Bool(true),
	})
	return results, err
}

func PutParameter(ctx context.Context, svc ssmiface.SSMAPI, name *string, value *string, typeStr *string) (*ssm.PutParameterOutput, error) {
	results, err := svc.PutParameterWithContext(ctx, &ssm.PutParameterInput{
		Name:  name,
		Value: value,
		Type:  aws.String(*typeStr),
	})
	return results, err
}

func HandleSSM(w http.ResponseWriter, r *http.Request, sess *session.Session) {
	svc := ssm.New(sess)

	if r.Method == http.MethodGet {
		handleGetSSM(w, r, svc)
	} else if r.Method == http.MethodPost {
		handlePostSSM(w, r, svc)
	} else {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
	}
}

func handleGetSSM(w http.ResponseWriter, r *http.Request, svc ssmiface.SSMAPI) {
	id := r.URL.Query().Get("name")
	if id == "" {
		http.Error(w, "Parameter 'name' is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := GetParameter(ctx, svc, &id)
	if err != nil {
		log.Printf("Error fetching parameter: %v", err)
		http.Error(w, "Error fetching parameter", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(*results.Parameter.Value))
}

func handlePostSSM(w http.ResponseWriter, r *http.Request, svc ssmiface.SSMAPI) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		log.Printf("Invalid form data: %v", err)
		return
	}

	name := r.FormValue("name")
	value := r.FormValue("value")
	typeStr := r.FormValue("type")

	if name == "" || value == "" || typeStr == "" {
		http.Error(w, "Parameters 'name', 'value', and 'type' are required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := PutParameter(ctx, svc, &name, &value, &typeStr)
	if err != nil {
		http.Error(w, "Error putting parameter", http.StatusInternalServerError)
		log.Printf("Error putting parameter: %v", err)
		return
	}

	log.Printf("Parameter %s uploaded successfully", name)
	w.WriteHeader(http.StatusOK)
}
