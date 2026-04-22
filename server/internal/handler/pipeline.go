package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

type PipelineResponse struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type PipelineColumnResponse struct {
	ID                 string   `json:"id"`
	PipelineID         string   `json:"pipeline_id"`
	StatusKey          string   `json:"status_key"`
	Label              string   `json:"label"`
	Position           int32    `json:"position"`
	IsTerminal         bool     `json:"is_terminal"`
	Instructions       string   `json:"instructions"`
	AllowedTransitions []string `json:"allowed_transitions"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
}

func pipelineToResponse(p db.Pipeline) PipelineResponse {
	return PipelineResponse{
		ID:          uuidToString(p.ID),
		WorkspaceID: uuidToString(p.WorkspaceID),
		Name:        p.Name,
		Description: p.Description,
		IsDefault:   p.IsDefault,
		CreatedAt:   timestampToString(p.CreatedAt),
		UpdatedAt:   timestampToString(p.UpdatedAt),
	}
}

func pipelineColumnToResponse(c db.PipelineColumn) PipelineColumnResponse {
	transitions := c.AllowedTransitions
	if transitions == nil {
		transitions = []string{}
	}
	return PipelineColumnResponse{
		ID:                 uuidToString(c.ID),
		PipelineID:         uuidToString(c.PipelineID),
		StatusKey:          c.StatusKey,
		Label:              c.Label,
		Position:           c.Position,
		IsTerminal:         c.IsTerminal,
		Instructions:       c.Instructions,
		AllowedTransitions: transitions,
		CreatedAt:          timestampToString(c.CreatedAt),
		UpdatedAt:          timestampToString(c.UpdatedAt),
	}
}

type CreatePipelineRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
}

type UpdatePipelineRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PipelineColumnInput struct {
	StatusKey          string   `json:"status_key"`
	Label              string   `json:"label"`
	Position           int32    `json:"position"`
	IsTerminal         bool     `json:"is_terminal"`
	Instructions       string   `json:"instructions"`
	AllowedTransitions []string `json:"allowed_transitions"`
}

func (h *Handler) ListPipelines(w http.ResponseWriter, r *http.Request) {
	workspaceID := workspaceIDFromURL(r, "id")

	pipelines, err := h.Queries.ListPipelinesByWorkspace(r.Context(), db.ListPipelinesByWorkspaceParams{
		WorkspaceID:    parseUUID(workspaceID),
		IncludeDeleted: false,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list pipelines")
		return
	}

	resp := make([]PipelineResponse, 0, len(pipelines))
	for _, p := range pipelines {
		resp = append(resp, pipelineToResponse(p))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetPipeline(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "pipelineId")

	pipeline, err := h.Queries.GetPipeline(r.Context(), parseUUID(pipelineID))
	if err != nil {
		writeError(w, http.StatusNotFound, "pipeline not found")
		return
	}

	writeJSON(w, http.StatusOK, pipelineToResponse(pipeline))
}

func (h *Handler) CreatePipeline(w http.ResponseWriter, r *http.Request) {
	workspaceID := workspaceIDFromURL(r, "id")

	var req CreatePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	pipeline, err := h.Queries.CreatePipeline(r.Context(), db.CreatePipelineParams{
		WorkspaceID: parseUUID(workspaceID),
		Name:        req.Name,
		Description: req.Description,
		IsDefault:   req.IsDefault,
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "a pipeline with this name already exists")
			return
		}
		slog.Warn("failed to create pipeline", "workspace_id", workspaceID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create pipeline")
		return
	}

	writeJSON(w, http.StatusCreated, pipelineToResponse(pipeline))
}

func (h *Handler) UpdatePipeline(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "pipelineId")

	var req UpdatePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	pipeline, err := h.Queries.UpdatePipeline(r.Context(), db.UpdatePipelineParams{
		ID:          parseUUID(pipelineID),
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "a pipeline with this name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update pipeline")
		return
	}

	writeJSON(w, http.StatusOK, pipelineToResponse(pipeline))
}

func (h *Handler) DeletePipeline(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "pipelineId")

	if err := h.Queries.SoftDeletePipeline(r.Context(), parseUUID(pipelineID)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete pipeline")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SetDefaultPipeline(w http.ResponseWriter, r *http.Request) {
	workspaceID := workspaceIDFromURL(r, "id")
	pipelineID := chi.URLParam(r, "pipelineId")

	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer tx.Rollback(r.Context())

	qtx := h.Queries.WithTx(tx)

	if err := qtx.ClearDefaultPipelines(r.Context(), parseUUID(workspaceID)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update pipelines")
		return
	}
	if err := qtx.MarkPipelineAsDefault(r.Context(), parseUUID(pipelineID)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to set default pipeline")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListPipelineColumns(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "pipelineId")

	columns, err := h.Queries.ListPipelineColumns(r.Context(), parseUUID(pipelineID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list pipeline columns")
		return
	}

	resp := make([]PipelineColumnResponse, 0, len(columns))
	for _, c := range columns {
		resp = append(resp, pipelineColumnToResponse(c))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) SyncPipelineColumns(w http.ResponseWriter, r *http.Request) {
	pipelineID := chi.URLParam(r, "pipelineId")

	var inputs []PipelineColumnInput
	if err := json.NewDecoder(r.Body).Decode(&inputs); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validatePipelineColumns(inputs); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer tx.Rollback(r.Context())

	qtx := h.Queries.WithTx(tx)
	pid := parseUUID(pipelineID)

	if err := qtx.DeletePipelineColumnsByPipeline(r.Context(), pid); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to replace pipeline columns")
		return
	}

	result := make([]PipelineColumnResponse, 0, len(inputs))
	for _, input := range inputs {
		transitions := input.AllowedTransitions
		if transitions == nil {
			transitions = []string{}
		}
		col, err := qtx.InsertPipelineColumn(r.Context(), db.InsertPipelineColumnParams{
			PipelineID:         pid,
			StatusKey:          input.StatusKey,
			Label:              input.Label,
			Position:           input.Position,
			IsTerminal:         input.IsTerminal,
			Instructions:       input.Instructions,
			AllowedTransitions: transitions,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to insert pipeline column")
			return
		}
		result = append(result, pipelineColumnToResponse(col))
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func validatePipelineColumns(inputs []PipelineColumnInput) error {
	seenKeys := map[string]struct{}{}
	seenPositions := map[int32]struct{}{}
	for _, col := range inputs {
		key := strings.TrimSpace(col.StatusKey)
		if key == "" {
			return fmt.Errorf("status_key cannot be empty")
		}
		if strings.ContainsAny(key, " \t\n") {
			return fmt.Errorf("status_key cannot contain whitespace")
		}
		if _, dup := seenKeys[key]; dup {
			return fmt.Errorf("duplicate status_key: %s", key)
		}
		seenKeys[key] = struct{}{}
		if _, dup := seenPositions[col.Position]; dup {
			return fmt.Errorf("duplicate position")
		}
		seenPositions[col.Position] = struct{}{}
	}
	return nil
}
