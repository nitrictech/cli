import { useEffect, useState } from 'react'

import { Loading } from '../shared'
import { useWebSocket } from '../../lib/hooks/use-web-socket'
import AppLayout from '../layout/AppLayout'
import StorageTreeView from './StorageTreeView'
import FileBrowser from './FileBrowser'
import type { Bucket } from '@/types'
import BreadCrumbs from '../layout/BreadCrumbs'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select'
import SectionCard from '../shared/SectionCard'

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
      hideTitle
      routePath={'/storage'}
      secondLevelNav={
        buckets &&
        selectedBucket && (
          <>
            <div className="flex min-h-12 items-center justify-between px-2 py-1">
              <span className="text-lg">Buckets</span>
            </div>
            <StorageTreeView
              initialItem={selectedBucket}
              notifications={data?.notifications || []}
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
              <BreadCrumbs className="hidden text-lg lg:block">
                Buckets
                <h2 className="font-body text-lg font-semibold">
                  {selectedBucket.name}
                </h2>
              </BreadCrumbs>

              <div className="lg:hidden">
                <Select
                  value={selectedBucket.name}
                  onValueChange={(name) => {
                    setSelectedBucket(buckets.find((b) => b.name === name))
                  }}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Select Bucket" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {buckets.map((bucket) => (
                        <SelectItem key={bucket.name} value={bucket.name}>
                          {bucket.name}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>
              <SectionCard title="File Explorer">
                <FileBrowser bucket={selectedBucket.name} />
              </SectionCard>
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
