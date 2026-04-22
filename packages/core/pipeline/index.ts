import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { CreatePipelineRequest, Pipeline, PipelineColumn, PipelineColumnInput, UpdatePipelineRequest } from "../types";
import { api } from "../api";

export const pipelineKeys = {
  list: (wsId: string) => ["pipelines", wsId] as const,
  detail: (wsId: string, id: string) => ["pipelines", wsId, id] as const,
  columns: (wsId: string, id: string) => ["pipeline-columns", wsId, id] as const,
};

export function pipelineListOptions(wsId: string) {
  return queryOptions<Pipeline[]>({
    queryKey: pipelineKeys.list(wsId),
    queryFn: () => api.listPipelines(wsId),
  });
}

export function pipelineColumnsOptions(wsId: string, pipelineId: string) {
  return queryOptions<PipelineColumn[]>({
    queryKey: pipelineKeys.columns(wsId, pipelineId),
    queryFn: () => api.listPipelineColumns(wsId, pipelineId),
    enabled: Boolean(wsId) && Boolean(pipelineId),
  });
}

export function usePipelines(wsId: string) {
  return useQuery(pipelineListOptions(wsId));
}

export function usePipelineColumns(wsId: string, pipelineId: string) {
  return useQuery(pipelineColumnsOptions(wsId, pipelineId));
}

export function useCreatePipeline(wsId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreatePipelineRequest) => api.createPipeline(wsId, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: pipelineKeys.list(wsId) });
    },
  });
}

export function useUpdatePipeline(wsId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...data }: { id: string } & UpdatePipelineRequest) =>
      api.updatePipeline(wsId, id, data),
    onSuccess: (_, { id }) => {
      qc.invalidateQueries({ queryKey: pipelineKeys.list(wsId) });
      qc.invalidateQueries({ queryKey: pipelineKeys.detail(wsId, id) });
    },
  });
}

export function useDeletePipeline(wsId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (pipelineId: string) => api.deletePipeline(wsId, pipelineId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: pipelineKeys.list(wsId) });
    },
  });
}

export function useSetDefaultPipeline(wsId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (pipelineId: string) => api.setDefaultPipeline(wsId, pipelineId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: pipelineKeys.list(wsId) });
    },
  });
}

export function useSyncPipelineColumns(wsId: string, pipelineId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (columns: PipelineColumnInput[]) =>
      api.syncPipelineColumns(wsId, pipelineId, columns),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: pipelineKeys.columns(wsId, pipelineId) });
    },
  });
}
