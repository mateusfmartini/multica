import { useState } from 'react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@multica/ui/components/ui/card'
import { Button } from '@multica/ui/components/ui/button'
import { Input } from '@multica/ui/components/ui/input'
import { Badge } from '@multica/ui/components/ui/badge'
import { useChannels, type UseChannelsReturn } from '../hooks/use-channels'

/** Status badge color mapping */
function statusVariant(status: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (status) {
    case 'running': return 'default'
    case 'starting': return 'secondary'
    case 'error': return 'destructive'
    default: return 'outline'
  }
}

function TelegramCard({ channels }: { channels: UseChannelsReturn }) {
  const { states, config, saveToken, removeToken, startChannel, stopChannel } = channels
  const [token, setToken] = useState('')
  const [saving, setSaving] = useState(false)
  const [localError, setLocalError] = useState<string | null>(null)

  // Current state and config for telegram:default
  const state = states.find((s) => s.channelId === 'telegram' && s.accountId === 'default')
  const savedConfig = config['telegram']?.['default'] as { botToken?: string } | undefined
  const hasToken = Boolean(savedConfig?.botToken)
  const isRunning = state?.status === 'running'
  const isStarting = state?.status === 'starting'

  const handleSave = async () => {
    if (!token.trim()) return
    setSaving(true)
    setLocalError(null)
    const result = await saveToken('telegram', 'default', token.trim())
    if (!result.ok) {
      setLocalError(result.error ?? 'Failed to save')
    } else {
      setToken('') // Clear input on success
    }
    setSaving(false)
  }

  const handleRemove = async () => {
    setSaving(true)
    setLocalError(null)
    const result = await removeToken('telegram', 'default')
    if (!result.ok) {
      setLocalError(result.error ?? 'Failed to remove')
    }
    setSaving(false)
  }

  const handleToggle = async () => {
    setSaving(true)
    setLocalError(null)
    if (isRunning || isStarting) {
      await stopChannel('telegram', 'default')
    } else {
      await startChannel('telegram', 'default')
    }
    setSaving(false)
  }

  // Mask the token for display: show first 5 and last 5 chars
  const maskedToken = savedConfig?.botToken
    ? `${savedConfig.botToken.slice(0, 5)}${'*'.repeat(10)}${savedConfig.botToken.slice(-5)}`
    : null

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>Telegram</CardTitle>
            <CardDescription>
              Connect a Telegram bot via Bot API long polling.
            </CardDescription>
          </div>
          {state && (
            <Badge variant={statusVariant(state.status)}>
              {state.status}
            </Badge>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {hasToken ? (
          // Token is configured — show masked token and actions
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <code className="text-sm text-muted-foreground bg-muted px-2 py-1 rounded flex-1 truncate">
                {maskedToken}
              </code>
            </div>

            {state?.error && (
              <p className="text-sm text-destructive">{state.error}</p>
            )}

            <div className="flex gap-2">
              <Button
                variant={isRunning ? 'outline' : 'default'}
                size="sm"
                onClick={handleToggle}
                disabled={saving}
              >
                {isRunning ? 'Stop' : isStarting ? 'Starting...' : 'Start'}
              </Button>
              <Button
                variant="destructive"
                size="sm"
                onClick={handleRemove}
                disabled={saving || isRunning}
                title={isRunning ? 'Stop the bot before removing' : undefined}
              >
                Remove
              </Button>
            </div>
          </div>
        ) : (
          // No token — show input form
          <div className="space-y-3">
            <Input
              type="password"
              placeholder="Bot Token (from @BotFather)"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSave()}
            />
            <Button
              size="sm"
              onClick={handleSave}
              disabled={saving || !token.trim()}
            >
              {saving ? 'Saving...' : 'Save & Connect'}
            </Button>
          </div>
        )}

        {localError && (
          <p className="text-sm text-destructive">{localError}</p>
        )}
      </CardContent>
    </Card>
  )
}

export default function ChannelsPage() {
  const channels = useChannels()
  const { loading, error } = channels

  return (
    <div className="max-w-4xl mx-auto space-y-4">
      <div>
        <h2 className="text-lg font-semibold">Channels</h2>
        <p className="text-sm text-muted-foreground">
          Connect messaging platforms to your Agent.
        </p>
      </div>

      {loading ? (
        <p className="text-sm text-muted-foreground">Loading...</p>
      ) : error ? (
        <p className="text-sm text-destructive">{error}</p>
      ) : (
        <TelegramCard channels={channels} />
      )}
    </div>
  )
}
