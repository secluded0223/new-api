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
import type { ApiKey } from '../types'

export type ApiKeyQuotaSummary = {
  isUnlimited: boolean
  total: number | null
  used: number
  remaining: number | null
  remainingPercentage: number | null
}

export function getApiKeyQuotaSummary(
  apiKey: Pick<ApiKey, 'quota' | 'remain_quota' | 'unlimited_quota' | 'used_quota'>
): ApiKeyQuotaSummary {
  if (apiKey.unlimited_quota) {
    return {
      isUnlimited: true,
      total: null,
      used: apiKey.used_quota,
      remaining: null,
      remainingPercentage: null,
    }
  }

  const total = apiKey.quota || apiKey.used_quota + apiKey.remain_quota

  return {
    isUnlimited: false,
    total,
    used: apiKey.used_quota,
    remaining: apiKey.remain_quota,
    remainingPercentage: total > 0 ? (apiKey.remain_quota / total) * 100 : 0,
  }
}
