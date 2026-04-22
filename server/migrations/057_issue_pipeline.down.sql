ALTER TABLE issue DROP COLUMN IF EXISTS pipeline_id;
ALTER TABLE issue ADD CONSTRAINT issue_status_check
  CHECK (status IN ('backlog', 'todo', 'in_progress', 'in_review', 'done', 'blocked', 'cancelled'));
