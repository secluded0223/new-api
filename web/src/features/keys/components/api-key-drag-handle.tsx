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
import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { GripVertical } from 'lucide-react'
import { useTranslation } from 'react-i18next'

export function ApiKeyDragHandle({ id }: { id: number }) {
  const { t } = useTranslation()
  const { attributes, listeners, setNodeRef, transform, transition } =
    useSortable({ id: String(id) })

  return (
    <button
      ref={setNodeRef}
      type='button'
      aria-label={t('Drag to reorder')}
      className='text-muted-foreground hover:text-foreground flex cursor-grab touch-none items-center justify-center active:cursor-grabbing'
      style={{
        transform: CSS.Transform.toString(transform),
        transition,
      }}
      {...attributes}
      {...listeners}
    >
      <GripVertical className='size-4' aria-hidden='true' />
    </button>
  )
}
