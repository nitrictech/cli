import { DataTable } from '@/components/shared/DataTable'
import type { SecretVersion } from '@/types'
import { columns } from './columns'
import { Button } from '@/components/ui/button'
import { MinusIcon, PlusIcon } from '@heroicons/react/20/solid'
import { VersionActionDialog } from '../VersionActionDialog'
import { useState } from 'react'
import { useSecretsContext } from '../SecretsContext'
import { useSecret } from '@/lib/hooks/use-secret'

export const SecretVersionsTable = () => {
  const {
    selectedSecret,
    setSelectedVersions,
    setDialogAction,
    setDialogOpen,
  } = useSecretsContext()
  const { data: secretVersions } = useSecret(selectedSecret?.name)

  if (!selectedSecret || !secretVersions) return null

  return (
    <>
      <DataTable
        title="Versions"
        columns={columns}
        data={secretVersions}
        headerSiblings={(selected) => {
          const versions =
            Object.keys(selected)
              .map((s) => {
                return secretVersions[parseInt(s)]
              })
              .filter(Boolean) ?? []

          return (
            <div className="flex gap-2">
              <Button
                size="sm"
                variant="outline"
                className="text-foreground hover:text-accent-foreground border-border"
                data-testid="delete-selected-versions"
                disabled={versions.length === 0}
                onClick={() => {
                  setSelectedVersions(versions)
                  setDialogAction('delete')
                  setDialogOpen(true)
                }}
              >
                <MinusIcon className="mr-1 h-5 w-5 text-muted-foreground group-hover:text-foreground" />
                Delete Selected
              </Button>

              <Button
                size="sm"
                className="text-primary-foreground hover:bg-primary/90"
                data-testid="create-new-version"
                onClick={() => {
                  setDialogAction('add')
                  setDialogOpen(true)
                }}
              >
                <PlusIcon className="mr-1 h-5 w-5 text-primary-foreground" />
                Create New Version
              </Button>
            </div>
          )
        }}
        noResultsChildren={
          <div className="flex flex-col items-center justify-center gap-6 text-muted-foreground">
            <span className="text-lg text-foreground">No versions found.</span>{' '}
            <Button
              size="sm"
              className="text-primary-foreground hover:bg-primary/90"
              onClick={() => {
                setDialogAction('add')
                setDialogOpen(true)
              }}
            >
              <PlusIcon className="mr-1 h-5 w-5 text-primary-foreground" />
              Create New Version
            </Button>
          </div>
        }
      />
    </>
  )
}
