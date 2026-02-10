/**
 * Hook for managing channel accounts (Telegram, Discord, etc.) in the Desktop App.
 *
 * Provides state and actions for the Channels settings page:
 * - List channel account states (running / stopped / error)
 * - Read channel config (tokens)
 * - Save / remove tokens with immediate start/stop
 */
import { useState, useEffect, useCallback } from 'react'

export interface UseChannelsReturn {
  /** Runtime states of all channel accounts */
  states: ChannelAccountStateInfo[]
  /** Raw channel config from credentials.json5 */
  config: Record<string, Record<string, Record<string, unknown>> | undefined>
  /** Loading state */
  loading: boolean
  /** Error message if any */
  error: string | null
  /** Refresh states and config */
  refresh: () => Promise<void>
  /** Save a bot token — persists to file and starts the bot immediately */
  saveToken: (channelId: string, accountId: string, token: string) => Promise<{ ok: boolean; error?: string }>
  /** Remove a bot token — stops the bot and removes from file */
  removeToken: (channelId: string, accountId: string) => Promise<{ ok: boolean; error?: string }>
  /** Stop a channel account without removing config */
  stopChannel: (channelId: string, accountId: string) => Promise<void>
  /** Start a channel account from saved config */
  startChannel: (channelId: string, accountId: string) => Promise<void>
}

export function useChannels(): UseChannelsReturn {
  const [states, setStates] = useState<ChannelAccountStateInfo[]>([])
  const [config, setConfig] = useState<Record<string, Record<string, Record<string, unknown>> | undefined>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    setError(null)

    try {
      const [stateList, channelConfig] = await Promise.all([
        window.electronAPI.channels.listStates(),
        window.electronAPI.channels.getConfig(),
      ])

      setStates(stateList)
      setConfig(channelConfig)
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      setError(message)
      console.error('[useChannels] Failed to load:', message)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
  }, [refresh])

  const saveToken = useCallback(async (channelId: string, accountId: string, token: string) => {
    setError(null)
    try {
      const result = await window.electronAPI.channels.saveToken(channelId, accountId, token)
      if (!result.ok) {
        setError(result.error ?? 'Failed to save token')
      }
      // Refresh to pick up new state
      await refresh()
      return result
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      setError(message)
      return { ok: false, error: message }
    }
  }, [refresh])

  const removeToken = useCallback(async (channelId: string, accountId: string) => {
    setError(null)
    try {
      const result = await window.electronAPI.channels.removeToken(channelId, accountId)
      if (!result.ok) {
        setError(result.error ?? 'Failed to remove token')
      }
      await refresh()
      return result
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      setError(message)
      return { ok: false, error: message }
    }
  }, [refresh])

  const stopChannel = useCallback(async (channelId: string, accountId: string) => {
    setError(null)
    try {
      const result = await window.electronAPI.channels.stop(channelId, accountId)
      if (!result.ok) {
        setError(result.error ?? 'Failed to stop channel')
      }
      await refresh()
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      setError(message)
    }
  }, [refresh])

  const startChannel = useCallback(async (channelId: string, accountId: string) => {
    setError(null)
    try {
      const result = await window.electronAPI.channels.start(channelId, accountId)
      if (!result.ok) {
        setError(result.error ?? 'Failed to start channel')
      }
      await refresh()
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      setError(message)
    }
  }, [refresh])

  return {
    states,
    config,
    loading,
    error,
    refresh,
    saveToken,
    removeToken,
    stopChannel,
    startChannel,
  }
}
