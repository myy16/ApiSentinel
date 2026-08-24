package grpc

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/apisentinel/apisentinel/internal/database"
	agentv1 "github.com/apisentinel/apisentinel/pkg/genproto/proto/agent/v1"
	replayv1 "github.com/apisentinel/apisentinel/pkg/genproto/proto/replay/v1"
	securityv1 "github.com/apisentinel/apisentinel/pkg/genproto/proto/security/v1"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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
	queries  *database.Queries
	grpcSrv  *grpc.Server
	sessions sync.Map // map[string]*AgentSession
	port     int
}

func NewServer(queries *database.Queries, port int) *Server {
	s := &Server{
		queries: queries,
		port:    port,
	}

	grpcServer := grpc.NewServer()
	agentv1.RegisterAgentServiceServer(grpcServer, s)
	reflection.Register(grpcServer)
	s.grpcSrv = grpcServer

	return s
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on gRPC port %d: %w", s.port, err)
	}

	log.Info().Int("port", s.port).Msg("ApiSentinel gRPC Server listening")
	return s.grpcSrv.Serve(lis)
}

func (s *Server) Stop() {
	s.grpcSrv.GracefulStop()
	log.Info().Msg("gRPC Server stopped gracefully")
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
