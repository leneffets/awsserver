package ssm

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

func GetParameter(ctx context.Context, client *ssm.Client, name string) (*ssm.GetParameterOutput, error) {
	return client.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(true),
	})
}
func PutParameter(ctx context.Context, client *ssm.Client, name, value, typeStr string) (*ssm.PutParameterOutput, error) {
	return client.PutParameter(ctx, &ssm.PutParameterInput{
		Name:  aws.String(name),
		Value: aws.String(value),
		Type:  types.ParameterType(typeStr),
	})
}

func HandleSSM(w http.ResponseWriter, r *http.Request, cfg aws.Config) {
	client := ssm.NewFromConfig(cfg)

	if r.Method == http.MethodGet {
		handleGetSSM(w, r, client)
	} else if r.Method == http.MethodPost {
		HandlePostSSM(w, r, client)
	} else {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
	}
}

func handleGetSSM(w http.ResponseWriter, r *http.Request, client *ssm.Client) {
	id := r.URL.Query().Get("name")
	if id == "" {
		http.Error(w, "Parameter 'name' is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results, err := GetParameter(ctx, client, id)
	if err != nil {
		log.Printf("Error fetching parameter: %v", err)
		http.Error(w, "Error fetching parameter", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(*results.Parameter.Value))
}

func HandlePostSSM(w http.ResponseWriter, r *http.Request, client *ssm.Client) {
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

	_, err := PutParameter(ctx, client, name, value, typeStr)
	if err != nil {
		http.Error(w, "Error putting parameter", http.StatusInternalServerError)
		log.Printf("Error putting parameter: %v", err)
		return
	}

	log.Printf("Parameter %s uploaded successfully", name)
	w.WriteHeader(http.StatusOK)
}
