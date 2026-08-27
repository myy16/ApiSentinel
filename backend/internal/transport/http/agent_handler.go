package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/apisentinel/apisentinel/internal/middleware"
	grpcTransport "github.com/apisentinel/apisentinel/internal/transport/grpc"
)

// AgentSessionDTO is the public representation of a connected agent
type AgentSessionDTO struct {
	AgentID  string    `json:"agentId"`
	Hostname string    `json:"hostname"`
	OS       string    `json:"os"`
	Version  string    `json:"version"`
	Status   string    `json:"status"`
	LastSeen time.Time `json:"lastSeen"`
}

type AgentHandler struct {
	grpcServer *grpcTransport.Server
}

func NewAgentHandler(grpcServer *grpcTransport.Server) *AgentHandler {
	return &AgentHandler{grpcServer: grpcServer}
}

// ListSessions returns all currently connected gRPC agent sessions
func (h *AgentHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	organizationID, _ := r.Context().Value(middleware.OrgIDKey).(string)
	sessions := h.grpcServer.GetActiveSessionsByOrganization(organizationID)

	dtos := make([]AgentSessionDTO, 0, len(sessions))
	for _, s := range sessions {
		status := "ONLINE"
		if time.Since(s.LastSeen) > 2*time.Minute {
			status = "STALE"
		}
		dtos = append(dtos, AgentSessionDTO{
			AgentID:  s.AgentID,
			Hostname: s.Hostname,
			OS:       s.OS,
			Version:  s.Version,
			Status:   status,
			LastSeen: s.LastSeen,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agents": dtos,
		"total":  len(dtos),
	})
}
