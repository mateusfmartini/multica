-- name: ListIssues :many
SELECT * FROM issue
WHERE workspace_id = $1
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('priority')::text IS NULL OR priority = sqlc.narg('priority'))
  AND (sqlc.narg('assignee_id')::uuid IS NULL OR assignee_id = sqlc.narg('assignee_id'))
ORDER BY position ASC, created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetIssue :one
SELECT * FROM issue
WHERE id = $1;

-- name: CreateIssue :one
INSERT INTO issue (
    workspace_id, title, description, status, priority,
    assignee_type, assignee_id, creator_type, creator_id,
    parent_issue_id, acceptance_criteria, context_refs,
    position
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) RETURNING *;

-- name: UpdateIssue :one
UPDATE issue SET
    title = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    status = COALESCE(sqlc.narg('status'), status),
    priority = COALESCE(sqlc.narg('priority'), priority),
    assignee_type = sqlc.narg('assignee_type'),
    assignee_id = sqlc.narg('assignee_id'),
    position = COALESCE(sqlc.narg('position'), position),
    due_date = sqlc.narg('due_date'),
    acceptance_criteria = COALESCE(sqlc.narg('acceptance_criteria'), acceptance_criteria),
    context_refs = COALESCE(sqlc.narg('context_refs'), context_refs),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateIssueStatus :one
UPDATE issue SET
    status = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteIssue :exec
DELETE FROM issue WHERE id = $1;
