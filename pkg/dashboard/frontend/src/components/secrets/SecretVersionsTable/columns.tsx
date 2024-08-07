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
          <span className="text-gray-500 font-mono">
            {secretVersion.value.split('\n').map((line, index) => (
              <span key={index}>
                {line}
                <br />
              </span>
            ))}
          </span>
        </div>
      )
    }
  },
  {
    accessorKey: 'createdAt',
    header: ({ column }) => {
      return (
        <Button
          variant="ghost"
          onClick={() => column.toggleSorting(column.getIsSorted() === 'asc')}
        >
          Created At
          <ArrowsUpDownIcon className="ml-2 h-4 w-4" />
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
              <Button variant="ghost" className="h-8 w-8 p-0">
                <span className="sr-only">Open menu</span>
                <EllipsisHorizontalIcon className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>Actions</DropdownMenuLabel>
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
