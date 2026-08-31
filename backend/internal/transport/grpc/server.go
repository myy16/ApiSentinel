package grpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/valkey"
	agentv1 "github.com/apisentinel/apisentinel/pkg/genproto/proto/agent/v1"
	replayv1 "github.com/apisentinel/apisentinel/pkg/genproto/proto/replay/v1"
	securityv1 "github.com/apisentinel/apisentinel/pkg/genproto/proto/security/v1"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type AgentSession struct {
	AgentID        string
	ProjectID      string
	OrganizationID string
	Hostname       string
	OS             string
	Version        string
	Stream         agentv1.AgentService_ConnectSessionServer
	LastSeen       time.Time
}

type agentScope struct {
	projectID      string
	organizationID string
}

type Server struct {
	agentv1.UnimplementedAgentServiceServer
	queries      *database.Queries
	valkeyClient *valkey.Client
	grpcSrv      *grpc.Server
	sessions     sync.Map // map[string]*AgentSession
	port         int
	jwtSecret    string
}

func NewServer(queries *database.Queries, port int, jwtSecret string, tlsCertFile, tlsKeyFile string, valkeyClient *valkey.Client) *Server {
	s := &Server{
		queries:      queries,
		valkeyClient: valkeyClient,
		port:         port,
		jwtSecret:    jwtSecret,
	}

	var serverOpts []grpc.ServerOption

	// 1. TLS Setup — required in production, optional in development
	if tlsCertFile != "" && tlsKeyFile != "" {
		creds, err := credentials.NewServerTLSFromFile(tlsCertFile, tlsKeyFile)
		if err != nil {
			if !isDevelopment() {
				log.Fatal().Err(err).Msg("FATAL: Failed to load TLS certificates for gRPC in production mode")
			}
			log.Warn().Err(err).Msg("Failed to load TLS certificates for gRPC, falling back to insecure (development only)")
		} else {
			serverOpts = append(serverOpts, grpc.Creds(creds))
			log.Info().Msg("gRPC TLS encryption enabled")
		}
	} else if !isDevelopment() {
		log.Fatal().Msg("FATAL: gRPC TLS is required in production — set TLS_CERT_FILE and TLS_KEY_FILE")
	}

	// 2. Authentication Interceptors
	serverOpts = append(serverOpts,
		grpc.StreamInterceptor(s.streamAuthInterceptor()),
		grpc.UnaryInterceptor(s.unaryAuthInterceptor()),
	)

	grpcServer := grpc.NewServer(serverOpts...)
	agentv1.RegisterAgentServiceServer(grpcServer, s)
	reflection.Register(grpcServer)
	s.grpcSrv = grpcServer

	return s
}

// isDevelopment checks if the current environment is development.
// ALLOW_INSECURE_GRPC is ONLY honored in development mode.
func isDevelopment() bool {
	env := strings.ToLower(os.Getenv("APP_ENV"))
	if env == "" {
		env = strings.ToLower(os.Getenv("NODE_ENV"))
	}
	return env != "production"
}

// allowInsecureGRPC returns true only if both the env var is set AND we are in development.
// In production, setting this env var is explicitly warned and ignored.
func allowInsecureGRPC() bool {
	if os.Getenv("ALLOW_INSECURE_GRPC") == "true" && !isDevelopment() {
		log.Warn().Msg("SECURITY: ALLOW_INSECURE_GRPC is set but IGNORED in production mode")
		return false
	}
	return os.Getenv("ALLOW_INSECURE_GRPC") == "true" && isDevelopment()
}

func (s *Server) streamAuthInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			if allowInsecureGRPC() {
				return handler(srv, ss)
			}
			return status.Errorf(codes.Unauthenticated, "missing gRPC metadata")
		}

		if err := s.validateToken(md); err != nil {
			if allowInsecureGRPC() {
				return handler(srv, ss)
			}
			return err
		}

		return handler(srv, ss)
	}
}

func (s *Server) unaryAuthInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			if allowInsecureGRPC() {
				return handler(ctx, req)
			}
			return nil, status.Errorf(codes.Unauthenticated, "missing gRPC metadata")
		}

		if err := s.validateToken(md); err != nil {
			if allowInsecureGRPC() {
				return handler(ctx, req)
			}
			return nil, err
		}

		return handler(ctx, req)
	}
}

func (s *Server) validateToken(md metadata.MD) error {
	_, err := s.resolveAgentScope(md)
	return err
}

// resolveAgentScope authenticates agent traffic with a non-expired project API key.
// A normal dashboard JWT intentionally cannot create a gRPC agent tunnel.
func (s *Server) resolveAgentScope(md metadata.MD) (agentScope, error) {
	tokens := md.Get("authorization")
	if len(tokens) == 0 {
		tokens = md.Get("x-agent-token")
	}

	if len(tokens) == 0 {
		return agentScope{}, status.Errorf(codes.Unauthenticated, "agent authorization token required")
	}

	rawToken := strings.TrimPrefix(tokens[0], "Bearer ")

	// 1. Development default token
	if isDevelopment() && (rawToken == "apisent_dev_token" || allowInsecureGRPC()) {
		log.Warn().Msg("SECURITY: gRPC auth bypassed using development-only token — do NOT use in production")
		return agentScope{}, nil
	}

	// Project API keys provide both authentication and the tenant scope of an agent session.
	if strings.HasPrefix(rawToken, "apisent_live_") || strings.HasPrefix(rawToken, "apisent_test_") {
		var prefix string
		if strings.HasPrefix(rawToken, "apisent_live_") {
			prefix = "apisent_live_"
		} else {
			prefix = "apisent_test_"
		}
		hash := sha256.Sum256([]byte(rawToken))
		keyHash := hex.EncodeToString(hash[:])
		key, err := s.queries.GetAPIKeyByPrefixAndHash(context.Background(), database.GetAPIKeyByPrefixAndHashParams{
			KeyPrefix: prefix,
			KeyHash:   keyHash,
		})
		if err == nil {
			organizationID, orgErr := s.queries.GetProjectOrganizationID(context.Background(), key.ProjectID)
			if orgErr != nil {
				return agentScope{}, status.Errorf(codes.Unauthenticated, "agent project no longer exists")
			}
			return agentScope{
				projectID:      uuid.UUID(key.ProjectID.Bytes).String(),
				organizationID: uuid.UUID(organizationID.Bytes).String(),
			}, nil
		}
	}

	return agentScope{}, status.Errorf(codes.Unauthenticated, "a valid project API key is required for agent connections")
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port %d: %w", s.port, err)
	}

	log.Info().Int("port", s.port).Msg("ApiSentinel gRPC Server listening (with Token Auth Interceptor)")
	return s.grpcSrv.Serve(lis)
}

func (s *Server) Stop() {
	s.grpcSrv.GracefulStop()
	log.Info().Msg("gRPC Server stopped gracefully")
}

// GetActiveSessions returns a snapshot of all currently connected agent sessions.
// Used by the HTTP agent handler to expose real agent data to the frontend.
func (s *Server) GetActiveSessions() []*AgentSession {
	var sessions []*AgentSession
	s.sessions.Range(func(key, value interface{}) bool {
		if session, ok := value.(*AgentSession); ok {
			sessions = append(sessions, session)
		}
		return true
	})
	return sessions
}

// GetActiveSessionsByOrganization prevents one tenant from seeing another tenant's agents.
func (s *Server) GetActiveSessionsByOrganization(organizationID string) []*AgentSession {
	var sessions []*AgentSession
	s.sessions.Range(func(_, value interface{}) bool {
		if session, ok := value.(*AgentSession); ok && session.OrganizationID == organizationID {
			sessions = append(sessions, session)
		}
		return true
	})
	return sessions
}

// ConnectSession handles the bidirectional streaming connection from Go Agent
func (s *Server) ConnectSession(stream agentv1.AgentService_ConnectSessionServer) error {
	md, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		return status.Errorf(codes.Unauthenticated, "missing agent metadata")
	}
	scope, err := s.resolveAgentScope(md)
	if err != nil {
		return err
	}
	if scope.organizationID == "" && !isDevelopment() {
		return status.Errorf(codes.Unauthenticated, "agent connection requires a project API key")
	}

	var currentAgentID string
	var session *AgentSession

	defer func() {
		if currentAgentID != "" {
			s.sessions.Delete(currentAgentID)
			log.Info().Str("agentId", currentAgentID).Msg("Agent disconnected from gRPC session")
		}
	}()

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Warn().Err(err).Msg("gRPC stream read error")
			return err
		}

		currentAgentID = msg.AgentId
		if session == nil {
			session = &AgentSession{
				AgentID:        currentAgentID,
				ProjectID:      scope.projectID,
				OrganizationID: scope.organizationID,
				Stream:         stream,
				LastSeen:       time.Now(),
			}
			s.sessions.Store(currentAgentID, session)
		}

		switch payload := msg.Payload.(type) {
		case *agentv1.AgentMessage_Heartbeat:
			session.Hostname = payload.Heartbeat.Hostname
			session.OS = payload.Heartbeat.Os
			session.Version = payload.Heartbeat.Version
			session.LastSeen = time.Now()

			// Send Ack back
			stream.Send(&agentv1.CloudMessage{
				Payload: &agentv1.CloudMessage_HeartbeatAck{
					HeartbeatAck: &agentv1.CloudHeartbeatAck{
						Timestamp: time.Now().Unix(),
					},
				},
			})

		case *agentv1.AgentMessage_ScanEvent:
			log.Info().
				Str("agentId", currentAgentID).
				Str("scanType", payload.ScanEvent.ScanType).
				Int("findings", len(payload.ScanEvent.Findings)).
				Msg("Received live scan event from Agent")

			scanAction := "ALLOW"
			for _, f := range payload.ScanEvent.Findings {
				if f.Severity == securityv1.Severity_SEVERITY_CRITICAL || f.Severity == securityv1.Severity_SEVERITY_HIGH {
					scanAction = "BLOCK"
					break
				}
			}

			s.persistScanAndFindings(
				stream.Context(),
				session.ProjectID,
				currentAgentID,
				payload.ScanEvent.Repository,
				"",
				"",
				payload.ScanEvent.ScanType,
				scanAction,
				payload.ScanEvent.Findings,
			)

		case *agentv1.AgentMessage_ReplayResult:
			log.Info().
				Str("jobId", payload.ReplayResult.JobId).
				Int32("status", payload.ReplayResult.ResponseStatus).
				Int64("latencyMs", payload.ReplayResult.LatencyMs).
				Msg("Received local replay execution result from Agent")
		}
	}
}

// SyncScanResults handles batch CLI scans
func (s *Server) SyncScanResults(ctx context.Context, req *agentv1.SyncScanRequest) (*agentv1.SyncScanResponse, error) {
	log.Info().
		Str("agentId", req.AgentId).
		Str("repo", req.Repository).
		Int("findings", len(req.Findings)).
		Msg("Batch scan results synced from CLI")

	// 1. Resolve and enforce project authentication from metadata (#1.3)
	var projectID string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if scope, err := s.resolveAgentScope(md); err == nil {
			projectID = scope.projectID
		}
	}

	if projectID == "" {
		return nil, status.Error(codes.Unauthenticated, "Proje doğrulaması başarısız: Geçersiz veya eksik Agent API anahtarı")
	}

	// 2. Validate commit hash & repository format (#1.4)
	commitHash := strings.TrimSpace(req.CommitHash)
	if len(commitHash) > 64 {
		commitHash = commitHash[:64]
	}
	repoName := strings.TrimSpace(req.Repository)
	if repoName == "" {
		repoName = "workspace"
	}

	action := "ALLOW"
	msg := "Temiz! Kritik bir güvenlik bulgusu tespit edilmedi."

	for _, f := range req.Findings {
		if f.Severity == securityv1.Severity_SEVERITY_CRITICAL || f.Severity == securityv1.Severity_SEVERITY_HIGH {
			action = "BLOCK"
			msg = fmt.Sprintf("Git İşlemi Engellendi: %s (%s)", f.Type, f.Message)
			break
		}
	}

	// 3. Persist scan and findings into PostgreSQL (including 0 findings clean scans) (#1.2, #1.4)
	_, err := s.persistScanAndFindings(
		ctx,
		projectID,
		req.AgentId,
		repoName,
		req.Branch,
		commitHash,
		"CLI_SCAN",
		action,
		req.Findings,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to persist agent scan results")
		return &agentv1.SyncScanResponse{
			Accepted: false,
			Action:   "ERROR",
			Message:  fmt.Sprintf("Veritabanı kayıt hatası: %v", err),
		}, status.Error(codes.Internal, "Tarama sonuçları kaydedilemedi")
	}

	return &agentv1.SyncScanResponse{
		Accepted: true,
		Action:   action,
		Message:  msg,
	}, nil
}

func (s *Server) persistScanAndFindings(
	ctx context.Context,
	projectIDStr string,
	agentID string,
	repository string,
	branch string,
	commitHash string,
	scanType string,
	action string,
	findings []*securityv1.SecurityFinding,
) (string, error) {
	if projectIDStr == "" {
		return "", fmt.Errorf("project ID is required")
	}
	projUUID, err := uuid.Parse(projectIDStr)
	if err != nil {
		return "", fmt.Errorf("invalid project UUID: %w", err)
	}
	pgProjID := pgtype.UUID{Bytes: projUUID, Valid: true}

	// 1. Idempotency Check: if identical scan for this commit already exists, reuse it (#1.4)
	if commitHash != "" && repository != "" {
		existingScan, err := s.queries.GetAgentScanByIdempotencyKey(ctx, database.GetAgentScanByIdempotencyKeyParams{
			ProjectID:  pgProjID,
			Repository: repository,
			CommitHash: commitHash,
			ScanType:   scanType,
		})
		if err == nil {
			log.Info().
				Str("scanId", uuid.UUID(existingScan.ID.Bytes).String()).
				Str("commit", commitHash).
				Msg("Idempotent scan match: returning existing scan record")
			return uuid.UUID(existingScan.ID.Bytes).String(), nil
		}
	}

	// 2. Insert agent_scans record (records even with 0 findings #1.2)
	scan, err := s.queries.CreateAgentScan(ctx, database.CreateAgentScanParams{
		ProjectID:     pgProjID,
		AgentID:       agentID,
		Repository:    repository,
		Branch:        branch,
		CommitHash:    commitHash,
		ScanType:      scanType,
		TotalFindings: int32(len(findings)),
		Action:        action,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to persist agent_scan record")
		return "", fmt.Errorf("failed to persist agent scan: %w", err)
	}

	scanIDStr := uuid.UUID(scan.ID.Bytes).String()

	// 3. Persist individual security findings with precise FilePath and LineNumber (#1.1)
	for _, f := range findings {
		sevStr := protoSeverityToString(f.Severity)
		filePath := f.FilePath
		if filePath == "" {
			filePath = f.FieldPath
		}

		findingRecord, fErr := s.queries.CreateSecurityFinding(ctx, database.CreateSecurityFindingParams{
			ProjectID:      pgProjID,
			ScanID:         scan.ID,
			SourceType:     "AGENT_GIT",
			Category:       f.Category,
			Type:           f.Type,
			Severity:       sevStr,
			Action:         action,
			FieldPath:      pgtype.Text{String: f.FieldPath, Valid: f.FieldPath != ""},
			FilePath:       pgtype.Text{String: filePath, Valid: filePath != ""},
			LineNumber:     pgtype.Int4{Int32: f.LineNumber, Valid: f.LineNumber > 0},
			Repository:     pgtype.Text{String: repository, Valid: repository != ""},
			CommitHash:     pgtype.Text{String: commitHash, Valid: commitHash != ""},
			Message:        f.Message,
			EvidenceMasked: pgtype.Text{String: f.EvidenceMasked, Valid: f.EvidenceMasked != ""},
			Confidence:     pgtype.Float8{Float64: f.Confidence, Valid: true},
		})
		if fErr != nil {
			log.Error().Err(fErr).Msg("Failed to persist agent security_finding")
			continue
		}

		// Publish finding.created SSE event
		if s.valkeyClient != nil {
			eventPayload, _ := json.Marshal(map[string]interface{}{
				"event":      "finding.created",
				"id":         uuid.UUID(findingRecord.ID.Bytes).String(),
				"scanId":     scanIDStr,
				"sourceType": "AGENT_GIT",
				"category":   findingRecord.Category,
				"type":       findingRecord.Type,
				"severity":   findingRecord.Severity,
				"action":     findingRecord.Action,
				"message":    findingRecord.Message,
				"repository": repository,
				"filePath":   filePath,
				"lineNumber": f.LineNumber,
				"createdAt":  time.Now().Format(time.RFC3339),
			})
			s.valkeyClient.PublishEvent(ctx, "channel:events:"+projectIDStr, string(eventPayload))
		}
	}

	// Publish scan.completed SSE event
	if s.valkeyClient != nil {
		scanPayload, _ := json.Marshal(map[string]interface{}{
			"event":         "scan.completed",
			"id":            scanIDStr,
			"agentId":       agentID,
			"repository":    repository,
			"branch":        branch,
			"commitHash":    commitHash,
			"scanType":      scanType,
			"totalFindings": len(findings),
			"action":        action,
			"createdAt":     time.Now().Format(time.RFC3339),
		})
		s.valkeyClient.PublishEvent(ctx, "channel:events:"+projectIDStr, string(scanPayload))
	}

	return scanIDStr, nil
}

func protoSeverityToString(sev securityv1.Severity) string {
	switch sev {
	case securityv1.Severity_SEVERITY_CRITICAL:
		return "CRITICAL"
	case securityv1.Severity_SEVERITY_HIGH:
		return "HIGH"
	case securityv1.Severity_SEVERITY_MEDIUM:
		return "MEDIUM"
	case securityv1.Severity_SEVERITY_LOW:
		return "LOW"
	default:
		return "INFO"
	}
}

// DispatchReplayToAgent sends a Replay command over the active bidirectional stream to the agent
func (s *Server) DispatchReplayToAgent(agentID string, replayReq *replayv1.ReplayRequest) error {
	val, ok := s.sessions.Load(agentID)
	if !ok {
		return fmt.Errorf("agent %s is not currently connected via gRPC session", agentID)
	}

	session := val.(*AgentSession)
	return session.Stream.Send(&agentv1.CloudMessage{
		Payload: &agentv1.CloudMessage_ReplayCommand{
			ReplayCommand: replayReq,
		},
	})
}

// ListActiveAgents returns all currently connected sessions
func (s *Server) ListActiveAgents() []map[string]interface{} {
	var list []map[string]interface{}
	s.sessions.Range(func(key, value interface{}) bool {
		sess := value.(*AgentSession)
		list = append(list, map[string]interface{}{
			"agentId":  sess.AgentID,
			"hostname": sess.Hostname,
			"os":       sess.OS,
			"version":  sess.Version,
			"status":   "ONLINE",
			"lastSeen": sess.LastSeen.Format(time.RFC3339),
		})
		return true
	})
	return list
}
