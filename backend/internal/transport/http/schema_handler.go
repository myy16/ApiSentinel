package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/apisentinel/apisentinel/internal/database"
	"github.com/apisentinel/apisentinel/internal/security/schema"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type SchemaHandler struct {
	queries *database.Queries
}

func NewSchemaHandler(queries *database.Queries) *SchemaHandler {
	return &SchemaHandler{queries: queries}
}

func getEndpointIDParam(r *http.Request) string {
	id := chi.URLParam(r, "endpointId")
	if id == "" {
		id = chi.URLParam(r, "id")
	}
	return id
}

// ListBaselines lists all schema baseline versions for an endpoint.
func (h *SchemaHandler) ListBaselines(w http.ResponseWriter, r *http.Request) {
	endpointIDStr := getEndpointIDParam(r)
	endpointUUID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ENDPOINT_ID", "Invalid endpoint ID format")
		return
	}

	baselines, err := h.queries.ListSchemaBaselinesByEndpoint(r.Context(), pgtype.UUID{Bytes: endpointUUID, Valid: true})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list schema baselines")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"baselines": baselines,
		"count":     len(baselines),
	})
}

// GetActiveBaseline returns the currently active schema baseline for an endpoint.
func (h *SchemaHandler) GetActiveBaseline(w http.ResponseWriter, r *http.Request) {
	endpointIDStr := getEndpointIDParam(r)
	endpointUUID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ENDPOINT_ID", "Invalid endpoint ID format")
		return
	}

	active, err := h.queries.GetActiveSchemaBaseline(r.Context(), pgtype.UUID{Bytes: endpointUUID, Valid: true})
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"active": nil,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"active": active,
	})
}

type InferSchemaInput struct {
	Payload  string `json:"payload"`
	Activate bool   `json:"activate"`
}

// InferBaseline generates a new JSON schema baseline from a sample payload.
func (h *SchemaHandler) InferBaseline(w http.ResponseWriter, r *http.Request) {
	endpointIDStr := getEndpointIDParam(r)
	endpointUUID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ENDPOINT_ID", "Invalid endpoint ID format")
		return
	}

	var input InferSchemaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Payload == "" {
		writeError(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Sample JSON payload is required")
		return
	}

	schemaJSON, err := schema.InferJSONSchema([]byte(input.Payload))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INFERENCE_FAILED", "Failed to infer schema: "+err.Error())
		return
	}

	// Verify schema syntax with validator
	if _, err := schema.NewValidator(schemaJSON); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_INFERRED_SCHEMA", "Inferred schema syntax error: "+err.Error())
		return
	}

	// Determine next version
	nextVer, err := h.queries.GetNextSchemaVersion(r.Context(), pgtype.UUID{Bytes: endpointUUID, Valid: true})
	if err != nil {
		nextVer = 1
	}

	if input.Activate {
		_ = h.queries.DeactivateAllSchemaBaselines(r.Context(), pgtype.UUID{Bytes: endpointUUID, Valid: true})
	}

	baseline, err := h.queries.CreateSchemaBaseline(r.Context(), database.CreateSchemaBaselineParams{
		EndpointID: pgtype.UUID{Bytes: endpointUUID, Valid: true},
		Version:    nextVer,
		SchemaJson: []byte(schemaJSON),
		Source:     "INFERRED_PAYLOAD",
		IsActive:   input.Activate,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to save schema baseline: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, baseline)
}

type OpenAPIImportInput struct {
	Spec          string `json:"spec"`
	OperationPath string `json:"operationPath"`
	Activate      bool   `json:"activate"`
}

// ImportOpenAPI extracts and saves a schema baseline from an OpenAPI spec.
func (h *SchemaHandler) ImportOpenAPI(w http.ResponseWriter, r *http.Request) {
	endpointIDStr := getEndpointIDParam(r)
	endpointUUID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ENDPOINT_ID", "Invalid endpoint ID format")
		return
	}

	var input OpenAPIImportInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Spec == "" {
		writeError(w, http.StatusBadRequest, "INVALID_SPEC", "OpenAPI spec content is required")
		return
	}

	schemaJSON, err := schema.ExtractSchemaFromOpenAPI([]byte(input.Spec), input.OperationPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, "OPENAPI_EXTRACT_FAILED", "Failed to extract schema from OpenAPI: "+err.Error())
		return
	}

	if _, err := schema.NewValidator(schemaJSON); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_OPENAPI_SCHEMA", "Extracted schema syntax error: "+err.Error())
		return
	}

	nextVer, err := h.queries.GetNextSchemaVersion(r.Context(), pgtype.UUID{Bytes: endpointUUID, Valid: true})
	if err != nil {
		nextVer = 1
	}

	if input.Activate {
		_ = h.queries.DeactivateAllSchemaBaselines(r.Context(), pgtype.UUID{Bytes: endpointUUID, Valid: true})
	}

	baseline, err := h.queries.CreateSchemaBaseline(r.Context(), database.CreateSchemaBaselineParams{
		EndpointID: pgtype.UUID{Bytes: endpointUUID, Valid: true},
		Version:    nextVer,
		SchemaJson: []byte(schemaJSON),
		Source:     "OPENAPI",
		IsActive:   input.Activate,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to save schema baseline: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, baseline)
}

type SaveManualSchemaInput struct {
	SchemaJSON string `json:"schemaJson"`
	Activate   bool   `json:"activate"`
}

// SaveManual creates a manual JSON schema baseline version.
func (h *SchemaHandler) SaveManual(w http.ResponseWriter, r *http.Request) {
	endpointIDStr := getEndpointIDParam(r)
	endpointUUID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ENDPOINT_ID", "Invalid endpoint ID format")
		return
	}

	var input SaveManualSchemaInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.SchemaJSON == "" {
		writeError(w, http.StatusBadRequest, "INVALID_SCHEMA", "JSON schema content is required")
		return
	}

	if _, err := schema.NewValidator(input.SchemaJSON); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_SCHEMA_SYNTAX", "Invalid JSON Schema syntax: "+err.Error())
		return
	}

	nextVer, err := h.queries.GetNextSchemaVersion(r.Context(), pgtype.UUID{Bytes: endpointUUID, Valid: true})
	if err != nil {
		nextVer = 1
	}

	if input.Activate {
		_ = h.queries.DeactivateAllSchemaBaselines(r.Context(), pgtype.UUID{Bytes: endpointUUID, Valid: true})
	}

	baseline, err := h.queries.CreateSchemaBaseline(r.Context(), database.CreateSchemaBaselineParams{
		EndpointID: pgtype.UUID{Bytes: endpointUUID, Valid: true},
		Version:    nextVer,
		SchemaJson: []byte(input.SchemaJSON),
		Source:     "MANUAL",
		IsActive:   input.Activate,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to save schema baseline: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, baseline)
}

// ActivateBaseline sets a specific schema baseline version as active.
func (h *SchemaHandler) ActivateBaseline(w http.ResponseWriter, r *http.Request) {
	endpointIDStr := getEndpointIDParam(r)
	endpointUUID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ENDPOINT_ID", "Invalid endpoint ID format")
		return
	}

	schemaIDStr := chi.URLParam(r, "schemaId")
	schemaUUID, err := uuid.Parse(schemaIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_SCHEMA_ID", "Invalid schema ID format")
		return
	}

	_ = h.queries.DeactivateAllSchemaBaselines(r.Context(), pgtype.UUID{Bytes: endpointUUID, Valid: true})

	updated, err := h.queries.ActivateSchemaBaseline(r.Context(), database.ActivateSchemaBaselineParams{
		ID:         pgtype.UUID{Bytes: schemaUUID, Valid: true},
		EndpointID: pgtype.UUID{Bytes: endpointUUID, Valid: true},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to activate schema baseline: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// ListDrifts lists detected schema drift events for an endpoint.
func (h *SchemaHandler) ListDrifts(w http.ResponseWriter, r *http.Request) {
	endpointIDStr := getEndpointIDParam(r)
	endpointUUID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ENDPOINT_ID", "Invalid endpoint ID format")
		return
	}

	drifts, err := h.queries.ListSchemaDriftsByEndpoint(r.Context(), pgtype.UUID{Bytes: endpointUUID, Valid: true})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list schema drifts")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"drifts": drifts,
		"count":  len(drifts),
	})
}

// DismissDrift acknowledges and hides a drift event.
func (h *SchemaHandler) DismissDrift(w http.ResponseWriter, r *http.Request) {
	endpointIDStr := getEndpointIDParam(r)
	endpointUUID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ENDPOINT_ID", "Invalid endpoint ID format")
		return
	}

	driftIDStr := chi.URLParam(r, "driftId")
	driftUUID, err := uuid.Parse(driftIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_DRIFT_ID", "Invalid drift ID format")
		return
	}

	updated, err := h.queries.AcknowledgeSchemaDrift(r.Context(), database.AcknowledgeSchemaDriftParams{
		ID:         pgtype.UUID{Bytes: driftUUID, Valid: true},
		EndpointID: pgtype.UUID{Bytes: endpointUUID, Valid: true},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to acknowledge drift event")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// AcceptDrift adopts the payload from a drift event and increments the baseline to the next version.
func (h *SchemaHandler) AcceptDrift(w http.ResponseWriter, r *http.Request) {
	endpointIDStr := getEndpointIDParam(r)
	endpointUUID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ENDPOINT_ID", "Invalid endpoint ID format")
		return
	}

	driftIDStr := chi.URLParam(r, "driftId")
	driftUUID, err := uuid.Parse(driftIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_DRIFT_ID", "Invalid drift ID format")
		return
	}

	// 1. Acknowledge drift
	_, _ = h.queries.AcknowledgeSchemaDrift(r.Context(), database.AcknowledgeSchemaDriftParams{
		ID:         pgtype.UUID{Bytes: driftUUID, Valid: true},
		EndpointID: pgtype.UUID{Bytes: endpointUUID, Valid: true},
	})

	// 2. Fetch associated captured request
	drifts, _ := h.queries.ListSchemaDriftsByEndpoint(r.Context(), pgtype.UUID{Bytes: endpointUUID, Valid: true})
	var targetRequestID pgtype.UUID
	for _, d := range drifts {
		if d.ID.Bytes == driftUUID {
			targetRequestID = d.RequestID
			break
		}
	}

	if !targetRequestID.Valid {
		writeError(w, http.StatusNotFound, "REQUEST_NOT_FOUND", "Associated request for drift event not found")
		return
	}

	captured, err := h.queries.GetCapturedRequestByID(r.Context(), targetRequestID)
	if err != nil {
		writeError(w, http.StatusNotFound, "REQUEST_NOT_FOUND", "Could not load captured request payload")
		return
	}

	payloadBytes := captured.ParsedJson
	if len(payloadBytes) == 0 && captured.MaskedBody.Valid {
		payloadBytes = []byte(captured.MaskedBody.String)
	}

	schemaJSON, err := schema.InferJSONSchema(payloadBytes)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INFERENCE_FAILED", "Failed to infer updated schema: "+err.Error())
		return
	}

	nextVer, err := h.queries.GetNextSchemaVersion(r.Context(), pgtype.UUID{Bytes: endpointUUID, Valid: true})
	if err != nil {
		nextVer = 1
	}

	_ = h.queries.DeactivateAllSchemaBaselines(r.Context(), pgtype.UUID{Bytes: endpointUUID, Valid: true})

	baseline, err := h.queries.CreateSchemaBaseline(r.Context(), database.CreateSchemaBaselineParams{
		EndpointID: pgtype.UUID{Bytes: endpointUUID, Valid: true},
		Version:    nextVer,
		SchemaJson: []byte(schemaJSON),
		Source:     "INFERRED_PAYLOAD",
		IsActive:   true,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create updated baseline")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":  fmt.Sprintf("Şema sapması kabul edildi ve v%d aktif baseline olarak güncellendi.", baseline.Version),
		"baseline": baseline,
	})
}
