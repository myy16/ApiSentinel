package agent

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	agentv1 "github.com/apisentinel/apisentinel/pkg/genproto/proto/agent/v1"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// CloudClient coordinates bidirectional gRPC streaming with ApiSentinel Cloud
type CloudClient struct {
	serverAddr string
	token      string
	agentID    string
	version    string
}

func NewCloudClient(serverAddr, token string) *CloudClient {
	if serverAddr == "" {
		serverAddr = "localhost:50051"
	}

	hostname, _ := os.Hostname()
	agentID := fmt.Sprintf("agent_%s_%s", hostname, uuid.New().String()[:8])

	return &CloudClient{
		serverAddr: serverAddr,
		token:      token,
		agentID:    agentID,
		version:    "0.2.0",
	}
}

// Connect establishes a persistent bidirectional gRPC stream with ApiSentinel Cloud
func (c *CloudClient) Connect(ctx context.Context) error {
	log.Info().Str("server", c.serverAddr).Str("agentId", c.agentID).Msg("Connecting to ApiSentinel Cloud...")

	conn, err := grpc.NewClient(
		c.serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to dial gRPC server: %w", err)
	}
	defer conn.Close()

	client := agentv1.NewAgentServiceClient(conn)

	// Attach authentication token in metadata
	md := metadata.Pairs(
		"authorization", "Bearer "+c.token,
		"x-agent-token", c.token,
		"x-agent-id", c.agentID,
	)
	streamCtx := metadata.NewOutgoingContext(ctx, md)

	stream, err := client.ConnectSession(streamCtx)
	if err != nil {
		return fmt.Errorf("failed to open gRPC bidirectional stream: %w", err)
	}

	log.Info().Msg("✅ Successfully connected to ApiSentinel Cloud (gRPC Stream Active)")

	// Goroutine for receiving server push messages (policy updates, acks)
	errChan := make(chan error, 2)
	go func() {
		for {
			cloudMsg, rErr := stream.Recv()
			if rErr != nil {
				errChan <- fmt.Errorf("gRPC stream receive error: %w", rErr)
				return
			}
			c.handleCloudMessage(cloudMsg)
		}
	}()

	// Goroutine for sending periodic heartbeats
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		// Send initial heartbeat immediately
		if sErr := c.sendHeartbeat(stream); sErr != nil {
			errChan <- sErr
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if sErr := c.sendHeartbeat(stream); sErr != nil {
					errChan <- sErr
					return
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		log.Info().Msg("Closing gRPC agent stream session...")
		_ = stream.CloseSend()
		return nil
	case err := <-errChan:
		return err
	}
}

func (c *CloudClient) sendHeartbeat(stream agentv1.AgentService_ConnectSessionClient) error {
	hostname, _ := os.Hostname()

	heartbeatMsg := &agentv1.AgentMessage{
		AgentId: c.agentID,
		Payload: &agentv1.AgentMessage_Heartbeat{
			Heartbeat: &agentv1.AgentHeartbeat{
				Hostname:  hostname,
				Os:        runtime.GOOS + "/" + runtime.GOARCH,
				Version:   c.version,
				Timestamp: time.Now().Unix(),
			},
		},
	}

	if err := stream.Send(heartbeatMsg); err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	log.Debug().Str("agentId", c.agentID).Msg("Heartbeat sent to Cloud")
	return nil
}

func (c *CloudClient) handleCloudMessage(msg *agentv1.CloudMessage) {
	if ack := msg.GetHeartbeatAck(); ack != nil {
		log.Info().Int64("serverTime", ack.Timestamp).Msg("💓 Heartbeat ACK received from Cloud")
	} else if cfg := msg.GetConfigUpdate(); cfg != nil {
		log.Info().Msg("🛡️ Received live configuration update from Cloud")
	} else if replay := msg.GetReplayCommand(); replay != nil {
		log.Info().Str("targetUrl", replay.TargetUrl).Msg("🔁 Received Replay command from Cloud")
	} else {
		log.Info().Interface("msg", msg).Msg("Received message from Cloud")
	}
}
