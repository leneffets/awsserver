package main

import (
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/leneffets/awsserver/pkg/ecr"
	"github.com/leneffets/awsserver/pkg/s3"
	"github.com/leneffets/awsserver/pkg/ssm"
	"github.com/leneffets/awsserver/pkg/sts"
)

func main() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	http.HandleFunc("/ssm", func(w http.ResponseWriter, r *http.Request) {
		ssm.HandleSSM(w, r, sess)
	})

	http.HandleFunc("/s3", func(w http.ResponseWriter, r *http.Request) {
		s3.HandleS3(w, r, sess)
	})

	http.HandleFunc("/ecr/login", func(w http.ResponseWriter, r *http.Request) {
		ecr.HandleECRLogin(w, r, sess)
	})

	http.HandleFunc("/sts", func(w http.ResponseWriter, r *http.Request) {
		sts.HandleSTS(w, r, sess)
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
