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
  selected: string
  onAfterClear: () => void
}

const DatabasesMenu: React.FC<Props> = ({
  storageKey,
  selected,
  onAfterClear,
}) => {
  const clearHistory = async () => {
    localStorage.removeItem(storageKey)

    onAfterClear()
  }

  return (
    <ResourceDropdownMenu>
      <DropdownMenuLabel className="capitalize">
        Database Menu
      </DropdownMenuLabel>
      <DropdownMenuSeparator />
      <DropdownMenuGroup>
        <DropdownMenuItem onClick={clearHistory}>
          <TrashIcon className="mr-2 h-4 w-4" />
          <span>Clear Saved Query</span>
        </DropdownMenuItem>
      </DropdownMenuGroup>
    </ResourceDropdownMenu>
  )
}

export default DatabasesMenu
