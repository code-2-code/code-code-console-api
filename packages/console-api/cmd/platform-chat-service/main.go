package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"code-code.internal/console-api/internal/chats"
	"code-code.internal/console-api/internal/platformclient"
	"code-code.internal/console-api/internal/telemetry"
	chatv1 "code-code.internal/go-contract/platform/chat/v1"
	managementv1 "code-code.internal/go-contract/platform/management/v1"
	sessiondomain "code-code.internal/session"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	addr := envOrDefault("PLATFORM_CHAT_SERVICE_GRPC_ADDR", ":8081")
	providerAddr := envOrDefault("PLATFORM_CHAT_SERVICE_PROVIDER_GRPC_ADDR", "platform-provider-service:8081")
	cliRuntimeAddr := envOrDefault("PLATFORM_CHAT_SERVICE_CLI_RUNTIME_GRPC_ADDR", "platform-cli-runtime-service:8081")
	sessionAddr := envOrDefault("PLATFORM_CHAT_SERVICE_AGENT_RUNTIME_GRPC_ADDR", "platform-agent-runtime-service:8081")
	supportAddr := envOrDefault("PLATFORM_CHAT_SERVICE_SUPPORT_GRPC_ADDR", "platform-support-service:8081")
	sessionNamespace := envOrDefault("PLATFORM_CHAT_SERVICE_SESSION_NAMESPACE", "code-code")
	databaseURL := firstEnv("PLATFORM_DATABASE_URL", "PLATFORM_CHAT_SERVICE_DATABASE_URL")

	telemetryShutdown, err := telemetry.Setup(context.Background(), envOrDefault("OTEL_SERVICE_NAME", "platform-chat-service"))
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

	cliRuntimeConn, err := grpc.NewClient(
		cliRuntimeAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	must(err)
	defer func() {
		if err := cliRuntimeConn.Close(); err != nil {
			log.Printf("close cli runtime connection failed: %v", err)
		}
	}()

	sessionConn, err := grpc.NewClient(
		sessionAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	must(err)
	defer func() {
		if err := sessionConn.Close(); err != nil {
			log.Printf("close session connection failed: %v", err)
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
		SessionConn:    sessionConn,
		ProviderConn:   providerConn,
		CLIRuntimeConn: cliRuntimeConn,
		SupportConn:    supportConn,
	})
	must(err)
	sessionClient, err := platformClient.AgentSessionManagementClient()
	must(err)

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	must(err)
	if poolConfig.ConnConfig.RuntimeParams == nil {
		poolConfig.ConnConfig.RuntimeParams = map[string]string{}
	}
	poolConfig.ConnConfig.RuntimeParams["application_name"] = "platform-chat-service"
	chatPool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	must(err)
	defer chatPool.Close()
	chatState, err := chats.NewPostgresState(context.Background(), chatPool, sessiondomain.PostgresRepositoryConfig{
		Namespace: sessionNamespace,
	})
	must(err)

	listener, err := net.Listen("tcp", addr)
	must(err)
	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	managementv1.RegisterAgentSessionManagementServiceServer(grpcServer, chats.NewAgentSessionManagementProxy(sessionClient))
	chatv1.RegisterChatServiceServer(grpcServer, chats.NewGRPCChatServer(chats.NewSessionRuntimeOptionsService(
		platformClient.Providers(),
		platformClient.CLIDefinitions(),
		platformClient.SupportResources(),
		platformClient.CLIRuntimes(),
	), chatState))

	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthv1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(managementv1.AgentSessionManagementService_ServiceDesc.ServiceName, healthv1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(chatv1.ChatService_ServiceDesc.ServiceName, healthv1.HealthCheckResponse_SERVING)
	healthv1.RegisterHealthServer(grpcServer, healthServer)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down platform-chat-service...")
		healthServer.SetServingStatus("", healthv1.HealthCheckResponse_NOT_SERVING)
		healthServer.SetServingStatus(managementv1.AgentSessionManagementService_ServiceDesc.ServiceName, healthv1.HealthCheckResponse_NOT_SERVING)
		healthServer.SetServingStatus(chatv1.ChatService_ServiceDesc.ServiceName, healthv1.HealthCheckResponse_NOT_SERVING)
		grpcServer.GracefulStop()
	}()

	log.Printf("platform-chat-service listening on %s (provider=%s cli_runtime=%s session=%s support=%s)", addr, providerAddr, cliRuntimeAddr, sessionAddr, supportAddr)
	if err := grpcServer.Serve(listener); err != nil {
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

func firstEnv(keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return ""
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
