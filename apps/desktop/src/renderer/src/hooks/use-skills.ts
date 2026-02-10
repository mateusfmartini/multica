import { useState, useEffect, useCallback, useMemo } from 'react'

// ============================================================================
// Types matching the IPC response from main process
// ============================================================================

export type SkillSource = 'bundled' | 'global' | 'profile'

export interface SkillInfo {
  id: string
  name: string
  description: string
  version: string
  enabled: boolean
  source: SkillSource
  triggers: string[]
}

export interface SkillGroup {
  source: SkillSource
  name: string
  skills: SkillInfo[]
}

// Source display names
const SOURCE_NAMES: Record<string, string> = {
  bundled: 'Built-in Skills',
  global: 'Global Skills',
  profile: 'Profile Skills',
}

export interface UseSkillsReturn {
  /** List of all skills */
  skills: SkillInfo[]
  /** Skills grouped by source */
  groups: SkillGroup[]
  /** Loading state */
  loading: boolean
  /** Error state */
  error: string | null

  /** Toggle a skill on/off */
  toggleSkill: (skillId: string) => Promise<void>
  /** Enable a skill */
  enableSkill: (skillId: string) => Promise<void>
  /** Disable a skill */
  disableSkill: (skillId: string) => Promise<void>

  /** Refresh skills list */
  refresh: () => Promise<void>

  /** Get skill by ID */
  getSkill: (id: string) => SkillInfo | undefined

  /** Filter skills by search query */
  filterSkills: (query: string) => SkillInfo[]

  /** Check if a skill is enabled */
  isSkillEnabled: (skillId: string) => boolean

  /** Stats */
  stats: {
    total: number
    enabled: number
    disabled: number
    bundled: number
    global: number
    profile: number
  }
}

/**
 * Hook for managing Agent skills configuration via IPC.
 *
 * This hook communicates with the Electron main process to:
 * - Fetch the list of all skills (bundled, global, profile)
 * - Toggle skills on/off
 * - Match the CLI `multica skills list` output
 */
export function useSkills(): UseSkillsReturn {
  const [skills, setSkills] = useState<SkillInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Fetch skills from main process
  const fetchSkills = useCallback(async () => {
    try {
      setLoading(true)
      setError(null)

      // Use new electronAPI if available, fallback to ipcRenderer
      const result = window.electronAPI
        ? await window.electronAPI.skills.list()
        : await window.ipcRenderer.invoke('skills:list')

      if (Array.isArray(result)) {
        setSkills(result)
      } else {
        setError('Invalid response from skills:list')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch skills')
      setSkills([])
    } finally {
      setLoading(false)
    }
  }, [])

  // Initial fetch
  useEffect(() => {
    fetchSkills()
  }, [fetchSkills])

  // Group skills by source
  const groups = useMemo<SkillGroup[]>(() => {
    const sourceOrder: SkillSource[] = ['bundled', 'global', 'profile']
    const groupMap = new Map<SkillSource, SkillInfo[]>()

    for (const skill of skills) {
      const sourceSkills = groupMap.get(skill.source) || []
      sourceSkills.push(skill)
      groupMap.set(skill.source, sourceSkills)
    }

    return sourceOrder
      .filter((source) => groupMap.has(source))
      .map((source) => ({
        source,
        name: SOURCE_NAMES[source] || source,
        skills: groupMap.get(source) || [],
      }))
  }, [skills])

  // Stats
  const stats = useMemo(() => ({
    total: skills.length,
    enabled: skills.filter((s) => s.enabled).length,
    disabled: skills.filter((s) => !s.enabled).length,
    bundled: skills.filter((s) => s.source === 'bundled').length,
    global: skills.filter((s) => s.source === 'global').length,
    profile: skills.filter((s) => s.source === 'profile').length,
  }), [skills])

  // Toggle skill via IPC
  const toggleSkill = useCallback(async (skillId: string) => {
    try {
      const result = window.electronAPI
        ? await window.electronAPI.skills.toggle(skillId)
        : await window.ipcRenderer.invoke('skills:toggle', skillId)

      const typedResult = result as { error?: string; enabled?: boolean }
      if (typedResult.error) {
        setError(typedResult.error)
        return
      }

      // Update local state
      setSkills((prev) =>
        prev.map((skill) =>
          skill.id === skillId ? { ...skill, enabled: typedResult.enabled ?? !skill.enabled } : skill
        )
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to toggle skill')
    }
  }, [])

  // Enable skill via IPC
  const enableSkill = useCallback(async (skillId: string) => {
    try {
      const result = window.electronAPI
        ? await window.electronAPI.skills.setStatus(skillId, true)
        : await window.ipcRenderer.invoke('skills:setStatus', skillId, true)

      const typedResult = result as { error?: string }
      if (typedResult.error) {
        setError(typedResult.error)
        return
      }

      setSkills((prev) =>
        prev.map((skill) =>
          skill.id === skillId ? { ...skill, enabled: true } : skill
        )
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to enable skill')
    }
  }, [])

  // Disable skill via IPC
  const disableSkill = useCallback(async (skillId: string) => {
    try {
      const result = window.electronAPI
        ? await window.electronAPI.skills.setStatus(skillId, false)
        : await window.ipcRenderer.invoke('skills:setStatus', skillId, false)

      const typedResult = result as { error?: string }
      if (typedResult.error) {
        setError(typedResult.error)
        return
      }

      setSkills((prev) =>
        prev.map((skill) =>
          skill.id === skillId ? { ...skill, enabled: false } : skill
        )
      )
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to disable skill')
    }
  }, [])

  // Get skill by ID
  const getSkill = useCallback(
    (id: string): SkillInfo | undefined => {
      return skills.find((s) => s.id === id)
    },
    [skills]
  )

  // Filter skills by search query
  const filterSkills = useCallback(
    (query: string): SkillInfo[] => {
      if (!query.trim()) return skills

      const lowerQuery = query.toLowerCase()
      return skills.filter(
        (skill) =>
          skill.name.toLowerCase().includes(lowerQuery) ||
          skill.id.toLowerCase().includes(lowerQuery) ||
          skill.description.toLowerCase().includes(lowerQuery) ||
          skill.triggers.some((t) => t.toLowerCase().includes(lowerQuery))
      )
    },
    [skills]
  )

  // Check if skill is enabled
  const isSkillEnabled = useCallback(
    (skillId: string): boolean => {
      const skill = skills.find((s) => s.id === skillId)
      return skill?.enabled ?? false
    },
    [skills]
  )

  return {
    skills,
    groups,
    loading,
    error,
    toggleSkill,
    enableSkill,
    disableSkill,
    refresh: fetchSkills,
    getSkill,
    filterSkills,
    isSkillEnabled,
    stats,
  }
}

export default useSkills
