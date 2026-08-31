package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"

	agentv1 "github.com/apisentinel/apisentinel/pkg/genproto/proto/agent/v1"
	replayv1 "github.com/apisentinel/apisentinel/pkg/genproto/proto/replay/v1"
	securityv1 "github.com/apisentinel/apisentinel/pkg/genproto/proto/security/v1"
	"github.com/fatih/color"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type AgentClient struct {
	serverAddr string
	agentID    string
	token      string
	httpClient *http.Client
}

func NewAgentClient(serverAddr, agentID, token string) *AgentClient {
	if agentID == "" {
		host, _ := os.Hostname()
		agentID = fmt.Sprintf("agent_%s_%d", host, time.Now().Unix())
	}
	return &AgentClient{
		serverAddr: serverAddr,
		agentID:    agentID,
		token:      token,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Connect establishes the bi-directional gRPC streaming session
func (c *AgentClient) Connect(ctx context.Context) error {
	color.Cyan("🔌 Connecting to ApiSentinel Cloud gRPC at %s...", c.serverAddr)

	transportCredentials := credentials.TransportCredentials(insecure.NewCredentials())
	if os.Getenv("APISENTINEL_GRPC_TLS") == "true" {
		transportCredentials = credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: os.Getenv("APISENTINEL_GRPC_SERVER_NAME"),
		})
	}

	conn, err := grpc.DialContext(ctx, c.serverAddr, grpc.WithTransportCredentials(transportCredentials), grpc.WithBlock())
	if err != nil {
		return fmt.Errorf("failed to connect to gRPC server: %w", err)
	}
	defer conn.Close()

	client := agentv1.NewAgentServiceClient(conn)

	// Attach authentication metadata
	streamCtx := ctx
	if c.token != "" {
		streamCtx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+c.token, "x-agent-token", c.token)
	}

	stream, err := client.ConnectSession(streamCtx)
	if err != nil {
		return fmt.Errorf("failed to open authenticated session stream: %w", err)
	}

	hostname, _ := os.Hostname()
	color.Green("✅ Authenticated & Connected to ApiSentinel Cloud! (Agent ID: %s)", c.agentID)
	color.White("   Listening for real-time Replay commands and policy updates...")

	// 1. Heartbeat loop (every 5 seconds)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				err := stream.Send(&agentv1.AgentMessage{
					AgentId: c.agentID,
					Payload: &agentv1.AgentMessage_Heartbeat{
						Heartbeat: &agentv1.AgentHeartbeat{
							Hostname:  hostname,
							Os:        runtime.GOOS,
							Version:   "0.1.0",
							Timestamp: time.Now().Unix(),
						},
					},
				})
				if err != nil {
					color.Yellow("⚠️  Heartbeat send error: %v", err)
					return
				}
			}
		}
	}()

	// 2. Message receiving loop
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			color.Yellow("ℹ️  Server closed connection")
			return nil
		}
		if err != nil {
			return fmt.Errorf("gRPC stream receive error: %w", err)
		}

		switch payload := msg.Payload.(type) {
		case *agentv1.CloudMessage_HeartbeatAck:
			// Heartbeat acknowledged

		case *agentv1.CloudMessage_ReplayCommand:
			color.Cyan("🔄 Received Local Replay Request for Job: %s", payload.ReplayCommand.JobId)
			c.executeLocalReplay(stream, payload.ReplayCommand)

		case *agentv1.CloudMessage_ConfigUpdate:
			color.Green("⚙️  Received Configuration Update from Cloud")
		}
	}
}

func (c *AgentClient) executeLocalReplay(stream agentv1.AgentService_ConnectSessionClient, cmd *replayv1.ReplayRequest) {
	startTime := time.Now()

	req, err := http.NewRequest(cmd.HttpMethod, cmd.TargetUrl, bytes.NewBuffer(cmd.Body))
	if err != nil {
		stream.Send(&agentv1.AgentMessage{
			AgentId: c.agentID,
			Payload: &agentv1.AgentMessage_ReplayResult{
				ReplayResult: &replayv1.ReplayResponse{
					JobId:        cmd.JobId,
					ErrorMessage: err.Error(),
				},
			},
		})
		return
	}

	for k, v := range cmd.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("X-ApiSentinel-Agent-Replay", "true")

	resp, err := c.httpClient.Do(req)
	latencyMs := time.Since(startTime).Milliseconds()

	if err != nil {
		color.Red("❌ Local replay to %s failed: %v", cmd.TargetUrl, err)
		stream.Send(&agentv1.AgentMessage{
			AgentId: c.agentID,
			Payload: &agentv1.AgentMessage_ReplayResult{
				ReplayResult: &replayv1.ReplayResponse{
					JobId:        cmd.JobId,
					LatencyMs:    latencyMs,
					ErrorMessage: err.Error(),
				},
			},
		})
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	color.Green("✅ Local replay completed! HTTP %d in %d ms", resp.StatusCode, latencyMs)

	stream.Send(&agentv1.AgentMessage{
		AgentId: c.agentID,
		Payload: &agentv1.AgentMessage_ReplayResult{
			ReplayResult: &replayv1.ReplayResponse{
				JobId:          cmd.JobId,
				ResponseStatus: int32(resp.StatusCode),
				Body:           respBody,
				LatencyMs:      latencyMs,
			},
		},
	})
}

// SyncScanResults sends scan findings to the ApiSentinel Cloud backend via unary gRPC call (#2.1, #2.5)
func (c *AgentClient) SyncScanResults(ctx context.Context, repository, branch, commitHash string, findings []*securityv1.SecurityFinding) (*agentv1.SyncScanResponse, error) {
	transportCredentials := credentials.TransportCredentials(insecure.NewCredentials())
	if os.Getenv("APISENTINEL_GRPC_TLS") == "true" {
		transportCredentials = credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: os.Getenv("APISENTINEL_GRPC_SERVER_NAME"),
		})
	}

	conn, err := grpc.DialContext(ctx, c.serverAddr, grpc.WithTransportCredentials(transportCredentials), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server at %s: %w", c.serverAddr, err)
	}
	defer conn.Close()

	client := agentv1.NewAgentServiceClient(conn)

	rpcCtx := ctx
	if c.token != "" {
		rpcCtx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+c.token, "x-agent-token", c.token)
	}

	return client.SyncScanResults(rpcCtx, &agentv1.SyncScanRequest{
		AgentId:    c.agentID,
		Repository: repository,
		Branch:     branch,
		CommitHash: commitHash,
		Findings:   findings,
	})
}

