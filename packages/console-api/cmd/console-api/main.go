package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"code-code.internal/console-api/internal/platformclient"
	"code-code.internal/console-api/internal/server"
	"code-code.internal/console-api/internal/telemetry"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	addr := envOrDefault("CONSOLE_API_ADDR", ":8080")
	modelConnectBaseURL := envOrDefault("CONSOLE_API_MODEL_CONNECT_BASE_URL", "http://platform-model-service:8080")
	providerConnectBaseURL := envOrDefault("CONSOLE_API_PROVIDER_CONNECT_BASE_URL", "http://platform-provider-service:8080")
	providerAddr := envOrDefault("CONSOLE_API_PROVIDER_GRPC_ADDR", "platform-provider-service:8081")
	profileAddr := envOrDefault("CONSOLE_API_PROFILE_GRPC_ADDR", "platform-profile-service:8081")
	egressAddr := envOrDefault("CONSOLE_API_EGRESS_GRPC_ADDR", "platform-egress-service.code-code-net.svc.cluster.local:8081")
	authAddr := envOrDefault("CONSOLE_API_AUTH_GRPC_ADDR", "platform-auth-service:8081")
	chatAddr := envOrDefault("CONSOLE_API_CHAT_GRPC_ADDR", "platform-chat-service:8081")
	supportAddr := envOrDefault("CONSOLE_API_SUPPORT_GRPC_ADDR", "platform-support-service:8081")
	prometheusBaseURL := envOrDefault("CONSOLE_API_PROMETHEUS_BASE_URL", "http://prometheus.code-code-observability.svc.cluster.local:9090")

	telemetryShutdown, err := telemetry.Setup(context.Background(), envOrDefault("OTEL_SERVICE_NAME", "console-api"))
	must(err)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetryShutdown(shutdownCtx); err != nil {
			log.Printf("shutdown telemetry failed: %v", err)
		}
	}()

	providerConn, err := grpc.NewClient(
		providerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	must(err)
	defer func() {
		if err := providerConn.Close(); err != nil {
			log.Printf("close provider connection failed: %v", err)
		}
	}()
	profileConn, err := grpc.NewClient(
		profileAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	must(err)
	defer func() {
		if err := profileConn.Close(); err != nil {
			log.Printf("close profile connection failed: %v", err)
		}
	}()
	egressConn, err := grpc.NewClient(
		egressAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	must(err)
	defer func() {
		if err := egressConn.Close(); err != nil {
			log.Printf("close egress connection failed: %v", err)
		}
	}()
	authConn, err := grpc.NewClient(
		authAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	must(err)
	defer func() {
		if err := authConn.Close(); err != nil {
			log.Printf("close auth connection failed: %v", err)
		}
	}()
	chatConn, err := grpc.NewClient(
		chatAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	must(err)
	defer func() {
		if err := chatConn.Close(); err != nil {
			log.Printf("close chat connection failed: %v", err)
		}
	}()
	supportConn, err := grpc.NewClient(
		supportAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	must(err)
	defer func() {
		if err := supportConn.Close(); err != nil {
			log.Printf("close support connection failed: %v", err)
		}
	}()

	platformClient, err := platformclient.New(platformclient.Config{
		SessionConn:  chatConn,
		ChatConn:     chatConn,
		ProviderConn: providerConn,
		ProfileConn:  profileConn,
		EgressConn:   egressConn,
		AuthConn:     authConn,
		SupportConn:  supportConn,
	})
	must(err)

	srv, err := server.New(server.Config{
		Platform:               platformClient,
		PrometheusBaseURL:      prometheusBaseURL,
		ModelConnectBaseURL:    modelConnectBaseURL,
		ProviderConnectBaseURL: providerConnectBaseURL,
	})
	must(err)

	apiServer := &http.Server{
		Addr:              addr,
		Handler:           otelhttp.NewHandler(srv.Handler, "console_api_http"),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down console-api...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := apiServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("console-api shutdown failed: %v", err)
		}
	}()

	log.Printf("console-api listening on %s (model_connect=%s, provider_connect=%s, provider=%s, profile=%s, egress=%s, auth=%s, chat=%s, support=%s)", addr, modelConnectBaseURL, providerConnectBaseURL, providerAddr, profileAddr, egressAddr, authAddr, chatAddr, supportAddr)
	if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		must(err)
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
