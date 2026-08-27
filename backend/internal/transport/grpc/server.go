package grpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	agentv1 "github.com/apisentinel/apisentinel/pkg/genproto/proto/agent/v1"
	replayv1 "github.com/apisentinel/apisentinel/pkg/genproto/proto/replay/v1"
	securityv1 "github.com/apisentinel/apisentinel/pkg/genproto/proto/security/v1"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type AgentSession struct {
	AgentID    string
	Hostname   string
	OS         string
	Version    string
	Stream     agentv1.AgentService_ConnectSessionServer
	ReplayChan chan *replayv1.ReplayRequest
	LastSeen   time.Time
}

type Server struct {
	agentv1.UnimplementedAgentServiceServer
	queries   *database.Queries
	grpcSrv   *grpc.Server
	sessions  sync.Map // map[string]*AgentSession
	port      int
	jwtSecret string
}

func NewServer(queries *database.Queries, port int, jwtSecret string, tlsCertFile, tlsKeyFile string) *Server {
	s := &Server{
		queries:   queries,
		port:      port,
		jwtSecret: jwtSecret,
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
		log.Warn().Msg("gRPC running without TLS in production — set TLS_CERT_FILE and TLS_KEY_FILE")
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
	tokens := md.Get("authorization")
	if len(tokens) == 0 {
		tokens = md.Get("x-agent-token")
	}

	if len(tokens) == 0 {
		return status.Errorf(codes.Unauthenticated, "agent authorization token required")
	}

	rawToken := strings.TrimPrefix(tokens[0], "Bearer ")

	// 1. Development default token
	if isDevelopment() && (rawToken == "apisent_dev_token" || allowInsecureGRPC()) {
		log.Warn().Msg("SECURITY: gRPC auth bypassed using development-only token — do NOT use in production")
		return nil
	}

	// 2. Check if token matches predefined agent secret
	if agentKey := os.Getenv("AGENT_SECRET_KEY"); agentKey != "" && rawToken == agentKey {
		return nil
	}

	// 3. Check database API key (apisent_live_... or apisent_test_...)
	if strings.HasPrefix(rawToken, "apisent_live_") || strings.HasPrefix(rawToken, "apisent_test_") {
		var prefix string
		if strings.HasPrefix(rawToken, "apisent_live_") {
			prefix = "apisent_live_"
		} else {
			prefix = "apisent_test_"
		}
		hash := sha256.Sum256([]byte(rawToken))
		keyHash := hex.EncodeToString(hash[:])
		if _, err := s.queries.GetAPIKeyByPrefixAndHash(context.Background(), database.GetAPIKeyByPrefixAndHashParams{
			KeyPrefix: prefix,
			KeyHash:   keyHash,
		}); err == nil {
			return nil
		}
	}

	// 4. Validate JWT token
	token, err := jwt.Parse(rawToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.jwtSecret), nil
	})

	if err == nil && token.Valid {
		return nil
	}

	return status.Errorf(codes.Unauthenticated, "invalid or expired agent authorization token")
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

// ConnectSession handles the bidirectional streaming connection from Go Agent
func (s *Server) ConnectSession(stream agentv1.AgentService_ConnectSessionServer) error {
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
				AgentID:    currentAgentID,
				Stream:     stream,
				ReplayChan: make(chan *replayv1.ReplayRequest, 10),
				LastSeen:   time.Now(),
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

	action := "ALLOW"
	msg := "Clean! No critical security findings detected."

	for _, f := range req.Findings {
		if f.Severity == securityv1.Severity_SEVERITY_CRITICAL || f.Severity == securityv1.Severity_SEVERITY_HIGH {
			action = "BLOCK"
			msg = fmt.Sprintf("Git Push Blocked: %s (%s)", f.Type, f.Message)
			break
		}
	}

	return &agentv1.SyncScanResponse{
		Accepted: true,
		Action:   action,
		Message:  msg,
	}, nil
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
