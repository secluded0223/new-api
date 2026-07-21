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
import { useMutation, useQuery } from '@tanstack/react-query'
import type { Table } from '@tanstack/react-table'
import { Check, Loader2, Shuffle } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { getUserGroups } from '@/lib/api'

import { batchUpdateApiKeyGroup } from '../api'
import { ERROR_MESSAGES } from '../constants'
import type { ApiKey } from '../types'
import { useApiKeys } from './api-keys-provider'

type ApiKeysGroupSwitcherProps = {
  table: Table<ApiKey>
}

export function ApiKeysGroupSwitcher(props: ApiKeysGroupSwitcherProps) {
  const { t } = useTranslation()
  const { triggerRefresh } = useApiKeys()
  const [open, setOpen] = useState(false)
  const selectedRows = props.table.getFilteredSelectedRowModel().rows
  const selectedCount = selectedRows.length

  const { data, isLoading } = useQuery({
    queryKey: ['user-groups'],
    queryFn: getUserGroups,
    enabled: open,
    staleTime: 30_000,
  })

  const groups = useMemo(
    () =>
      Object.entries(data?.data || {}).map(([value, info]) => ({
        value,
        description: info.desc,
      })),
    [data?.data]
  )

  const switchGroup = useMutation({
    mutationFn: ({ ids, group }: { ids: number[]; group: string }) =>
      batchUpdateApiKeyGroup(ids, group),
    onSuccess: (result, variables) => {
      if (!result.success) {
        toast.error(result.message || t(ERROR_MESSAGES.UPDATE_FAILED))
        return
      }

      toast.success(
        t('Switched {{count}} API key(s) to group {{group}}', {
          count: result.data ?? selectedCount,
          group: variables.group,
        })
      )
      props.table.resetRowSelection()
      setOpen(false)
      triggerRefresh()
    },
    onError: () => toast.error(t(ERROR_MESSAGES.UNEXPECTED)),
  })

  const handleSwitch = (group: string) => {
    if (selectedCount === 0 || switchGroup.isPending) return
    switchGroup.mutate({
      ids: selectedRows.map((row) => row.original.id),
      group,
    })
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            type='button'
            variant='outline'
            size='sm'
            disabled={selectedCount === 0 || switchGroup.isPending}
            aria-label={t('Switch group')}
          />
        }
      >
        {switchGroup.isPending ? (
          <Loader2 className='size-4 animate-spin' />
        ) : (
          <Shuffle className='size-4' />
        )}
        {t('Switch group')}
        {selectedCount > 0 && (
          <span className='text-muted-foreground tabular-nums'>
            ({selectedCount})
          </span>
        )}
      </PopoverTrigger>
      <PopoverContent align='end' className='w-72 p-0'>
        <Command>
          <CommandInput placeholder={t('Search groups...')} />
          <CommandList>
            <CommandEmpty>
              {isLoading ? t('Loading...') : t('No group found.')}
            </CommandEmpty>
            <CommandGroup heading={t('Select target group')}>
              {groups.map((group) => (
                <CommandItem
                  key={group.value}
                  value={`${group.value} ${group.description}`}
                  onSelect={() => handleSwitch(group.value)}
                  disabled={switchGroup.isPending}
                >
                  <Check className='size-4 opacity-0' />
                  <span className='min-w-0 flex-1'>
                    <span className='block truncate font-medium'>
                      {group.value}
                    </span>
                    <span className='text-muted-foreground block truncate text-xs'>
                      {group.description}
                    </span>
                  </span>
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
