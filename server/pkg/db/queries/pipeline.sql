-- name: GetPipeline :one
SELECT * FROM pipeline
WHERE id = $1;

-- name: CreatePipeline :one
INSERT INTO pipeline (workspace_id, name, description, is_default)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdatePipeline :one
UPDATE pipeline
SET name = $2, description = $3, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeletePipeline :exec
UPDATE pipeline SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL;

-- name: ClearDefaultPipelines :exec
UPDATE pipeline SET is_default = FALSE WHERE workspace_id = $1 AND deleted_at IS NULL;

-- name: MarkPipelineAsDefault :exec
UPDATE pipeline SET is_default = TRUE, updated_at = NOW() WHERE id = $1;

-- name: InsertPipelineColumn :one
INSERT INTO pipeline_column (pipeline_id, status_key, label, position, is_terminal, instructions, allowed_transitions)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: DeletePipelineColumnsByPipeline :exec
DELETE FROM pipeline_column WHERE pipeline_id = $1;

-- name: ListPipelinesByWorkspace :many
SELECT * FROM pipeline
WHERE workspace_id = $1
  AND (sqlc.arg(include_deleted)::bool OR deleted_at IS NULL)
ORDER BY created_at ASC;

-- name: ListPipelineColumns :many
SELECT * FROM pipeline_column
WHERE pipeline_id = $1
ORDER BY position ASC;
