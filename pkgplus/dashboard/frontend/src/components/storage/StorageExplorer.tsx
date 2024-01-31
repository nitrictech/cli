import { useEffect, useState } from 'react'

import { Loading, Select } from '../shared'
import { useWebSocket } from '../../lib/hooks/use-web-socket'
import AppLayout from '../layout/AppLayout'
import StorageTreeView from './StorageTreeView'
import FileBrowser from './FileBrowser'
import type { Bucket } from '@/types'

const LOCAL_STORAGE_KEY = 'nitric-local-dash-storage-history'

const StorageExplorer = () => {
  const [selectedBucket, setSelectedBucket] = useState<Bucket>()
  const { data, loading } = useWebSocket()

  const { buckets } = data || {}

  useEffect(() => {
    if (buckets?.length && !selectedBucket) {
      const previousBucket = localStorage.getItem(
        `${LOCAL_STORAGE_KEY}-last-bucket`,
      )

      setSelectedBucket(
        buckets.find((b) => b.name === previousBucket) || buckets[0],
      )
    }
  }, [buckets])

  useEffect(() => {
    if (selectedBucket) {
      // set history
      localStorage.setItem(
        `${LOCAL_STORAGE_KEY}-last-bucket`,
        selectedBucket.name,
      )
    }
  }, [selectedBucket])

  return (
    <AppLayout
      title="Storage"
      routePath={'/storage'}
      secondLevelNav={
        buckets &&
        selectedBucket && (
          <>
            <div className="mb-2 flex items-center justify-between px-2">
              <span className="text-lg">Buckets</span>
            </div>
            <StorageTreeView
              initialItem={selectedBucket}
              onSelect={(b) => {
                setSelectedBucket(b)
              }}
              buckets={buckets}
            />
          </>
        )
      }
    >
      <Loading delay={400} conditionToShow={!loading}>
        {buckets && selectedBucket ? (
          <div className="flex max-w-7xl flex-col gap-8 md:flex-row md:pr-8">
            <div className="flex w-full flex-col gap-8">
              <h2 className="text-2xl font-medium">{selectedBucket.name}</h2>
              <div className="md:hidden">
                <nav className="flex items-end gap-4" aria-label="Breadcrumb">
                  <ol className="flex min-w-[200px] items-center gap-4">
                    <li className="w-full">
                      <Select
                        id="bucket-select"
                        items={buckets || []}
                        label="Select Bucket"
                        selected={selectedBucket}
                        setSelected={setSelectedBucket}
                        display={(v) => (
                          <div className="flex items-center gap-4 p-0.5 text-lg">
                            {v.name}
                          </div>
                        )}
                      />
                    </li>
                  </ol>
                </nav>
              </div>
              <div className="bg-white shadow sm:rounded-lg">
                <div className="flex flex-col gap-4 px-4 py-5 sm:p-6">
                  <FileBrowser bucket={selectedBucket.name} />
                </div>
              </div>
            </div>
          </div>
        ) : !buckets?.length ? (
          <div>
            <p>
              Please refer to our documentation on{' '}
              <a
                className="underline"
                target="_blank"
                href="https://nitric.io/docs/storage#buckets"
                rel="noreferrer"
              >
                creating buckets
              </a>{' '}
              as we are unable to find any existing buckets.
            </p>
            <p>
              To ensure that the buckets are created, execute an API that
              utilizes them.
            </p>
          </div>
        ) : null}
      </Loading>
    </AppLayout>
  )
}

export default StorageExplorer
