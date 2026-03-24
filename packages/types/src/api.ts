import type { Issue, IssueStatus, IssuePriority, IssueAssigneeType } from "./issue.js";
import type { MemberRole } from "./workspace.js";

// Issue API
export interface CreateIssueRequest {
  title: string;
  description?: string;
  status?: IssueStatus;
  priority?: IssuePriority;
  assignee_type?: IssueAssigneeType;
  assignee_id?: string;
  parent_issue_id?: string;
  acceptance_criteria?: string[];
  context_refs?: string[];
}

export interface UpdateIssueRequest {
  title?: string;
  description?: string;
  status?: IssueStatus;
  priority?: IssuePriority;
  assignee_type?: IssueAssigneeType | null;
  assignee_id?: string | null;
  position?: number;
  due_date?: string | null;
  acceptance_criteria?: string[];
  context_refs?: string[];
}

export interface ListIssuesParams {
  limit?: number;
  offset?: number;
  workspace_id?: string;
  status?: IssueStatus;
  priority?: IssuePriority;
  assignee_id?: string;
}

export interface ListIssuesResponse {
  issues: Issue[];
  total: number;
}

export interface UpdateMeRequest {
  name?: string;
  avatar_url?: string;
}

export interface CreateMemberRequest {
  email: string;
  role?: MemberRole;
}

export interface UpdateMemberRequest {
  role: MemberRole;
}

// Pagination
export interface PaginationParams {
  limit?: number;
  offset?: number;
}
