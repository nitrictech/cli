import { TrashIcon } from '@heroicons/react/20/solid'

import { useHistory } from '../../lib/hooks/use-history'
import ResourceDropdownMenu from '../shared/ResourceDropdownMenu'
import {
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
} from '../ui/dropdown-menu'

interface Props {
  storageKey: string
  workerType: string
  selected: string
  onAfterClear: () => void
}

const EventsMenu: React.FC<Props> = ({
  workerType,
  storageKey,
  selected,
  onAfterClear,
}) => {
  const { deleteHistory } = useHistory(workerType)

  const clearHistory = async () => {
    const prefix = `${storageKey}-${selected}-`

    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i)
      if (key?.startsWith(prefix)) {
        localStorage.removeItem(key)
      }
    }

    localStorage.removeItem(`${storageKey}-requests`)

    await deleteHistory()

    onAfterClear()
  }

  return (
    <ResourceDropdownMenu>
      <DropdownMenuLabel className="capitalize text-foreground">
        {workerType} Menu
      </DropdownMenuLabel>
      <DropdownMenuSeparator />
      <DropdownMenuGroup>
        <DropdownMenuItem onClick={clearHistory} className="text-foreground">
          <TrashIcon className="mr-2 h-4 w-4" />
          <span>Clear History</span>
        </DropdownMenuItem>
      </DropdownMenuGroup>
    </ResourceDropdownMenu>
  )
}

export default EventsMenu
