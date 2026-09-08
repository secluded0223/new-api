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
import { Download, Loader2, Plus } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { formatTimestampToDate, formatQuota } from '@/lib/format'

import { getAllApiKeys } from '../api'
import { buildApiKeysCsv } from '../lib/api-key-export'
import { useApiKeys } from './api-keys-provider'

export function ApiKeysPrimaryButtons() {
  const { t } = useTranslation()
  const { setOpen, resolveRealKeysBatch, selectedApiKeyIds } = useApiKeys()
  const [isExporting, setIsExporting] = useState(false)

  const handleExport = async () => {
    if (selectedApiKeyIds.length === 0) return
    setIsExporting(true)
    try {
      const apiKeys = (await getAllApiKeys()).filter((apiKey) =>
        selectedApiKeyIds.includes(apiKey.id)
      )
      const resolvedKeys: Record<number, string> = {}
      for (let index = 0; index < apiKeys.length; index += 100) {
        const chunk = apiKeys.slice(index, index + 100)
        Object.assign(
          resolvedKeys,
          await resolveRealKeysBatch(chunk.map((apiKey) => apiKey.id))
        )
      }

      if (apiKeys.some((apiKey) => !resolvedKeys[apiKey.id])) {
        throw new Error('Missing API key')
      }

      const csv = buildApiKeysCsv(
        apiKeys,
        resolvedKeys,
        {
          name: t('Name'),
          key: t('Key'),
          used: t('Used'),
          totalConsumption: t('Total consumption'),
          created: t('Created'),
        },
        formatQuota,
        formatTimestampToDate
      )
      const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `api-keys-${new Date().toISOString().slice(0, 10)}.csv`
      link.click()
      URL.revokeObjectURL(url)
      toast.success(t('API keys exported successfully'))
    } catch {
      toast.error(t('Failed to export API keys'))
    } finally {
      setIsExporting(false)
    }
  }

  return (
    <div className='flex gap-2'>
      <Button
        size='sm'
        variant='outline'
        onClick={() => void handleExport()}
        disabled={isExporting || selectedApiKeyIds.length === 0}
      >
        {isExporting ? (
          <Loader2 className='h-4 w-4 animate-spin' />
        ) : (
          <Download className='h-4 w-4' />
        )}
        {t('Export')}
      </Button>
      <Button size='sm' onClick={() => setOpen('create')}>
        <Plus className='h-4 w-4' />
        {t('Create API Key')}
      </Button>
    </div>
  )
}
