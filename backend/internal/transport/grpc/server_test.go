package grpc

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/apisentinel/apisentinel/internal/config"
	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/service"
	agentv1 "github.com/apisentinel/apisentinel/pkg/genproto/proto/agent/v1"
	securityv1 "github.com/apisentinel/apisentinel/pkg/genproto/proto/security/v1"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestSyncScanResults_CleanScanAndIdempotency(t *testing.T) {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Skip("Skipping gRPC integration test: PostgreSQL unavailable")
		return
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skip("Skipping gRPC integration test: PostgreSQL ping failed")
		return
	}

	_ = database.RunMigrations(ctx, pool)
	queries := database.New(pool)

	authService := service.NewAuthService(queries, cfg.JWTSecret)
	projectService := service.NewProjectService(queries)
	apiKeyService := service.NewAPIKeyService(queries)
	findingService := service.NewFindingService(queries)

	// 1. Setup User, Org, Project, and API Key
	email := fmt.Sprintf("grpc_test_%d@apisentinel.dev", time.Now().UnixNano())
	authResp, err := authService.Register(ctx, email, "Password123!", "gRPC Test Org")
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	proj, err := projectService.CreateProject(ctx, authResp.Organization.ID, "gRPC Sentinel Project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	apiKeyResp, err := apiKeyService.GenerateAPIKey(ctx, proj.ID, "CI Scanner Key", authResp.User.ID, false, nil)
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	// 2. Start local in-memory gRPC server
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer lis.Close()

	server := NewServer(queries, lis.Addr().(*net.TCPAddr).Port, cfg.JWTSecret, "", "", nil)
	go func() {
		_ = server.grpcSrv.Serve(lis)
	}()
	defer server.grpcSrv.Stop()

	// 3. Connect gRPC Client
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial gRPC: %v", err)
	}
	defer conn.Close()

	client := agentv1.NewAgentServiceClient(conn)

	// Case A: Unauthenticated request should fail with codes.Unauthenticated (#1.3)
	_, err = client.SyncScanResults(ctx, &agentv1.SyncScanRequest{
		AgentId:    "test-agent",
		Repository: "apisentinel",
		CommitHash: "c0ffee1234567890123456789012345678901234",
	})
	if err == nil {
		t.Fatal("Expected error on unauthenticated SyncScanResults, got nil")
	}
	if st, ok := status.FromError(err); !ok || st.Code() != codes.Unauthenticated {
		t.Fatalf("Expected codes.Unauthenticated, got: %v", err)
	}

	// Context with valid API Key header
	authCtx := metadata.AppendToOutgoingContext(ctx, "x-agent-token", apiKeyResp.SecretKey)

	// Case B: Clean Scan (0 findings) should be recorded successfully (#1.2)
	commitHashClean := "a1b2c3d4e5f60718293a4b5c6d7e8f9012345678"
	cleanResp, err := client.SyncScanResults(authCtx, &agentv1.SyncScanRequest{
		AgentId:    "test-agent",
		Repository: "apisentinel-clean",
		Branch:     "main",
		CommitHash: commitHashClean,
		Findings:   []*securityv1.SecurityFinding{},
	})
	if err != nil {
		t.Fatalf("Failed to sync clean scan: %v", err)
	}
	if !cleanResp.Accepted || cleanResp.Action != "ALLOW" {
		t.Fatalf("Expected clean scan accepted ALLOW, got: %+v", cleanResp)
	}

	// Verify clean scan is recorded in DB
	scans, err := findingService.ListAgentScans(ctx, proj.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list agent scans: %v", err)
	}
	foundClean := false
	for _, sc := range scans {
		if sc.CommitHash == commitHashClean && sc.TotalFindings == 0 {
			foundClean = true
			break
		}
	}
	if !foundClean {
		t.Fatalf("Clean scan with 0 findings was not persisted in agent_scans table")
	}

	// Case C: Scan with findings (including FilePath and LineNumber #1.1)
	commitHashThreat := "beef00112233445566778899aabbccddeeff0011"
	threatResp, err := client.SyncScanResults(authCtx, &agentv1.SyncScanRequest{
		AgentId:    "test-agent",
		Repository: "apisentinel-threat",
		Branch:     "feat/payments",
		CommitHash: commitHashThreat,
		Findings: []*securityv1.SecurityFinding{
			{
				Category:       "SECRET",
				Type:           "OPENAI_API_KEY",
				Severity:       securityv1.Severity_SEVERITY_CRITICAL,
				FieldPath:      "backend/config/keys.go:42",
				FilePath:       "backend/config/keys.go",
				LineNumber:     42,
				Message:        "Leaked OpenAI Secret Key in code",
				EvidenceMasked: "sk-proj-****1234",
				Confidence:     1.0,
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to sync threat scan: %v", err)
	}
	if !threatResp.Accepted || threatResp.Action != "BLOCK" {
		t.Fatalf("Expected threat scan to BLOCK, got: %+v", threatResp)
	}

	// Verify findings in DB with FilePath and LineNumber
	findingsList, err := findingService.ListByProject(ctx, proj.ID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list findings: %v", err)
	}
	var foundThreatFinding bool
	for _, f := range findingsList {
		if f.Type == "OPENAI_API_KEY" && f.FilePath != nil && *f.FilePath == "backend/config/keys.go" && f.LineNumber != nil && *f.LineNumber == 42 {
			foundThreatFinding = true
			break
		}
	}
	if !foundThreatFinding {
		t.Fatalf("Threat finding was not saved with exact FilePath and LineNumber: %+v", findingsList)
	}

	// Case D: Idempotent Scan (#1.4) - sending same commit again must not duplicate
	threatResp2, err := client.SyncScanResults(authCtx, &agentv1.SyncScanRequest{
		AgentId:    "test-agent",
		Repository: "apisentinel-threat",
		Branch:     "feat/payments",
		CommitHash: commitHashThreat,
		Findings: []*securityv1.SecurityFinding{
			{
				Category:   "SECRET",
				Type:       "OPENAI_API_KEY",
				Severity:   securityv1.Severity_SEVERITY_CRITICAL,
				FilePath:   "backend/config/keys.go",
				LineNumber: 42,
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed second sync of same commit: %v", err)
	}
	if !threatResp2.Accepted {
		t.Fatalf("Expected second sync to be accepted idempotently")
	}

	// Count findings for this project: should still be exactly 1
	stats, err := findingService.GetStats(ctx, proj.ID)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.TotalCount != 1 {
		t.Fatalf("Expected total findings count to be 1 due to idempotency, got %d", stats.TotalCount)
	}
}
