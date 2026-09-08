import { createFileRoute } from '@tanstack/react-router'

import { TokenRanking } from '@/features/token-ranking'

export const Route = createFileRoute('/_authenticated/token-ranking/')({
  component: TokenRanking,
})
