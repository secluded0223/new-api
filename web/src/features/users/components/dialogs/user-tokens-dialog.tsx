import { Pencil, RefreshCw } from 'lucide-react'
import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Dialog } from '@/components/dialog'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { API_KEY_STATUS, API_KEY_STATUSES } from '@/features/keys/constants'
import type { ApiKey } from '@/features/keys/types'
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import { formatQuota } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import {
  getAdminUserTokens,
  mergeAdminUserTokenConsumption,
  updateAdminUserTokenUsedQuota,
} from '../../api'

interface UserTokensDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  userId: number | null
  username: string
  onSuccess?: () => void
}

export function UserTokensDialog(props: UserTokensDialogProps) {
  const { t } = useTranslation()
  const currentUser = useAuthStore((s) => s.auth.user)
  const canMerge = hasPermission(
    currentUser,
    ADMIN_PERMISSION_RESOURCES.TOKEN,
    ADMIN_PERMISSION_ACTIONS.MERGE
  )
  const isRoot = currentUser?.role === ROLE.SUPER_ADMIN
  const [tokens, setTokens] = useState<ApiKey[]>([])
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [sourceId, setSourceId] = useState('')
  const [targetId, setTargetId] = useState('')
  const [mergeOpen, setMergeOpen] = useState(false)
  const [editingUsedQuota, setEditingUsedQuota] = useState<ApiKey | null>(null)
  const [usedQuotaValue, setUsedQuotaValue] = useState('0')

  const loadTokens = useCallback(async () => {
    if (!props.userId) return
    setLoading(true)
    try {
      const result = await getAdminUserTokens(props.userId)
      if (result.success) setTokens(result.data ?? [])
      else toast.error(result.message || t('Failed to load API keys'))
    } catch {
      toast.error(t('An unexpected error occurred'))
    } finally {
      setLoading(false)
    }
  }, [props.userId, t])

  useEffect(() => {
    if (!props.open) return
    setSourceId('')
    setTargetId('')
    void loadTokens()
  }, [props.open, loadTokens])

  const selectedSource = useMemo(
    () => tokens.find((token) => String(token.id) === sourceId),
    [sourceId, tokens]
  )
  const selectedTarget = useMemo(
    () => tokens.find((token) => String(token.id) === targetId),
    [targetId, tokens]
  )

  const mergeTokens = async () => {
    if (!props.userId || !selectedSource || !selectedTarget) return
    setSaving(true)
    try {
      const result = await mergeAdminUserTokenConsumption(
        props.userId,
        selectedSource.id,
        selectedTarget.id
      )
      if (!result.success) {
        toast.error(result.message || t('An unexpected error occurred'))
        return
      }
      toast.success(t('API key consumption merged successfully'))
      setMergeOpen(false)
      setSourceId('')
      setTargetId('')
      await loadTokens()
      props.onSuccess?.()
    } catch {
      toast.error(t('An unexpected error occurred'))
    } finally {
      setSaving(false)
    }
  }

  const saveUsedQuota = async () => {
    if (!props.userId || !editingUsedQuota) return
    const usedQuota = Number(usedQuotaValue)
    if (
      !Number.isInteger(usedQuota) ||
      usedQuota < 0 ||
      usedQuota > editingUsedQuota.quota
    ) {
      toast.error(t('Invalid quota'))
      return
    }
    setSaving(true)
    try {
      const result = await updateAdminUserTokenUsedQuota(
        props.userId,
        editingUsedQuota.id,
        usedQuota
      )
      if (!result.success) {
        toast.error(result.message || t('An unexpected error occurred'))
        return
      }
      toast.success(t('Used quota updated successfully'))
      setEditingUsedQuota(null)
      await loadTokens()
      props.onSuccess?.()
    } catch {
      toast.error(t('An unexpected error occurred'))
    } finally {
      setSaving(false)
    }
  }

  let tokenContent: ReactNode
  if (loading) {
    tokenContent = (
      <div className='text-muted-foreground py-8 text-center'>
        {t('Loading...')}
      </div>
    )
  } else if (tokens.length === 0) {
    tokenContent = (
      <div className='text-muted-foreground py-8 text-center'>
        {t('No API keys found for this user')}
      </div>
    )
  } else {
    tokenContent = (
      <div className='divide-border overflow-hidden rounded-md border'>
        {tokens.map((token) => {
          const status =
            API_KEY_STATUSES[token.status] ??
            API_KEY_STATUSES[API_KEY_STATUS.DISABLED]
          return (
            <div
              key={token.id}
              className='flex flex-wrap items-center gap-3 border-b p-3 last:border-b-0'
            >
              <div className='min-w-[140px] flex-1'>
                <div className='font-medium'>{token.name}</div>
                <div className='text-muted-foreground text-xs'>{token.key}</div>
              </div>
              <StatusBadge
                label={t(status.label)}
                variant={status.variant}
                copyable={false}
              />
              <div className='text-muted-foreground text-xs'>
                {t('Quota')}:{' '}
                <span className='text-foreground'>
                  {formatQuota(token.quota)}
                </span>
              </div>
              <div className='text-muted-foreground text-xs'>
                {t('Used quota')}:{' '}
                <span className='text-foreground'>
                  {formatQuota(token.used_quota)}
                </span>
              </div>
              {isRoot && (
                <Button
                  variant='ghost'
                  size='icon-sm'
                  aria-label={t('Edit used quota')}
                  onClick={() => {
                    setEditingUsedQuota(token)
                    setUsedQuotaValue(String(token.used_quota ?? 0))
                  }}
                >
                  <Pencil className='size-4' />
                </Button>
              )}
              <div className='text-muted-foreground text-xs'>
                {t('Remaining quota')}:{' '}
                <span className='text-foreground'>
                  {formatQuota(token.remain_quota)}
                </span>
              </div>
              <div className='text-muted-foreground text-xs'>
                {t('Total consumption')}:{' '}
                <span className='text-foreground'>
                  {formatQuota(token.total_used_quota)}
                </span>
              </div>
            </div>
          )
        })}
      </div>
    )
  }

  return (
    <>
      <Dialog
        open={props.open}
        onOpenChange={props.onOpenChange}
        title={t('Manage API Keys')}
        description={t('Manage API keys for {{username}}', {
          username: props.username,
        })}
        contentClassName='sm:max-w-3xl'
        footer={
          <Button
            variant='outline'
            onClick={() => void loadTokens()}
            disabled={loading}
          >
            <RefreshCw className='size-4' />
            {t('Refresh')}
          </Button>
        }
      >
        {canMerge && tokens.length > 1 && (
          <div className='mb-4 flex flex-wrap items-end gap-2 rounded-md border p-3'>
            <div className='min-w-[180px] flex-1'>
              <label
                htmlFor='admin-token-source'
                className='text-muted-foreground mb-1 block text-xs'
              >
                {t('Source key')}
              </label>
              <select
                id='admin-token-source'
                className='border-input bg-background h-9 w-full rounded-md border px-2 text-sm'
                value={sourceId}
                onChange={(event) => setSourceId(event.target.value)}
              >
                <option value=''>{t('Select a key')}</option>
                {tokens
                  .filter((token) => token.id !== Number(targetId))
                  .map((token) => (
                    <option key={token.id} value={token.id}>
                      {token.name}
                    </option>
                  ))}
              </select>
            </div>
            <div className='min-w-[180px] flex-1'>
              <label
                htmlFor='admin-token-target'
                className='text-muted-foreground mb-1 block text-xs'
              >
                {t('Target key')}
              </label>
              <select
                id='admin-token-target'
                className='border-input bg-background h-9 w-full rounded-md border px-2 text-sm'
                value={targetId}
                onChange={(event) => setTargetId(event.target.value)}
              >
                <option value=''>{t('Select a key')}</option>
                {tokens
                  .filter((token) => token.id !== Number(sourceId))
                  .map((token) => (
                    <option key={token.id} value={token.id}>
                      {token.name}
                    </option>
                  ))}
              </select>
            </div>
            <Button
              variant='secondary'
              disabled={!selectedSource || !selectedTarget || saving}
              onClick={() => setMergeOpen(true)}
            >
              {t('Merge consumption')}
            </Button>
          </div>
        )}

        {isRoot && editingUsedQuota && (
          <div className='mb-4 flex flex-wrap items-end gap-2 rounded-md border p-3'>
            <div className='flex-1'>
              <label
                htmlFor='admin-token-used-quota'
                className='text-muted-foreground mb-1 block text-xs'
              >
                {t('Used quota')}
              </label>
              <input
                id='admin-token-used-quota'
                type='number'
                min={0}
                max={editingUsedQuota.quota}
                value={usedQuotaValue}
                onChange={(event) => setUsedQuotaValue(event.target.value)}
                className='border-input bg-background h-9 w-full rounded-md border px-3 text-sm'
              />
            </div>
            <Button onClick={() => void saveUsedQuota()} disabled={saving}>
              {t('Save')}
            </Button>
            <Button variant='ghost' onClick={() => setEditingUsedQuota(null)}>
              {t('Cancel')}
            </Button>
          </div>
        )}

        {tokenContent}
      </Dialog>

      <ConfirmDialog
        open={mergeOpen}
        onOpenChange={setMergeOpen}
        title={t('Merge consumption')}
        desc={t(
          'Move cumulative consumption from {{source}} to {{target}}. Quota and remaining balance will not change. The source key will be disabled.',
          {
            source: selectedSource?.name ?? '',
            target: selectedTarget?.name ?? '',
          }
        )}
        confirmText={t('Confirm merge')}
        handleConfirm={() => void mergeTokens()}
        isLoading={saving}
      />
    </>
  )
}
