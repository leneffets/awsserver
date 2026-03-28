package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ecrpkg "github.com/leneffets/awsserver/pkg/ecr"
	s3pkg "github.com/leneffets/awsserver/pkg/s3"
	smpkg "github.com/leneffets/awsserver/pkg/secretsmanager"
	ssmpkg "github.com/leneffets/awsserver/pkg/ssm"
	stspkg "github.com/leneffets/awsserver/pkg/sts"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start).String(),
		)
	})
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		slog.Error("failed to load AWS config", "error", err)
		os.Exit(1)
	}

	ssmSvc := ssm.NewFromConfig(cfg)
	s3Svc := s3.NewFromConfig(cfg)
	ecrSvc := ecr.NewFromConfig(cfg)
	stsSvc := sts.NewFromConfig(cfg)
	smSvc := secretsmanager.NewFromConfig(cfg)

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("/ssm", func(w http.ResponseWriter, r *http.Request) {
		ssmpkg.HandleSSM(w, r, ssmSvc)
	})

	mux.HandleFunc("/s3", func(w http.ResponseWriter, r *http.Request) {
		s3pkg.HandleS3(w, r, s3Svc)
	})

	mux.HandleFunc("/ecr/login", func(w http.ResponseWriter, r *http.Request) {
		ecrpkg.HandleECRLogin(w, r, ecrSvc)
	})

	mux.HandleFunc("/sts", func(w http.ResponseWriter, r *http.Request) {
		stspkg.HandleSTS(w, r, stsSvc)
	})

	mux.HandleFunc("/secrets", func(w http.ResponseWriter, r *http.Request) {
		smpkg.HandleSecrets(w, r, smSvc)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	bind := os.Getenv("BIND_ADDRESS")
	if bind == "" {
		bind = "0.0.0.0"
	}

	addr := bind + ":" + port
	srv := &http.Server{
		Addr:         addr,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}
