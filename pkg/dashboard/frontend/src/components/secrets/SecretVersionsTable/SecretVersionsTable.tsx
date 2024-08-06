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
                disabled={versions.length === 0}
                onClick={() => {
                  setSelectedVersions(versions)
                  setDialogAction('delete')
                  setDialogOpen(true)
                }}
              >
                <MinusIcon className="mr-1 h-5 w-5" />
                Delete Selected
              </Button>

              <Button
                size="sm"
                onClick={() => {
                  setDialogAction('add')
                  setDialogOpen(true)
                }}
              >
                <PlusIcon className="mr-1 h-5 w-5" />
                Create New Version
              </Button>
            </div>
          )
        }}
        noResultsChildren={
          <div className="flex flex-col items-center justify-center gap-6">
            <span className="text-lg">No versions found.</span>{' '}
            <Button
              size="sm"
              onClick={() => {
                setDialogAction('add')
                setDialogOpen(true)
              }}
            >
              <PlusIcon className="mr-1 h-5 w-5" />
              Create New Version
            </Button>
          </div>
        }
      />
    </>
  )
}
