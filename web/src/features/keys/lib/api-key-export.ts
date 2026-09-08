/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import type { ApiKey } from '../types'

export type ApiKeyExportLabels = {
  name: string
  key: string
  used: string
  totalConsumption: string
  created: string
}

function escapeCsvValue(value: string | number): string {
  const text = String(value)
  return /[",\r\n]/.test(text) ? `"${text.replaceAll('"', '""')}"` : text
}

export function buildApiKeysCsv(
  apiKeys: ApiKey[],
  resolvedKeys: Record<number, string>,
  labels: ApiKeyExportLabels,
  formatQuotaValue: (quota: number) => string,
  formatCreatedTime: (timestamp: number) => string
): string {
  const rows = [
    [
      labels.name,
      labels.key,
      labels.used,
      labels.totalConsumption,
      labels.created,
    ],
    ...apiKeys.map((apiKey) => [
      apiKey.name,
      resolvedKeys[apiKey.id] ?? '',
      formatQuotaValue(apiKey.used_quota),
      formatQuotaValue(apiKey.total_used_quota),
      formatCreatedTime(apiKey.created_time),
    ]),
  ]

  return `\uFEFF${rows.map((row) => row.map(escapeCsvValue).join(',')).join('\r\n')}\r\n`
}
