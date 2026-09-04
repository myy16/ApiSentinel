package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/apisentinel/apisentinel/internal/config"
	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/security/envelope"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSecretEncryption_AlertWebhookURL(t *testing.T) {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Skip("Skipping secret encryption test: PostgreSQL unavailable")
		return
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skip("Skipping secret encryption test: PostgreSQL ping failed")
		return
	}

	_ = database.RunMigrations(ctx, pool)
	queries := database.New(pool)

	encryptionKey := "super-secure-test-encryption-key-32b"
	authService := NewAuthService(queries, cfg.JWTSecret)
	projectService := NewProjectService(queries)
	alertService := NewAlertService(queries, nil, encryptionKey)

	// Setup Project
	email := fmt.Sprintf("alert_enc_%d@apisentinel.dev", time.Now().UnixNano())
	authResp, err := authService.Register(ctx, email, "Password123!", "Alert Enc Org")
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}
	proj, err := projectService.CreateProject(ctx, authResp.Organization.ID, "Alert Enc Project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// 1. Create Alert Channel with real Discord webhook URL
	rawDiscordURL := "https://discord.com/api/webhooks/1234567890/token_secret_abcdef123456"
	channel, err := alertService.CreateChannel(ctx, CreateAlertChannelInput{
		ProjectID:   proj.ID,
		Name:        "Discord Alerts",
		ChannelType: "DISCORD",
		WebhookURL:  rawDiscordURL,
		MinSeverity: "HIGH",
	})
	if err != nil {
		t.Fatalf("Failed to create alert channel: %v", err)
	}

	// 2. Verify API response masks the secret URL (#3.2)
	if strings.Contains(channel.WebhookUrl, "token_secret_abcdef123456") {
		t.Errorf("API response must NOT expose plaintext webhook secret: %s", channel.WebhookUrl)
	}
	if !strings.Contains(channel.WebhookUrl, "****") {
		t.Errorf("Expected masked URL in response, got: %s", channel.WebhookUrl)
	}

	// 3. Verify Database stores AES-256-GCM ciphertext, NOT the plaintext URL (#3.1)
	chUUID, _ := uuid.Parse(uuid.UUID(channel.ID.Bytes).String())
	rawDBRecord, err := queries.GetAlertChannelByID(ctx, pgtype.UUID{Bytes: chUUID, Valid: true})
	if err != nil {
		t.Fatalf("Failed to get DB record: %v", err)
	}
	if rawDBRecord.WebhookUrl == rawDiscordURL {
		t.Fatalf("CRITICAL SECURITY FLAW: Webhook URL is stored in plaintext in database!")
	}

	// 4. Verify ListChannels also returns masked URLs
	channels, err := alertService.ListChannels(ctx, proj.ID)
	if err != nil || len(channels) == 0 {
		t.Fatalf("ListChannels failed: %v", err)
	}
	if strings.Contains(channels[0].WebhookUrl, "token_secret_abcdef123456") {
		t.Errorf("ListChannels must NOT expose plaintext webhook secret: %s", channels[0].WebhookUrl)
	}

	// 5. Verify resolveChannelURL decrypts correctly for internal dispatch
	decryptedURL := alertService.resolveChannelURL(rawDBRecord)
	if decryptedURL != rawDiscordURL {
		t.Errorf("Expected decrypted URL to match original %s, got: %s", rawDiscordURL, decryptedURL)
	}
}

func TestSecretEncryption_ForwardingCustomHeaders(t *testing.T) {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Skip("Skipping secret encryption test: PostgreSQL unavailable")
		return
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skip("Skipping secret encryption test: PostgreSQL ping failed")
		return
	}

	_ = database.RunMigrations(ctx, pool)
	queries := database.New(pool)

	encryptionKey := "super-secure-test-encryption-key-32b"
	authService := NewAuthService(queries, cfg.JWTSecret)
	projectService := NewProjectService(queries)
	endpointService := NewEndpointService(queries)
	fwdService := NewForwardingService(queries, nil, encryptionKey)

	// Setup Endpoint
	email := fmt.Sprintf("fwd_enc_%d@apisentinel.dev", time.Now().UnixNano())
	authResp, err := authService.Register(ctx, email, "Password123!", "Fwd Enc Org")
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}
	proj, err := projectService.CreateProject(ctx, authResp.Organization.ID, "Fwd Enc Project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	ep, err := endpointService.CreateEndpoint(ctx, CreateEndpointInput{
		ProjectID: proj.ID,
		Name:      "Payment Ingestion",
		Slug:      fmt.Sprintf("pay-%d", time.Now().UnixNano()),
		Mode:      "DEVELOPMENT",
	})
	if err != nil {
		t.Fatalf("Failed to create endpoint: %v", err)
	}

	// Mock Upstream Server to capture incoming decrypted headers
	var capturedAuthHeader string
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuthHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstreamServer.Close()

	// 1. Save Forwarding Config with secret token
	rawAuthHeader := "Bearer secret_jwt_upstream_token_999"
	_, err = fwdService.SaveConfig(ctx, SaveForwardingConfigInput{
		EndpointID:    ep.ID,
		TargetURL:     upstreamServer.URL,
		MaxRetries:    3,
		TimeoutMs:     5000,
		CustomHeaders: map[string]string{"Authorization": rawAuthHeader},
		IsEnabled:     true,
	})
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// 2. Verify Database stores encrypted custom_headers, NOT plaintext JSON (#3.1)
	epUUID, _ := uuid.Parse(ep.ID)
	rawDBRecord, err := queries.GetForwardingConfigByEndpoint(ctx, pgtype.UUID{Bytes: epUUID, Valid: true})
	if err != nil {
		t.Fatalf("Failed to query DB: %v", err)
	}
	if strings.Contains(string(rawDBRecord.CustomHeaders), "secret_jwt_upstream_token_999") {
		t.Fatalf("CRITICAL SECURITY FLAW: custom_headers stored in plaintext in database!")
	}

	// 3. Verify GetConfig returns masked headers (#3.2)
	apiConfig, err := fwdService.GetConfig(ctx, ep.ID)
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	var maskedHeaders map[string]string
	_ = json.Unmarshal(apiConfig.CustomHeaders, &maskedHeaders)
	if maskedHeaders["Authorization"] == rawAuthHeader {
		t.Errorf("GetConfig API response must NOT expose raw token value")
	}
	if !strings.Contains(maskedHeaders["Authorization"], "****") {
		t.Errorf("Expected masked Authorization header, got: %s", maskedHeaders["Authorization"])
	}

	// 4. Verify Dispatch sends real decrypted Authorization header to upstream
	capReq, err := queries.CreateCapturedRequest(ctx, database.CreateCapturedRequestParams{
		EndpointID:       pgtype.UUID{Bytes: epUUID, Valid: true},
		RequestID:        fmt.Sprintf("req_enc_%d", time.Now().UnixNano()),
		HttpMethod:       "POST",
		Headers:          []byte(`{}`),
		QueryParams:      []byte(`{}`),
		RawBody:          pgtype.Text{String: `{"event":"paid"}`, Valid: true},
		MaskedBody:       pgtype.Text{String: `{"event":"paid"}`, Valid: true},
		ParsedJson:       []byte(`{}`),
		ResponseStatus:   pgtype.Int4{Int32: 200, Valid: true},
		ProcessingStatus: "PENDING",
	})
	if err != nil {
		t.Fatalf("Failed to create captured request: %v", err)
	}

	capReqID := uuid.UUID(capReq.ID.Bytes).String()
	fwdService.ForwardCleanWebhook(ctx, ep.ID, capReqID, "POST", map[string]string{}, []byte(`{"event":"paid"}`))

	records, err := fwdService.ListDLQ(ctx, ep.ID)
	if err == nil && len(records) > 0 {
		fwdService.executeOutboxJob(ctx, records[0], "POST", map[string]string{}, []byte(`{"event":"paid"}`))
	}

	if capturedAuthHeader != rawAuthHeader {
		t.Errorf("Expected upstream to receive decrypted header %s, got: %s", rawAuthHeader, capturedAuthHeader)
	}
}

func TestEnvelopeMasking_Helpers(t *testing.T) {
	// Discord
	discord := envelope.MaskWebhookURL("https://discord.com/api/webhooks/12345/abcdefg-secret-token")
	if discord != "https://discord.com/api/webhooks/12345/****" {
		t.Errorf("Unexpected Discord mask: %s", discord)
	}

	// Slack
	slack := envelope.MaskWebhookURL("https://hooks.slack.com/services/T001/B002/secret_key_123")
	if slack != "https://hooks.slack.com/services/T001/B002/****" {
		t.Errorf("Unexpected Slack mask: %s", slack)
	}

	// Telegram
	telegram := envelope.MaskWebhookURL("https://api.telegram.org/bot123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11/sendMessage")
	if telegram != "https://api.telegram.org/bot****/sendMessage" {
		t.Errorf("Unexpected Telegram mask: %s", telegram)
	}

	// Headers
	bearer := envelope.MaskHeaderValue("Authorization", "Bearer sk-proj-1234567890abcdef")
	if !strings.HasPrefix(bearer, "Bearer sk-") || !strings.HasSuffix(bearer, "********") {
		t.Errorf("Unexpected Bearer mask: %s", bearer)
	}

	apiKey := envelope.MaskHeaderValue("X-Api-Key", "my-secret-key-123")
	if !strings.HasSuffix(apiKey, "********") {
		t.Errorf("Unexpected API Key mask: %s", apiKey)
	}
}

func TestSecretEncryption_TestSuiteCustomHeaders(t *testing.T) {
	cfg := config.Load()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Skip("Skipping test: PostgreSQL unavailable")
		return
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skip("Skipping test: PostgreSQL ping failed")
		return
	}

	_ = database.RunMigrations(ctx, pool)
	queries := database.New(pool)

	encryptionKey := "test-encryption-key-for-suites-32"
	authService := NewAuthService(queries, cfg.JWTSecret)
	projectService := NewProjectService(queries)
	replayService := NewReplayService(queries, encryptionKey)
	testSuiteService := NewTestSuiteService(queries, replayService, encryptionKey)

	// Setup Project
	email := fmt.Sprintf("suite_enc_%d@apisentinel.dev", time.Now().UnixNano())
	authResp, err := authService.Register(ctx, email, "Password123!", "Suite Enc Org")
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}
	proj, err := projectService.CreateProject(ctx, authResp.Organization.ID, "Suite Enc Proj")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Create endpoint & mock captured request
	endpointService := NewEndpointService(queries)
	ep, err := endpointService.CreateEndpoint(ctx, CreateEndpointInput{
		ProjectID: proj.ID,
		Name:      "Suite Webhook",
		Slug:      fmt.Sprintf("suite-%d", time.Now().UnixNano()),
		Mode:      "DEVELOPMENT",
	})
	if err != nil {
		t.Fatalf("Failed to create endpoint: %v", err)
	}

	epUUID, _ := uuid.Parse(ep.ID)
	capReq, err := queries.CreateCapturedRequest(ctx, database.CreateCapturedRequestParams{
		EndpointID:       pgtype.UUID{Bytes: epUUID, Valid: true},
		RequestID:        fmt.Sprintf("req_suite_%d", time.Now().UnixNano()),
		HttpMethod:       "POST",
		Headers:          []byte(`{}`),
		QueryParams:      []byte(`{}`),
		RawBody:          pgtype.Text{String: `{"test":true}`, Valid: true},
		MaskedBody:       pgtype.Text{String: `{"test":true}`, Valid: true},
		ParsedJson:       []byte(`{}`),
		ResponseStatus:   pgtype.Int4{Int32: 200, Valid: true},
		ProcessingStatus: "PENDING",
	})
	if err != nil {
		t.Fatalf("Failed to create captured request: %v", err)
	}

	reqIDStr := uuid.UUID(capReq.ID.Bytes).String()
	rawSecretHeader := "Bearer sk-secret-suite-token-12345"

	// 1. Create Test Suite with sensitive custom header
	suite, err := testSuiteService.CreateSuite(ctx, CreateTestSuiteParams{
		ProjectID:   proj.ID,
		Name:        "Encrypted Header Suite",
		Description: "Testing header encryption in suite",
		RequestIDs:  []string{reqIDStr},
		CustomHeaders: map[string]string{
			"Authorization": rawSecretHeader,
			"X-Custom-Env":  "Staging",
		},
		TargetEnvironment: "STAGING",
		TargetURL:         "https://example.com/webhook",
	})
	if err != nil {
		t.Fatalf("CreateSuite failed: %v", err)
	}

	// 2. Returned suite should have MASKED headers (not plaintext, not encrypted envelope)
	var returnedHeaders map[string]string
	_ = json.Unmarshal(suite.CustomHeaders, &returnedHeaders)
	if returnedHeaders["Authorization"] == rawSecretHeader {
		t.Fatalf("Custom header Authorization was not masked in CreateSuite response!")
	}
	if !strings.HasPrefix(returnedHeaders["Authorization"], "Bearer sk-") || !strings.HasSuffix(returnedHeaders["Authorization"], "********") {
		t.Fatalf("Custom header Authorization mask format incorrect: %s", returnedHeaders["Authorization"])
	}

	// 3. Database record should store ENCRYPTED headers
	suiteUUID := uuid.UUID(suite.ID.Bytes)
	dbRecord, err := queries.GetReplayTestSuiteByID(ctx, pgtype.UUID{Bytes: suiteUUID, Valid: true})
	if err != nil {
		t.Fatalf("Failed to fetch suite from DB: %v", err)
	}
	if strings.Contains(string(dbRecord.CustomHeaders), rawSecretHeader) {
		t.Fatalf("PostgreSQL contains plaintext secret in replay_test_suites.custom_headers!")
	}

	// 4. Decrypt method should recover original secret
	resolved := testSuiteService.resolveCustomHeaders(dbRecord.CustomHeaders)
	if resolved["Authorization"] != rawSecretHeader {
		t.Fatalf("resolveCustomHeaders failed: expected %s, got %s", rawSecretHeader, resolved["Authorization"])
	}
}
