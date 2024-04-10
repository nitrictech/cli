import type { Endpoint } from '../../types'
import { LOCAL_STORAGE_KEY } from './APIExplorer'
import { useHistory } from '../../lib/hooks/use-history'
import { formatJSON } from '@/lib/utils'
import {
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
} from '../ui/dropdown-menu'
import TrashIcon from '@heroicons/react/24/outline/TrashIcon'
import { ArrowDownOnSquareIcon } from '@heroicons/react/24/outline'
import ResourceDropdownMenu from '../shared/ResourceDropdownMenu'

interface Props {
  selected: Endpoint
  onAfterClear: () => void
}

const APIMenu: React.FC<Props> = ({ selected, onAfterClear }) => {
  const { deleteHistory } = useHistory('apis')
  const clearHistory = async () => {
    const prefix = `${LOCAL_STORAGE_KEY}-${selected.api}-`

    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i)
      if (key?.startsWith(prefix)) {
        localStorage.removeItem(key)
      }
    }

    localStorage.removeItem(`${LOCAL_STORAGE_KEY}-requests`)

    await deleteHistory()

    onAfterClear()
  }

  const downloadSpec = () => {
    const json = formatJSON(selected.doc)
    const blob = new Blob([json], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${selected.api}-spec.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  return (
    <ResourceDropdownMenu>
      <DropdownMenuLabel>API Menu</DropdownMenuLabel>
      <DropdownMenuSeparator />
      <DropdownMenuGroup>
        <DropdownMenuItem onClick={downloadSpec}>
          <ArrowDownOnSquareIcon className="mr-2 h-4 w-4" />
          <span>Export Spec</span>
        </DropdownMenuItem>
        <DropdownMenuItem onClick={clearHistory}>
          <TrashIcon className="mr-2 h-4 w-4" />
          <span>Clear History</span>
        </DropdownMenuItem>
      </DropdownMenuGroup>
    </ResourceDropdownMenu>
  )
}

export default APIMenu
