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
import { RotateCcw } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import type { ApiKey } from '../types'
import { ApiKeysBatchResetQuotaDialog } from './api-keys-batch-reset-quota-dialog'
import { useApiKeys } from './api-keys-provider'

export function ApiKeysResetQuotaButton({
  table,
}: {
  table: Table<ApiKey>
}) {
  const { t } = useTranslation()
  const { triggerRefresh } = useApiKeys()
  const [open, setOpen] = useState(false)
  const selectedRows = table.getFilteredSelectedRowModel().rows
  const selectedCount = selectedRows.length

  return (
    <>
      <Button
        type='button'
        variant='outline'
        size='sm'
        disabled={selectedCount === 0}
        onClick={() => setOpen(true)}
        aria-label={t('Reset quota')}
      >
        <RotateCcw className='size-4' />
        {t('Reset quota')}
        {selectedCount > 0 && (
          <span className='text-muted-foreground tabular-nums'>
            ({selectedCount})
          </span>
        )}
      </Button>
      <ApiKeysBatchResetQuotaDialog
        open={open}
        onOpenChange={setOpen}
        table={table}
        triggerRefresh={triggerRefresh}
      />
    </>
  )
}
