package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/leneffets/awsserver/pkg/ecr"
	"github.com/leneffets/awsserver/pkg/s3"
	"github.com/leneffets/awsserver/pkg/ssm"
	"github.com/leneffets/awsserver/pkg/sts"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("unable to load AWS SDK config, %v", err)
	}

	http.HandleFunc("/ssm", func(w http.ResponseWriter, r *http.Request) {
		ssm.HandleSSM(w, r, cfg)
	})

	http.HandleFunc("/s3", func(w http.ResponseWriter, r *http.Request) {
		s3.HandleS3(w, r, cfg)
	})

	http.HandleFunc("/ecr/login", func(w http.ResponseWriter, r *http.Request) {
		ecr.HandleECRLogin(w, r, cfg)
	})

	http.HandleFunc("/sts", func(w http.ResponseWriter, r *http.Request) {
		sts.HandleSTS(w, r, cfg)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server running on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
