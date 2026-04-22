export interface Pipeline {
  id: string;
  workspace_id: string;
  name: string;
  description: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
}

export interface PipelineColumn {
  id: string;
  pipeline_id: string;
  status_key: string;
  label: string;
  position: number;
  is_terminal: boolean;
  instructions: string;
  allowed_transitions: string[];
  created_at: string;
  updated_at: string;
}

export interface CreatePipelineRequest {
  name: string;
  description: string;
  is_default?: boolean;
}

export interface UpdatePipelineRequest {
  name: string;
  description: string;
}

export interface PipelineColumnInput {
  status_key: string;
  label: string;
  position: number;
  is_terminal: boolean;
  instructions: string;
  allowed_transitions: string[];
}
