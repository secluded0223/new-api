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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { getApiKeyQuotaSummary } from '../api-key-quota.ts'

describe('API key quota summary', () => {
  test('separates total, used, and remaining values for limited keys', () => {
    const summary = getApiKeyQuotaSummary({
      quota: 1000,
      remain_quota: 948.5,
      unlimited_quota: false,
      used_quota: 51.5,
    })

    assert.deepEqual(summary, {
      isUnlimited: false,
      total: 1000,
      used: 51.5,
      remaining: 948.5,
      remainingPercentage: 94.85,
    })
  })

  test('exposes only used quota as numeric data for unlimited keys', () => {
    const summary = getApiKeyQuotaSummary({
      quota: 0,
      remain_quota: 0,
      unlimited_quota: true,
      used_quota: 51.5,
    })

    assert.deepEqual(summary, {
      isUnlimited: true,
      total: null,
      used: 51.5,
      remaining: null,
      remainingPercentage: null,
    })
  })
})
