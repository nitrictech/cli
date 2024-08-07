import { useEffect, useState } from 'react'
import { useWebSocket } from '../../lib/hooks/use-web-socket'
import type { APIResponse, EventHistoryItem, Schedule, Topic } from '@/types'
import { Badge, Spinner, Tabs, Loading } from '../shared'
import APIResponseContent from '../apis/APIResponseContent'
import {
  fieldRowArrToHeaders,
  getHost,
  generateResponse,
  formatFileSize,
  formatResponseTime,
  formatJSON,
} from '../../lib/utils'
import EventsHistory from './EventsHistory'
import { useHistory } from '../../lib/hooks/use-history'
import CodeEditor from '../apis/CodeEditor'
import EventsMenu from './EventsMenu'
import AppLayout from '../layout/AppLayout'
import EventsTreeView from './EventsTreeView'
import { copyToClipboard } from '../../lib/utils/copy-to-clipboard'
import ClipboardIcon from '@heroicons/react/24/outline/ClipboardIcon'
import toast from 'react-hot-toast'
import { capitalize } from 'radash'
import { Button } from '../ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '../ui/tooltip'
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

interface Props {
  workerType: 'schedules' | 'topics'
}

type Worker = Schedule | Topic

const EventsExplorer: React.FC<Props> = ({ workerType }) => {
  const storageKey = `nitric-local-dash-${workerType}-history`

  const { data, loading } = useWebSocket()
  const [callLoading, setCallLoading] = useState(false)

  const { data: history } = useHistory(workerType)

  const [response, setResponse] = useState<APIResponse>()

  const [selectedWorker, setSelectedWorker] = useState<Worker>()
  const [responseTabIndex, setResponseTabIndex] = useState(0)

  const [eventHistory, setEventHistory] = useState<EventHistoryItem[]>([])

  const [body, setBody] = useState({})

  useEffect(() => {
    if (history) {
      setEventHistory(history ? history[workerType] : [])
    }
  }, [history])

  useEffect(() => {
    if (data && data[workerType]) {
      // restore history or select first if not selected
      if (!selectedWorker) {
        const previousId = localStorage.getItem(
          `${storageKey}-last-${workerType}`,
        )

        const worker =
          (previousId && data[workerType].find((s) => s.name === previousId)) ||
          data[workerType][0]

        setSelectedWorker(worker)
      } else {
        // could be a refresh from ws, so update the selected endpoint
        const latest = data[workerType].find(
          (s) => s.name === selectedWorker.name,
        )

        if (latest) {
          setSelectedWorker(latest)
        }
      }
    }
  }, [data])

  useEffect(() => {
    if (selectedWorker) {
      // set history
      localStorage.setItem(
        `${storageKey}-last-${workerType}`,
        selectedWorker.name,
      )
    }
  }, [selectedWorker])

  const handleSend = async (
    e: React.MouseEvent<HTMLButtonElement, MouseEvent>,
  ) => {
    if (!selectedWorker) return
    setCallLoading(true)
    e.preventDefault()

    const url =
      `http://${getHost()}/api/call` + `/${workerType}/${selectedWorker.name}`
    const requestOptions: RequestInit = {
      method: 'POST',
      body: JSON.stringify(body),
      headers: fieldRowArrToHeaders([
        {
          key: 'Accept',
          value: '*/*',
        },
        {
          key: 'User-Agent',
          value: 'Nitric Client (https://www.nitric.io)',
        },
        {
          key: 'X-Nitric-Local-Call-Address',
          value: data?.triggerAddress || 'localhost:4000',
        },
      ]),
    }

    const startTime = window.performance.now()
    const res = await fetch(url, requestOptions)

    const callResponse = await generateResponse(res, startTime)
    setResponse(callResponse)

    setTimeout(() => setCallLoading(false), 300)
  }

  const workerTitleSingle = capitalize(workerType).slice(0, -1)
  const generatedURL = `http://${data?.triggerAddress}/${workerType}/${selectedWorker?.name}`

  const hasData = Boolean(data && data[workerType]?.length)

  return (
    <AppLayout
      title={capitalize(workerType)}
      hideTitle
      routePath={`/${workerType}`}
      secondLevelNav={
        data &&
        selectedWorker && (
          <>
            <div className="flex min-h-12 items-center justify-between px-2 py-1">
              <span className="text-lg">{capitalize(workerType)}</span>
              <EventsMenu
                selected={selectedWorker.name}
                storageKey={storageKey}
                workerType={workerType}
                onAfterClear={() => {
                  return
                }}
              />
            </div>
            <EventsTreeView
              type={workerType}
              subscriptions={data.subscriptions}
              initialItem={selectedWorker}
              onSelect={(resource) => {
                setSelectedWorker(resource)
              }}
              resources={data[workerType] ?? []}
            />
          </>
        )
      }
    >
      <Loading delay={400} conditionToShow={!loading}>
        {selectedWorker && hasData ? (
          <div className="flex max-w-7xl flex-col gap-8 md:pr-8">
            <div className="flex w-full flex-col gap-8">
              <div className="lg:hidden">
                <div className="ml-auto flex items-center">
                  <EventsMenu
                    selected={selectedWorker.name}
                    storageKey={storageKey}
                    workerType={workerType}
                    onAfterClear={() => {
                      return
                    }}
                  />
                </div>
                {data![workerType] && (
                  <Select
                    value={selectedWorker.name}
                    onValueChange={(name) => {
                      setSelectedWorker(
                        data![workerType].find((b) => b.name === name),
                      )
                    }}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue
                        placeholder={`Select ${capitalize(workerType)}`}
                      />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        {data![workerType].map((worker) => (
                          <SelectItem key={worker.name} value={worker.name}>
                            {worker.name}
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                )}
              </div>
              <div className="flex items-center gap-4">
                <BreadCrumbs className="hidden text-lg lg:block">
                  <span>{capitalize(workerType)}</span>
                  <h2 className="font-body text-lg font-semibold">
                    {selectedWorker.name}
                  </h2>
                  <div className="flex items-center">
                    <span className="flex gap-2 text-lg">
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span
                            className="max-w-lg truncate"
                            data-testid="generated-request-path"
                          >
                            {generatedURL}
                          </span>
                        </TooltipTrigger>
                        <TooltipContent>
                          <p>{generatedURL}</p>
                        </TooltipContent>
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <button
                            type="button"
                            onClick={() => {
                              copyToClipboard(generatedURL)
                              toast.success(`Copied ${workerTitleSingle} URL`)
                            }}
                          >
                            <span className="sr-only">
                              Copy {workerTitleSingle} URL
                            </span>
                            <ClipboardIcon className="h-5 w-5 text-gray-500" />
                          </button>
                        </TooltipTrigger>
                        <TooltipContent>
                          <p>Copy {workerTitleSingle} URL</p>
                        </TooltipContent>
                      </Tooltip>
                    </span>
                  </div>
                </BreadCrumbs>
                {workerType === 'schedules' && (
                  <Button
                    size="lg"
                    className="ml-auto"
                    data-testid={`trigger-${workerType}-btn`}
                    onClick={handleSend}
                  >
                    Trigger
                  </Button>
                )}
              </div>

              {workerType === 'topics' && (
                <SectionCard title="Payload">
                  <div>
                    <CodeEditor
                      value={formatJSON(body)}
                      contentType="application/json"
                      onChange={(payload: string) => {
                        try {
                          setBody(JSON.parse(payload))
                        } catch {
                          return
                        }
                      }}
                    />

                    <Button
                      size="lg"
                      className="ml-auto mt-6 flex"
                      data-testid={`trigger-${workerType}-btn`}
                      onClick={handleSend}
                    >
                      {workerType === 'topics' ? 'Publish' : 'Trigger'}
                    </Button>
                  </div>
                </SectionCard>
              )}
              <SectionCard
                title="Response"
                headerSiblings={
                  <>
                    {callLoading && (
                      <Spinner
                        className="absolute left-0 top-0 ml-28"
                        color="info"
                        size={'md'}
                      />
                    )}
                    <div className="absolute right-0 top-0 flex gap-2">
                      {response?.status && (
                        <Badge
                          status={response.status >= 400 ? 'red' : 'green'}
                        >
                          Status: {response.status}
                        </Badge>
                      )}
                      {response?.time && (
                        <Badge status={'green'}>
                          Time: {formatResponseTime(response.time)}
                        </Badge>
                      )}
                      {typeof response?.size === 'number' && (
                        <Badge status={'green'}>
                          Size: {formatFileSize(response.size)}
                        </Badge>
                      )}
                    </div>
                  </>
                }
              >
                <div className="my-4 max-w-full text-sm">
                  {response?.data ? (
                    <div className="flex flex-col gap-4">
                      <Tabs
                        tabs={[
                          {
                            name: 'Response',
                          },
                          {
                            name: 'Headers',
                            count: Object.keys(response.headers || {}).length,
                          },
                        ]}
                        round
                        index={responseTabIndex}
                        setIndex={setResponseTabIndex}
                      />
                      {responseTabIndex === 0 && (
                        <APIResponseContent response={response} />
                      )}
                      {responseTabIndex === 1 && (
                        <div className="overflow-x-auto">
                          <div className="inline-block min-w-full py-2 align-middle">
                            <table className="min-w-full divide-y divide-gray-300">
                              <thead>
                                <tr>
                                  <th
                                    scope="col"
                                    className="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 sm:pl-6 lg:pl-8"
                                  >
                                    Header
                                  </th>
                                  <th
                                    scope="col"
                                    className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
                                  >
                                    Value
                                  </th>
                                </tr>
                              </thead>
                              <tbody className="divide-y divide-gray-200 bg-white">
                                {Object.entries(response.headers || {}).map(
                                  ([key, value]) => (
                                    <tr key={key}>
                                      <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:pl-6 lg:pl-8">
                                        {key}
                                      </td>
                                      <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500">
                                        {value}
                                      </td>
                                    </tr>
                                  ),
                                )}
                              </tbody>
                            </table>
                          </div>
                        </div>
                      )}
                    </div>
                  ) : response ? (
                    <span className="text-lg text-gray-500">
                      No response data available for this request.
                    </span>
                  ) : (
                    <span className="text-lg text-gray-500">
                      Send a request to get a response.
                    </span>
                  )}
                </div>
              </SectionCard>
            </div>
            <SectionCard
              title="History"
              className="m-0 mb-20 border-none px-0 shadow-none sm:px-0"
              headerClassName="px-4 sm:px-2"
            >
              <EventsHistory
                history={eventHistory}
                workerType={workerType}
                selectedWorker={selectedWorker}
              />
            </SectionCard>
          </div>
        ) : !hasData ? (
          <div>
            Please refer to our documentation on{' '}
            <a
              className="underline"
              target="_blank"
              href="https://nitric.io/docs/messaging"
              rel="noreferrer"
            >
              creating {capitalize(workerType)}
            </a>{' '}
            as we are unable to find any existing {workerType}.
          </div>
        ) : null}
      </Loading>
    </AppLayout>
  )
}

export default EventsExplorer
