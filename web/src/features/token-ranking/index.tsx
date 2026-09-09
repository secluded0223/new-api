import { useQuery } from '@tanstack/react-query'
import { useMemo, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatNumber, formatQuota } from '@/lib/format'
import { api } from '@/lib/api'

type RankingPeriod = 'daily' | 'weekly' | 'monthly' | 'total'
type RankingSort = 'tokens' | 'quota' | 'requests'

type TokenUsageRankingItem = {
  token_id: number
  token_name: string
  tokens: number
  quota: number
  requests: number
}

type TokenUsageRanking = Record<RankingPeriod, TokenUsageRankingItem[]>

async function fetchTokenRanking(): Promise<TokenUsageRanking> {
  const response = await api.get('/api/token/ranking')
  if (!response.data?.success || !response.data?.data) {
    throw new Error(response.data?.message || 'Failed to load token rankings')
  }
  return response.data.data as TokenUsageRanking
}

const periods: { value: RankingPeriod; label: string }[] = [
  { value: 'daily', label: 'Today' },
  { value: 'weekly', label: 'Last 7 days' },
  { value: 'monthly', label: 'This month' },
  { value: 'total', label: 'Total' },
]

const sortOptions: { value: RankingSort; label: string }[] = [
  { value: 'tokens', label: 'Tokens used' },
  { value: 'quota', label: 'Cost' },
  { value: 'requests', label: 'Requests' },
]

export function TokenRanking() {
  const { t } = useTranslation()
  const [period, setPeriod] = useState<RankingPeriod>('daily')
  const [sortBy, setSortBy] = useState<RankingSort>('quota')
  const rankingQuery = useQuery({
    queryKey: ['token-ranking'],
    queryFn: fetchTokenRanking,
  })

  const rows = rankingQuery.data?.[period]
  const sortedRows = useMemo(
    () =>
      [...(rows ?? [])].sort((left, right) => {
        const difference = right[sortBy] - left[sortBy]
        if (difference !== 0) return difference
        if (sortBy !== 'quota') {
          const quotaDifference = right.quota - left.quota
          if (quotaDifference !== 0) return quotaDifference
        }
        return left.token_id - right.token_id
      }),
    [rows, sortBy]
  )
  let rankingContent: ReactNode
  if (rankingQuery.isLoading) {
    rankingContent = (
      <div className='space-y-3 p-4'>
        <Skeleton className='h-10 w-full' />
        <Skeleton className='h-10 w-full' />
        <Skeleton className='h-10 w-full' />
      </div>
    )
  } else if (rankingQuery.isError) {
    rankingContent = (
      <div className='text-muted-foreground px-4 py-12 text-center text-sm'>
        {t('Unable to load token rankings')}
      </div>
    )
  } else if (sortedRows.length === 0) {
    rankingContent = (
      <div className='text-muted-foreground px-4 py-12 text-center text-sm'>
        {t('No token usage data')}
      </div>
    )
  } else {
    rankingContent = (
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className='w-20'>{t('Rank')}</TableHead>
            <TableHead>{t('API Key')}</TableHead>
            <TableHead className='text-right'>{t('Tokens used')}</TableHead>
            <TableHead className='text-right'>{t('Cost')}</TableHead>
            <TableHead className='text-right'>{t('Requests')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {sortedRows.map((row, index) => (
            <TableRow key={row.token_id}>
              <TableCell className='font-semibold tabular-nums'>
                {index + 1}
              </TableCell>
              <TableCell className='max-w-[min(50vw,28rem)] truncate font-medium'>
                {row.token_name ||
                  t('Deleted key ({{id}})', { id: row.token_id })}
              </TableCell>
              <TableCell className='text-right font-mono tabular-nums'>
                {formatNumber(row.tokens)}
              </TableCell>
              <TableCell className='text-right font-mono tabular-nums'>
                {formatQuota(row.quota)}
              </TableCell>
              <TableCell className='text-right font-mono tabular-nums'>
                {formatNumber(row.requests)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    )
  }

  return (
    <SectionPageLayout fixedContent>
      <SectionPageLayout.Title>{t('Token Rankings')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <div className='flex flex-wrap justify-end gap-2'>
          <Tabs
            value={period}
            onValueChange={(value) => setPeriod(value as RankingPeriod)}
          >
            <TabsList>
              {periods.map((item) => (
                <TabsTrigger key={item.value} value={item.value}>
                  {t(item.label)}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
          <Tabs
            value={sortBy}
            onValueChange={(value) => setSortBy(value as RankingSort)}
          >
            <TabsList>
              {sortOptions.map((item) => (
                <TabsTrigger key={item.value} value={item.value}>
                  {t(item.label)}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        </div>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='bg-card h-full overflow-auto rounded-lg border'>
          {rankingContent}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
