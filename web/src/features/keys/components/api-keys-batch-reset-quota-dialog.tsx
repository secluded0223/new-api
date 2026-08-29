/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import type { Table } from '@tanstack/react-table'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'

import { batchResetApiKeyQuota } from '../api'
import { ERROR_MESSAGES, SUCCESS_MESSAGES } from '../constants'
import type { ApiKey } from '../types'

export function ApiKeysBatchResetQuotaDialog<TData>({
  open,
  onOpenChange,
  table,
  triggerRefresh,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  table: Table<TData>
  triggerRefresh: () => void
}) {
  const { t } = useTranslation()
  const [isResetting, setIsResetting] = useState(false)
  const selectedRows = table.getFilteredSelectedRowModel().rows
  const limitedCount = selectedRows.filter(
    (row) => !(row.original as ApiKey).unlimited_quota
  ).length

  const handleReset = async () => {
    const ids = selectedRows
      .map((row) => row.original as ApiKey)
      .filter((apiKey) => !apiKey.unlimited_quota)
      .map((apiKey) => apiKey.id)
    if (ids.length === 0) return
    setIsResetting(true)
    try {
      const result = await batchResetApiKeyQuota(ids)
      if (result.success) {
        toast.success(t(SUCCESS_MESSAGES.API_KEY_QUOTA_RESET))
        table.resetRowSelection()
        onOpenChange(false)
        triggerRefresh()
      } else {
        toast.error(result.message || t(ERROR_MESSAGES.RESET_QUOTA_FAILED))
      }
    } catch {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    } finally {
      setIsResetting(false)
    }
  }

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t('Reset selected API key quotas?')}</AlertDialogTitle>
          <AlertDialogDescription>
            {t('This will reset {{count}} finite API key quota(s). Total consumption and usage history will not be changed.', { count: limitedCount })}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isResetting}>{t('Cancel')}</AlertDialogCancel>
          <AlertDialogAction onClick={handleReset} disabled={isResetting || limitedCount === 0}>
            {isResetting ? t('Resetting...') : t('Reset quota')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
