package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/apisentinel/apisentinel/internal/config"
	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestForwardingOutbox_StateMachineAndRecovery(t *testing.T) {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Skip("Skipping forwarding outbox test: PostgreSQL unavailable")
		return
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skip("Skipping forwarding outbox test: PostgreSQL ping failed")
		return
	}

	_ = database.RunMigrations(ctx, pool)
	queries := database.New(pool)

	authService := NewAuthService(queries, cfg.JWTSecret)
	projectService := NewProjectService(queries)
	endpointService := NewEndpointService(queries)
	fwdService := NewForwardingService(queries, nil)

	// 1. Setup Org, Project, Endpoint, Request
	email := fmt.Sprintf("outbox_test_%d@apisentinel.dev", time.Now().UnixNano())
	authResp, err := authService.Register(ctx, email, "Password123!", "Outbox Test Org")
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}

	proj, err := projectService.CreateProject(ctx, authResp.Organization.ID, "Outbox Project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	ep, err := endpointService.CreateEndpoint(ctx, proj.ID, "Payment Hook", fmt.Sprintf("payment-hook-%d", time.Now().UnixNano()), "DEVELOPMENT", nil)
	if err != nil {
		t.Fatalf("Failed to create endpoint: %v", err)
	}

	epUUID, _ := uuid.Parse(ep.ID)
	capReq, err := queries.CreateCapturedRequest(ctx, database.CreateCapturedRequestParams{
		EndpointID:       pgtype.UUID{Bytes: epUUID, Valid: true},
		RequestID:        fmt.Sprintf("req_test_%d", time.Now().UnixNano()),
		HttpMethod:       "POST",
		Headers:          []byte(`{"Content-Type":"application/json"}`),
		QueryParams:      []byte(`{}`),
		RawBody:          pgtype.Text{String: `{"user":"john"}`, Valid: true},
		MaskedBody:       pgtype.Text{String: `{"user":"john"}`, Valid: true},
		ParsedJson:       []byte(`{"user":"john"}`),
		ResponseStatus:   pgtype.Int4{Int32: 200, Valid: true},
		ProcessingStatus: "PENDING",
	})
	if err != nil {
		t.Fatalf("Failed to capture request: %v", err)
	}
	capReqID := uuid.UUID(capReq.ID.Bytes).String()

	// 2. Setup mock upstream server that fails 500
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failServer.Close()

	// 3. Save forwarding config pointing to failServer with max_retries = 2
	_, err = fwdService.SaveConfig(ctx, SaveForwardingConfigInput{
		EndpointID: ep.ID,
		TargetURL:  failServer.URL,
		MaxRetries: 2,
		TimeoutMs:  1000,
		IsEnabled:  true,
	})
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// 4. Trigger ForwardCleanWebhook (sync execution for test)
	fwdService.ForwardCleanWebhook(ctx, ep.ID, capReqID, "POST", map[string]string{"Content-Type": "application/json"}, []byte(`{"user":"john"}`))

	// Verify job created in outbox
	records, err := fwdService.ListDLQ(ctx, ep.ID)
	if err != nil || len(records) == 0 {
		t.Fatalf("Expected outbox/DLQ record to be created, got error: %v", err)
	}

	job := records[0]
	if job.Status != "PENDING" && job.Status != "RETRY_WAIT" && job.Status != "DLQ" {
		t.Errorf("Unexpected outbox status: %s", job.Status)
	}

	// 5. Test Crash Recovery: Simulate job stuck in PROCESSING with an old locked_at timestamp
	oldLockedAt := time.Now().Add(-5 * time.Minute)
	_, err = pool.Exec(ctx, "UPDATE forwarding_dlq SET status = 'PROCESSING', locked_at = $1, locked_by = 'crashed-worker' WHERE id = $2", oldLockedAt, job.ID)
	if err != nil {
		t.Fatalf("Failed to simulate crashed worker lock: %v", err)
	}

	// Run recovery query
	err = queries.RecoverStaleOutboxJobs(ctx)
	if err != nil {
		t.Fatalf("RecoverStaleOutboxJobs failed: %v", err)
	}

	// Verify job is back in RETRY_WAIT and unlocked
	recovered, err := queries.GetDLQRecordByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("Failed to fetch recovered job: %v", err)
	}
	if recovered.Status != "RETRY_WAIT" || recovered.LockedAt.Valid {
		t.Errorf("Expected job status RETRY_WAIT and unlocked, got status=%s, lockedAt=%v", recovered.Status, recovered.LockedAt)
	}

	// 6. Test Atomic Claim with SKIP LOCKED
	claimed, err := queries.ClaimPendingOutboxJobs(ctx, database.ClaimPendingOutboxJobsParams{
		LockedBy: pgtype.Text{String: "worker-active-1", Valid: true},
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("ClaimPendingOutboxJobs failed: %v", err)
	}
	if len(claimed) == 0 {
		t.Fatalf("Expected to claim recovered outbox job")
	}
	if claimed[0].Status != "PROCESSING" || claimed[0].LockedBy.String != "worker-active-1" {
		t.Errorf("Expected job to be locked by worker-active-1 in PROCESSING state, got: %+v", claimed[0])
	}
}
