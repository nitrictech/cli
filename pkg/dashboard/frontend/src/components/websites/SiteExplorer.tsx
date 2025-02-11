import { useEffect, useState } from 'react'

import { Loading } from '../shared'
import { useWebSocket } from '../../lib/hooks/use-web-socket'
import AppLayout from '../layout/AppLayout'
import type { Website } from '@/types'
import BreadCrumbs from '../layout/BreadCrumbs'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select'
import NotFoundAlert from '../shared/NotFoundAlert'
import SiteTreeView from './SiteTreeView'
import { Button } from '../ui/button'

const LOCAL_STORAGE_KEY = 'nitric-local-dash-storage-history'

const SiteExplorer = () => {
  const [selectedWebsite, setSelectedWebsite] = useState<Website>()
  const { data, loading } = useWebSocket()

  const { websites } = data || {}

  useEffect(() => {
    if (websites?.length && !selectedWebsite) {
      const previousWebsite = localStorage.getItem(
        `${LOCAL_STORAGE_KEY}-last-website`,
      )

      setSelectedWebsite(
        websites.find((b) => b.name === previousWebsite) || websites[0],
      )
    }
  }, [websites])

  useEffect(() => {
    if (selectedWebsite) {
      // set history
      localStorage.setItem(
        `${LOCAL_STORAGE_KEY}-last-website`,
        selectedWebsite.name,
      )
    }
  }, [selectedWebsite])

  return (
    <AppLayout
      hideTitle
      title="Websites"
      routePath={'/websites'}
      secondLevelNav={
        websites &&
        selectedWebsite && (
          <>
            <div className="flex min-h-12 items-center justify-between px-2 py-1">
              <span className="text-lg">Websites</span>
            </div>
            <SiteTreeView
              initialItem={selectedWebsite}
              onSelect={(b) => {
                setSelectedWebsite(b)
              }}
              websites={websites}
            />
          </>
        )
      }
    >
      <Loading delay={400} conditionToShow={!loading}>
        {websites && selectedWebsite ? (
          <div className="flex max-w-screen-2xl flex-col gap-8 md:flex-row md:pr-8">
            <div className="flex w-full flex-col gap-8">
              <div className="space-y-4">
                <div className="hidden items-center justify-between lg:flex">
                  <BreadCrumbs className="text-lg">
                    Websites
                    <h2 className="font-body text-lg font-semibold">
                      {selectedWebsite.name}
                    </h2>
                  </BreadCrumbs>
                  <Button asChild>
                    <a
                      href={selectedWebsite.url}
                      target="_blank"
                      rel="noopener noreferrer ml-auto"
                    >
                      Open in a new tab
                    </a>
                  </Button>
                </div>

                {!data?.websites.some(
                  (s) => s.name === selectedWebsite.name,
                ) && (
                  <NotFoundAlert>
                    Website not found. It might have been updated or removed.
                    Select another website.
                  </NotFoundAlert>
                )}
              </div>

              <div className="flex gap-2 lg:hidden">
                <Select
                  value={selectedWebsite.name}
                  onValueChange={(name) => {
                    setSelectedWebsite(websites.find((b) => b.name === name))
                  }}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Select Website" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {websites.map((website) => (
                        <SelectItem key={website.name} value={website.name}>
                          {website.name}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
                <Button asChild>
                  <a
                    href={selectedWebsite.url}
                    target="_blank"
                    rel="noopener noreferrer ml-auto"
                  >
                    Open in a new tab
                  </a>
                </Button>
              </div>
              <div className="overflow-hidden rounded-lg border border-gray-300 shadow-md">
                <div className="bg-gray-50 p-2">
                  <div className="rounded-md bg-gray-200 p-2 text-center text-sm font-semibold text-gray-600">
                    {selectedWebsite.url}
                  </div>
                </div>
                <iframe
                  src={selectedWebsite.url}
                  title={selectedWebsite.name}
                  className="h-screen w-full"
                />
              </div>
            </div>
          </div>
        ) : !websites?.length ? (
          <div>
            <p>
              Please refer to our documentation on{' '}
              <a
                className="underline"
                target="_blank"
                href="https://nitric.io/docs/websites"
                rel="noreferrer"
              >
                creating websites
              </a>{' '}
              as we are unable to find any existing websites.
            </p>
          </div>
        ) : null}
      </Loading>
    </AppLayout>
  )
}

export default SiteExplorer
