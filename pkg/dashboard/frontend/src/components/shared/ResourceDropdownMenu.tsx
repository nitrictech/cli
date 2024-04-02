import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '../ui/dropdown-menu'
import { EllipsisHorizontalIcon } from '@heroicons/react/20/solid'

import { Button } from '../ui/button'
import type { PropsWithChildren } from 'react'

const ResourceDropdownMenu = ({ children }: PropsWithChildren) => {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="ml-auto">
          <span className="sr-only">Open options</span>
          <EllipsisHorizontalIcon className="size-6" aria-hidden="true" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent className="w-56">{children}</DropdownMenuContent>
    </DropdownMenu>
  )
}

export default ResourceDropdownMenu
