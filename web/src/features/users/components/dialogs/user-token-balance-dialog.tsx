import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import type { ApiKey } from '@/features/keys/types'
import { formatQuota, parseQuotaFromDollars } from '@/lib/format'

import {
  allocateAdminUserTokenQuota,
  getAdminUserTokens,
  type TokenQuotaAllocation,
} from '../../api'
import type { User } from '../../types'

interface UserTokenBalanceDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: User
  onSuccess?: () => void
}

type AllocationMode = 'equal' | 'manual'

export function UserTokenBalanceDialog(props: UserTokenBalanceDialogProps) {
  const { t } = useTranslation()
  const [tokens, setTokens] = useState<ApiKey[]>([])
  const [selectedIds, setSelectedIds] = useState<number[]>([])
  const [manualValues, setManualValues] = useState<Record<number, string>>({})
  const [mode, setMode] = useState<AllocationMode>('equal')
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)

  const loadTokens = useCallback(async () => {
    setLoading(true)
    try {
      const result = await getAdminUserTokens(props.user.id)
      if (result.success) setTokens(result.data ?? [])
      else toast.error(result.message || t('Failed to load API keys'))
    } catch {
      toast.error(t('An unexpected error occurred'))
    } finally {
      setLoading(false)
    }
  }, [props.user.id, t])

  useEffect(() => {
    if (!props.open) return
    setSelectedIds([])
    setManualValues({})
    setMode('equal')
    void loadTokens()
  }, [props.open, loadTokens])

  const finiteTokens = useMemo(
    () => tokens.filter((token) => !token.unlimited_quota),
    [tokens]
  )
  const userTotalQuota = props.user.quota + props.user.used_quota
  const keyUsageSummary = useMemo(
    () =>
      tokens.reduce(
        (summary, token) => ({
          used: summary.used + token.used_quota,
        }),
        { used: 0 }
      ),
    [tokens]
  )
  // User quota is the current wallet balance. The missing key record is the
  // user's consumed amount minus usage still present on existing keys.
  const difference = userTotalQuota - props.user.quota - keyUsageSummary.used

  const allocations = useMemo<TokenQuotaAllocation[]>(() => {
    if (difference <= 0 || selectedIds.length === 0) return []
    if (mode === 'manual') {
      return selectedIds.map((tokenId) => ({
        token_id: tokenId,
        quota: parseQuotaFromDollars(Number(manualValues[tokenId] ?? 0)),
      }))
    }
    const base = Math.floor(difference / selectedIds.length)
    const remainder = difference - base * selectedIds.length
    return selectedIds.map((tokenId, index) => ({
      token_id: tokenId,
      quota: base + (index === selectedIds.length - 1 ? remainder : 0),
    }))
  }, [difference, manualValues, mode, selectedIds])

  const allocationTotal = allocations.reduce(
    (total, allocation) => total + allocation.quota,
    0
  )
  const canSubmit =
    !loading &&
    !saving &&
    difference > 0 &&
    allocations.length > 0 &&
    allocations.every(
      (allocation) =>
        Number.isSafeInteger(allocation.quota) && allocation.quota > 0
    ) &&
    allocationTotal <= difference

  const toggleToken = (tokenId: number, checked: boolean) => {
    setSelectedIds((current) =>
      checked
        ? [...current, tokenId]
        : current.filter((selectedId) => selectedId !== tokenId)
    )
  }

  let allocationContent: ReactNode
  if (difference <= 0) {
    allocationContent = (
      <p className='text-muted-foreground py-6 text-center text-sm'>
        {t('No positive difference to allocate')}
      </p>
    )
  } else if (finiteTokens.length === 0) {
    allocationContent = (
      <p className='text-muted-foreground py-6 text-center text-sm'>
        {t('No finite API keys found for allocation')}
      </p>
    )
  } else {
    allocationContent = (
      <div className='space-y-4'>
        <div>
          <div className='text-muted-foreground mb-2 text-sm'>
            {t('Allocation mode')}
          </div>
          <div className='flex gap-2'>
            <Button
              type='button'
              size='sm'
              variant={mode === 'equal' ? 'default' : 'outline'}
              onClick={() => setMode('equal')}
            >
              {t('Split evenly')}
            </Button>
            <Button
              type='button'
              size='sm'
              variant={mode === 'manual' ? 'default' : 'outline'}
              onClick={() => setMode('manual')}
            >
              {t('Manual allocation')}
            </Button>
          </div>
        </div>
        <div className='space-y-2'>
          <div className='text-muted-foreground text-sm'>
            {t('Select API keys')}
          </div>
          {finiteTokens.map((token) => {
            const selected = selectedIds.includes(token.id)
            const allocation = allocations.find(
              (item) => item.token_id === token.id
            )
            let allocationValue: ReactNode = null
            if (mode === 'manual' && selected) {
              allocationValue = (
                <input
                  type='number'
                  min={1}
                  value={manualValues[token.id] ?? ''}
                  onChange={(event) =>
                    setManualValues((current) => ({
                      ...current,
                      [token.id]: event.target.value,
                    }))
                  }
                  aria-label={t('Amount to allocate')}
                  className='border-input bg-background h-9 w-36 rounded-md border px-3 text-sm'
                />
              )
            } else if (selected) {
              allocationValue = (
                <span className='text-sm font-medium'>
                  {formatQuota(allocation?.quota ?? 0)}
                </span>
              )
            }
            return (
              <div
                key={token.id}
                className='flex flex-wrap items-center gap-3 rounded-md border p-3'
              >
                <Checkbox
                  checked={selected}
                  onCheckedChange={(value) =>
                    toggleToken(token.id, value === true)
                  }
                  aria-label={token.name}
                />
                <div className='min-w-0 flex-1 space-y-2'>
                  <div className='font-medium'>{token.name}</div>
                  <div className='grid gap-x-4 gap-y-1 text-xs sm:grid-cols-3'>
                    <div>
                      <span className='text-muted-foreground'>
                        {t('Current quota')}:
                      </span>{' '}
                      <span className='font-medium'>
                        {formatQuota(token.quota)}
                      </span>
                    </div>
                    <div>
                      <span className='text-muted-foreground'>
                        {t('Used quota')}:
                      </span>{' '}
                      <span className='font-medium'>
                        {formatQuota(token.used_quota)}
                      </span>
                    </div>
                    <div>
                      <span className='text-muted-foreground'>
                        {t('Remaining quota')}:
                      </span>{' '}
                      <span className='font-medium'>
                        {formatQuota(token.remain_quota)}
                      </span>
                    </div>
                  </div>
                </div>
                {allocationValue}
              </div>
            )
          })}
        </div>
        {selectedIds.length > 0 && allocationTotal > difference && (
          <p className='text-destructive text-sm'>
            {t('Allocation cannot exceed the difference')}
          </p>
        )}
      </div>
    )
  }

  const handleAllocate = async () => {
    if (!canSubmit) return
    setSaving(true)
    try {
      const result = await allocateAdminUserTokenQuota(
        props.user.id,
        allocations
      )
      if (!result.success) {
        toast.error(result.message || t('An unexpected error occurred'))
        return
      }
      toast.success(t('Quota allocation completed successfully'))
      props.onOpenChange(false)
      props.onSuccess?.()
    } catch {
      toast.error(t('An unexpected error occurred'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={t('Reconcile quota')}
      description={t('Reconcile API key quota for {{username}}', {
        username: props.user.username,
      })}
      contentClassName='sm:max-w-2xl'
      footer={
        <>
          <Button variant='outline' onClick={() => props.onOpenChange(false)}>
            {t('Cancel')}
          </Button>
          <Button onClick={() => void handleAllocate()} disabled={!canSubmit}>
            {t('Allocate quota')}
          </Button>
        </>
      }
    >
      <div className='grid gap-2 rounded-md border p-3 text-sm sm:grid-cols-4'>
        <div>
          <div className='text-muted-foreground'>{t('User total quota')}</div>
          <div className='font-medium'>{formatQuota(userTotalQuota)}</div>
        </div>
        <div>
          <div className='text-muted-foreground'>
            {t('User remaining quota')}
          </div>
          <div className='font-medium'>{formatQuota(props.user.quota)}</div>
        </div>
        <div>
          <div className='text-muted-foreground'>{t('Key used quota')}</div>
          <div className='font-medium'>{formatQuota(keyUsageSummary.used)}</div>
        </div>
        <div>
          <div className='text-muted-foreground'>{t('Difference')}</div>
          <div
            className={
              difference > 0 ? 'text-primary font-medium' : 'font-medium'
            }
          >
            {formatQuota(difference)}
          </div>
        </div>
      </div>

      {allocationContent}
    </Dialog>
  )
}
