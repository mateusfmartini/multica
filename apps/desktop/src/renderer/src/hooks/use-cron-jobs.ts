import { useState, useEffect, useCallback } from 'react'

export interface CronJobInfo {
  id: string
  name: string
  description?: string
  enabled: boolean
  schedule: string
  sessionTarget: string
  nextRunAt: string | null
  lastStatus: 'ok' | 'error' | 'skipped' | null
  lastRunAt: string | null
  lastDurationMs: number | null
  lastError: string | null
}

export interface UseCronJobsReturn {
  jobs: CronJobInfo[]
  loading: boolean
  error: string | null
  toggleJob: (jobId: string) => Promise<void>
  removeJob: (jobId: string) => Promise<void>
  refresh: () => Promise<void>
}

export function useCronJobs(): UseCronJobsReturn {
  const [jobs, setJobs] = useState<CronJobInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchJobs = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)

      const result = window.electronAPI
        ? await window.electronAPI.cron.list()
        : await window.ipcRenderer.invoke('cron:list')

      if (Array.isArray(result)) {
        setJobs(result)
      } else {
        setError('Invalid response from cron:list')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch cron jobs')
      setJobs([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchJobs()
  }, [fetchJobs])

  const toggleJob = useCallback(async (jobId: string) => {
    try {
      const result = window.electronAPI
        ? await window.electronAPI.cron.toggle(jobId)
        : await window.ipcRenderer.invoke('cron:toggle', jobId)

      const typed = result as { error?: string; id?: string; enabled?: boolean }
      if (typed.error) {
        setError(typed.error)
        return
      }

      setJobs((prev) =>
        prev.map((job) =>
          job.id === jobId ? { ...job, enabled: typed.enabled ?? !job.enabled } : job
        )
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to toggle job')
    }
  }, [])

  const removeJob = useCallback(async (jobId: string) => {
    try {
      const result = window.electronAPI
        ? await window.electronAPI.cron.remove(jobId)
        : await window.ipcRenderer.invoke('cron:remove', jobId)

      const typed = result as { error?: string; ok?: boolean }
      if (typed.error) {
        setError(typed.error)
        return
      }

      setJobs((prev) => prev.filter((job) => job.id !== jobId))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove job')
    }
  }, [])

  return {
    jobs,
    loading,
    error,
    toggleJob,
    removeJob,
    refresh: fetchJobs,
  }
}

export default useCronJobs
