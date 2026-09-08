import { useQuery } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
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
import { formatNumber } from '@/lib/format'
import { api } from '@/lib/api'

type RankingPeriod = 'daily' | 'weekly' | 'monthly' | 'total'

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

export function TokenRanking() {
  const { t } = useTranslation()
  const [period, setPeriod] = useState<RankingPeriod>('daily')
  const rankingQuery = useQuery({
    queryKey: ['token-ranking'],
    queryFn: fetchTokenRanking,
  })

  const rows = rankingQuery.data?.[period] ?? []
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
  } else if (rows.length === 0) {
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
            <TableHead className='text-right'>{t('Requests')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row, index) => (
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
        <Tabs value={period} onValueChange={(value) => setPeriod(value as RankingPeriod)}>
          <TabsList>
            {periods.map((item) => (
              <TabsTrigger key={item.value} value={item.value}>
                {t(item.label)}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='bg-card h-full overflow-auto rounded-lg border'>
          {rankingContent}
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
