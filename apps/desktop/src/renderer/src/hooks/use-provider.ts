/**
 * Hook for managing LLM providers in the Desktop App.
 *
 * Provides functionality similar to CLI `/provider` command:
 * - List all providers with status
 * - Get current provider/model
 * - Switch provider/model
 */
import { useState, useEffect, useCallback } from 'react'

// Types are defined in electron-env.d.ts and available globally

interface UseProviderReturn {
  /** All providers with their status */
  providers: ProviderStatus[]
  /** Only available (configured) providers */
  availableProviders: ProviderStatus[]
  /** Current provider and model info */
  current: CurrentProviderInfo | null
  /** Loading state */
  loading: boolean
  /** Error message if any */
  error: string | null
  /** Refresh provider list and current status */
  refresh: () => Promise<void>
  /** Switch to a different provider (and optionally model) */
  setProvider: (providerId: string, modelId?: string) => Promise<{ ok: boolean; error?: string }>
  /** Get metadata for a specific provider */
  getProviderMeta: (providerId: string) => ProviderStatus | undefined
}

export function useProvider(): UseProviderReturn {
  const [providers, setProviders] = useState<ProviderStatus[]>([])
  const [current, setCurrent] = useState<CurrentProviderInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    setError(null)

    try {
      const [providerList, currentInfo] = await Promise.all([
        window.electronAPI.provider.list(),
        window.electronAPI.provider.current(),
      ])

      setProviders(providerList)
      setCurrent(currentInfo)
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      setError(message)
      console.error('[useProvider] Failed to load providers:', message)
    } finally {
      setLoading(false)
    }
  }, [])

  // Load providers on mount
  useEffect(() => {
    refresh()
  }, [refresh])

  const setProvider = useCallback(async (providerId: string, modelId?: string) => {
    setError(null)

    try {
      const result = await window.electronAPI.provider.set(providerId, modelId)

      if (result.ok) {
        // Refresh to update current status
        await refresh()
        return { ok: true }
      } else {
        setError(result.error ?? 'Unknown error')
        return { ok: false, error: result.error }
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      setError(message)
      return { ok: false, error: message }
    }
  }, [refresh])

  const getProviderMeta = useCallback((providerId: string) => {
    return providers.find((p) => p.id === providerId)
  }, [providers])

  const availableProviders = providers.filter((p) => p.available)

  return {
    providers,
    availableProviders,
    current,
    loading,
    error,
    refresh,
    setProvider,
    getProviderMeta,
  }
}
