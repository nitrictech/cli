import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { copyToClipboard } from '@/lib/utils/copy-to-clipboard'
import type { SecretVersion } from '@/types'
import { EllipsisHorizontalIcon } from '@heroicons/react/20/solid'
import { ArrowsUpDownIcon } from '@heroicons/react/24/outline'
import type { ColumnDef } from '@tanstack/react-table'
import { Checkbox } from '@/components/ui/checkbox'
import { useSecretsContext } from '../SecretsContext'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { ScrollArea } from '@/components/ui/scroll-area'

export const columns: ColumnDef<SecretVersion>[] = [
  {
    id: 'select',
    header: ({ table }) => (
      <Checkbox
        className="flex"
        checked={
          table.getIsAllPageRowsSelected() ||
          (table.getIsSomePageRowsSelected() && 'indeterminate')
        }
        onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        aria-label="Select all"
      />
    ),
    cell: ({ row }) => (
      <Checkbox
        className="flex"
        checked={row.getIsSelected()}
        onCheckedChange={(value) => row.toggleSelected(!!value)}
        aria-label="Select row"
      />
    ),
  },
  {
    accessorKey: 'version',
    header: 'Version',
    cell: ({ row }) => {
      const secretVersion = row.original

      return (
        <div className="text-left">
          {secretVersion.version}
          {secretVersion.latest && (
            <Badge
              className="ml-2"
              data-testid={`data-table-${row.id}-latest-badge`}
            >
              Latest
            </Badge>
          )}
        </div>
      )
    },
  },
  {
    accessorKey: 'value',
    header: 'Value',
    cell: ({ row }) => {
      const secretVersion = row.original

      return (
        <div className="text-left">
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="max-w-md truncate font-mono text-muted-foreground">
                {secretVersion.value}
              </div>
            </TooltipTrigger>
            <TooltipContent>
              <ScrollArea className="max-w-96 whitespace-pre font-mono text-muted-foreground">
                <div className="max-h-72">{secretVersion.value}</div>
              </ScrollArea>
            </TooltipContent>
          </Tooltip>
        </div>
      )
    },
  },
  {
    accessorKey: 'createdAt',
    header: ({ column }) => {
      return (
        <Button
          variant="ghost"
          className="text-foreground hover:text-accent-foreground hover:bg-accent"
          onClick={() => column.toggleSorting(column.getIsSorted() === 'asc')}
        >
          Created At
          <ArrowsUpDownIcon className="ml-2 h-4 w-4 text-muted-foreground group-hover:text-foreground" />
        </Button>
      )
    },
  },
  {
    accessorKey: 'Actions',
    cell: ({ row }) => {
      const secretVersion = row.original
      const { setDialogAction, setDialogOpen, setSelectedVersions } =
        useSecretsContext()

      return (
        <>
          <DropdownMenu modal={false}>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="h-8 w-8 p-0 hover:bg-accent">
                <span className="sr-only">Open menu</span>
                <EllipsisHorizontalIcon className="h-4 w-4 text-muted-foreground hover:text-foreground" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="bg-popover border-border">
              <DropdownMenuLabel className="text-foreground">Actions</DropdownMenuLabel>
              <DropdownMenuItem
                onClick={() => copyToClipboard(secretVersion.value)}
              >
                Copy secret value
              </DropdownMenuItem>
              <DropdownMenuItem
                onSelect={() => {
                  setSelectedVersions([secretVersion])
                  setDialogAction('delete')
                  setDialogOpen(true)
                }}
              >
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </>
      )
    },
  },
]
