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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '../ui/dropdown-menu'
import { EllipsisVerticalIcon } from '@heroicons/react/24/outline'

const LOCAL_STORAGE_KEY = 'nitric-local-dash-website-history'

const SiteExplorer = () => {
  const [selectedWebsite, setSelectedWebsite] = useState<Website>()
  const { data, loading } = useWebSocket()

  const { websites, localCloudMode } = data || {}

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
                  <div className="ml-auto space-x-4">
                    <Button asChild variant="outline">
                      <a
                        data-testid="requesting-service"
                        href={`vscode://file/${selectedWebsite.directory}`}
                      >
                        Open in VSCode
                      </a>
                    </Button>
                    <Button asChild>
                      <a
                        href={selectedWebsite.url}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        Open in a new tab
                      </a>
                    </Button>
                  </div>
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
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      size="icon"
                      variant="outline"
                      className="ml-auto"
                      data-testid="website-options-btn"
                    >
                      <span className="sr-only">Open website actions</span>
                      <EllipsisVerticalIcon
                        className="size-6 text-foreground"
                        aria-hidden="true"
                      />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent className="w-56">
                    <DropdownMenuGroup>
                      <DropdownMenuItem asChild>
                        <a href={`vscode://file/${selectedWebsite.directory}`}>
                          Open in VSCode
                        </a>
                      </DropdownMenuItem>
                      <DropdownMenuItem asChild>
                        <a
                          href={selectedWebsite.url}
                          target="_blank"
                          rel="noopener noreferrer"
                        >
                          Open in a new tab
                        </a>
                      </DropdownMenuItem>
                    </DropdownMenuGroup>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
              <div className="overflow-hidden rounded-lg border border-gray-300 shadow-md">
                <div className="bg-gray-50 p-2">
                  <div className="rounded-md bg-gray-200 p-2 text-center text-sm font-semibold text-gray-600">
                    {selectedWebsite.url}
                  </div>
                </div>
                {localCloudMode === 'run' || selectedWebsite.devUrl ? (
                  <iframe
                    src={selectedWebsite.url}
                    title={selectedWebsite.name}
                    className="h-screen w-full"
                  />
                ) : (
                  <p className="p-4">
                    A development URL is required when running{' '}
                    <pre className="inline-block px-0.5 text-xs">
                      nitric start
                    </pre>{' '}
                    to ensure the website operates in development mode.
                    <br /> Set the URL in your website dev config within your
                    nitric.yaml file, e.g. http://localhost:4321.
                  </p>
                )}
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
