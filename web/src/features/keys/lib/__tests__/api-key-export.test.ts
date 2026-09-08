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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { buildApiKeysCsv } from '../api-key-export.ts'

describe('API key CSV export', () => {
  test('writes the requested fields and escapes CSV values', () => {
    const csv = buildApiKeysCsv(
      [
        {
          id: 7,
          name: 'Primary, key',
          key: 'sk-********',
          status: 1,
          remain_quota: 1,
          quota: 2,
          used_quota: 1,
          total_used_quota: 3,
          unlimited_quota: false,
          expired_time: -1,
          created_time: 1710000000,
          accessed_time: 0,
          group: '',
          cross_group_retry: false,
          model_limits_enabled: false,
          model_limits: '',
          allow_ips: '',
        },
      ],
      { 7: 'sk-real-key' },
      {
        name: 'Name',
        key: 'Key',
        used: 'Used',
        totalConsumption: 'Total consumption',
        created: 'Created',
      },
      (quota) => `quota:${quota}`,
      (timestamp) => `time:${timestamp}`
    )

    assert.equal(
      csv,
      '\uFEFFName,Key,Used,Total consumption,Created\r\n"Primary, key",sk-real-key,quota:1,quota:3,time:1710000000\r\n'
    )
  })
})
