import { useEffect, useRef, useState } from 'react'
import type { APIRequest, WebSocket, WebSocketsInfo } from '../../types'
import { FieldRows, Loading } from '../shared'
import { formatJSON, generatePath, getHost } from '../../lib/utils'

import { useWebSocket } from '../../lib/hooks/use-web-socket'
import AppLayout from '../layout/AppLayout'
import WSTreeView from './WSTreeView'
import { copyToClipboard } from '../../lib/utils/copy-to-clipboard'
import toast from 'react-hot-toast'
import {
  CheckCircleIcon,
  ClipboardIcon,
  ArrowDownCircleIcon,
  ArrowUpCircleIcon,
  ExclamationCircleIcon,
  InformationCircleIcon,
  TrashIcon,
} from '@heroicons/react/24/outline'
import { Button } from '../ui/button'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '../ui/accordion'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select'
import { format } from 'date-fns/format'

import { Input } from '../ui/input'
import { ScrollArea } from '../ui/scroll-area'
import CodeEditor from '../apis/CodeEditor'

import { Textarea } from '../ui/textarea'
import useSWRSubscription from 'swr/subscription'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../ui/tabs'
import { Badge } from '../ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '../ui/tooltip'
import BreadCrumbs from '../layout/BreadCrumbs'
import SectionCard from '../shared/SectionCard'
import NotFoundAlert from '../shared/NotFoundAlert'

export const LOCAL_STORAGE_KEY = 'nitric-local-dash-api-history'

interface Message {
  ts: number
  data: any
  type: 'connect' | 'disconnect' | 'message-in' | 'message-out' | 'error'
}

const MessageIcon = ({ type }: Pick<Message, 'type'>) => {
  const className = 'w-6 h-6 mr-1'

  switch (type) {
    case 'connect':
      return <CheckCircleIcon className={`${className} text-green-500`} />
    case 'error':
      return <ExclamationCircleIcon className={`${className} text-red-500`} />
    case 'message-in':
      return <ArrowDownCircleIcon className={`${className} text-blue-500`} />
    case 'message-out':
      return <ArrowUpCircleIcon className={`${className} text-orange-500`} />
  }

  return <InformationCircleIcon className={className} />
}

const WSExplorer = () => {
  const { data, loading } = useWebSocket()
  const websocketRef = useRef<globalThis.WebSocket>()
  const [messages, setMessages] = useState<Message[]>([])
  const [currentPayload, setCurrentPayload] = useState<string>()
  const [payloadType, setPayloadType] = useState('text')
  const [monitorMessageFilter, setMonitorMessageFilter] = useState('')
  const [messageFilter, setMessageFilter] = useState('')
  const [messageTypeFilter, setMessageTypeFilter] = useState('all')
  const [tab, setTab] = useState('monitor')
  const [selectedWebsocket, setSelectedWebsocket] = useState<WebSocket>()

  const [connected, setConnected] = useState(false)
  const [queryParams, setQueryParams] = useState<APIRequest['queryParams']>([
    {
      key: '',
      value: '',
    },
  ])

  useEffect(() => {
    if (selectedWebsocket) {
      setMonitorMessageFilter('')
      setMessageFilter('')

      // restore data
      const dataStr = localStorage.getItem(
        `nitric-local-ws-history-${selectedWebsocket?.name}`,
      )

      if (dataStr) {
        const data = JSON.parse(dataStr)

        setQueryParams(data.queryParams)
        setMessages(data.messages)
        setTab(data.tab)
        setPayloadType(data.payloadType)
        setCurrentPayload(data.currentPayload)
      } else {
        setQueryParams([
          {
            key: '',
            value: '',
          },
        ])
        setMessages([])
        setTab('monitor')
        setPayloadType('text')
        setCurrentPayload('')
      }

      if (connected) {
        websocketRef.current?.close(4001) // to indicate in close callback to not add close message
      }
    }
  }, [selectedWebsocket])

  useEffect(() => {
    localStorage.setItem(
      `nitric-local-ws-history-${selectedWebsocket?.name}`,
      JSON.stringify({
        queryParams,
        messages,
        payloadType,
        currentPayload,
        tab,
      }),
    )
  }, [queryParams, messages, payloadType, currentPayload, tab])

  useEffect(() => {
    if (!selectedWebsocket && data?.websockets?.length) {
      setSelectedWebsocket(data?.websockets[0])
    }
  }, [data?.websockets])

  const websocketAddress =
    selectedWebsocket && data?.websocketAddresses[selectedWebsocket?.name]
      ? `ws://${generatePath(
          data?.websocketAddresses[selectedWebsocket?.name],
          [],
          queryParams,
        )}`
      : ''

  const host = getHost()

  const { data: wsData, error } = useSWRSubscription<WebSocketsInfo>(
    host ? `ws://${host}/ws-info` : null,
    (key: any, { next }: any) => {
      const socket = new WebSocket(key)

      socket.addEventListener('message', (event) => {
        const message = JSON.parse(event.data) as WebSocketsInfo

        next(null, message)
      })

      socket.addEventListener('error', (event: any) => next(event.error))
      return () => socket.close()
    },
  )

  const wsInfo =
    selectedWebsocket && wsData ? wsData![selectedWebsocket?.name] : undefined

  useEffect(() => {
    if (websocketAddress && connected) {
      const socket = new WebSocket(websocketAddress)

      // set socket ref
      websocketRef.current = socket

      socket.addEventListener('message', (event) => {
        setMessages((prev) => [
          {
            data: event.data,
            ts: new Date().getTime(),
            type: 'message-in',
          },
          ...prev,
        ])
      })
      socket.addEventListener('error', (event: any) => {
        setMessages((prev) => [
          {
            data: `Error connecting to ${websocketAddress}, check your connect callback`,
            ts: new Date().getTime(),
            type: 'error',
          },
          ...prev,
        ])
      })
      // Event listener to handle connection open
      socket.addEventListener('open', (event) => {
        setMessages((prev) => [
          {
            data: `Connected to ${websocketAddress}`,
            ts: new Date().getTime(),
            type: 'connect',
          },
          ...prev,
        ])
      })

      socket.addEventListener('close', (event) => {
        if (event.code !== 4001) {
          // code from switching websockets in dash, ignore disconnect message
          setMessages((prev) => [
            {
              data: `Disconnected from ${websocketAddress}`,
              ts: new Date().getTime(),
              type: 'disconnect',
            },
            ...prev,
          ])
        }

        websocketRef.current = undefined

        setConnected(false)
      })
    } else if (websocketRef.current) {
      websocketRef.current.close()
    }

    return () => websocketRef.current?.close()
  }, [connected])

  const sendMessage = () => {
    if (currentPayload) {
      websocketRef.current?.send(currentPayload)
      setMessages((prev) => [
        {
          data: currentPayload,
          ts: new Date().getTime(),
          type: 'message-out',
        },
        ...prev,
      ])
    }
  }

  const clearMessages = async () => {
    if (!selectedWebsocket) return

    await toast.promise(
      fetch(
        `http://${getHost()}/api/ws-clear-messages?socket=${encodeURIComponent(
          selectedWebsocket.name,
        )}`,
        {
          method: 'DELETE',
        },
      ),
      {
        error: 'Error clearinging messages',
        loading: 'Clearing messages',
        success: 'Messages cleared',
      },
    )
  }

  return (
    <AppLayout
      title="WebSockets"
      routePath={'/websockets'}
      hideTitle
      secondLevelNav={
        data?.websockets?.length && selectedWebsocket ? (
          <>
            <div className="flex min-h-12 items-center justify-between px-2 py-1">
              <span className="text-lg">WebSockets</span>
            </div>
            <WSTreeView
              initialItem={selectedWebsocket}
              onSelect={(ws) => {
                setSelectedWebsocket(ws)
              }}
              websockets={data.websockets}
            />
          </>
        ) : null
      }
    >
      <Loading delay={400} conditionToShow={!loading}>
        {data?.websockets?.length && selectedWebsocket ? (
          <div className="flex max-w-7xl flex-col md:pr-8">
            <div className="flex w-full flex-col gap-8">
              <div>
                <BreadCrumbs className="mb-2 lg:hidden">
                  WebSockets
                  <span className="font-semibold">
                    {selectedWebsocket?.name}
                  </span>
                </BreadCrumbs>

                <div className="flex w-full items-center gap-4 lg:hidden">
                  {data?.websockets?.length ? (
                    <Select
                      value={selectedWebsocket.name}
                      onValueChange={(socketName) => {
                        const ws = data?.websockets.find(
                          (ws) => ws.name === socketName,
                        )

                        setSelectedWebsocket(ws)
                      }}
                    >
                      <SelectTrigger>
                        <SelectValue placeholder="Select Message Type" />
                      </SelectTrigger>
                      <SelectContent>
                        {data?.websockets.map((ws) => (
                          <SelectItem key={ws.name} value={ws.name}>
                            {ws.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  ) : null}
                  {tab === 'send-messages' && (
                    <Button
                      onClick={() => setConnected(!connected)}
                      data-testid="connect-btn"
                      variant={connected ? 'destructive' : 'default'}
                    >
                      {connected ? 'Disconnect' : 'Connect'}
                    </Button>
                  )}
                </div>
                <div className="hidden h-10 items-center lg:flex">
                  <BreadCrumbs className="hidden text-lg lg:block">
                    WebSockets
                    <h2 className="font-body text-lg font-semibold">
                      {selectedWebsocket?.name}
                    </h2>
                    <div className="flex items-center">
                      <span className="flex gap-2 text-lg">
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <span
                              data-testid="generated-request-path"
                              className="max-w-xl truncate"
                            >
                              {websocketAddress}
                            </span>
                          </TooltipTrigger>
                          <TooltipContent>
                            <p>{websocketAddress}</p>
                          </TooltipContent>
                        </Tooltip>

                        <Tooltip>
                          <TooltipTrigger asChild>
                            <button
                              type="button"
                              onClick={() => {
                                copyToClipboard(websocketAddress)
                                toast.success('Copied Websocket URL')
                              }}
                            >
                              <span className="sr-only">Copy Route URL</span>
                              <ClipboardIcon className="h-5 w-5 text-gray-500" />
                            </button>
                          </TooltipTrigger>
                          <TooltipContent>
                            <p>Copy</p>
                          </TooltipContent>
                        </Tooltip>
                      </span>
                    </div>
                  </BreadCrumbs>
                  {tab === 'send-messages' && (
                    <div className="ml-auto">
                      <Button
                        onClick={() => setConnected(!connected)}
                        size={'lg'}
                        data-testid="connect-btn"
                        variant={connected ? 'destructive' : 'default'}
                      >
                        {connected ? 'Disconnect' : 'Connect'}
                      </Button>
                    </div>
                  )}
                </div>
                {!data?.websockets.some(
                  (s) => s.name === selectedWebsocket.name,
                ) && (
                  <NotFoundAlert className="mt-4">
                    WebSocket not found. It might have been updated or removed.
                    Select another WebSocket.
                  </NotFoundAlert>
                )}
              </div>
              <Tabs value={tab} onValueChange={setTab}>
                <TabsList>
                  <TabsTrigger
                    value="monitor"
                    data-testid="monitor-tab-trigger"
                  >
                    Monitor
                  </TabsTrigger>
                  <TabsTrigger
                    value="send-messages"
                    data-testid="send-messages-tab-trigger"
                  >
                    Send Messages
                  </TabsTrigger>
                </TabsList>
                <TabsContent value="monitor">
                  <SectionCard
                    className="mt-4"
                    title="Messages"
                    headerSiblings={
                      <div className="flex gap-2 ">
                        <Badge
                          data-testid="connections-status"
                          className="font-semibold uppercase"
                          variant={
                            wsInfo && wsInfo.connectionCount > 0
                              ? 'success'
                              : 'destructive'
                          }
                        >
                          Connections: {wsInfo?.connectionCount || 0}
                        </Badge>
                      </div>
                    }
                  >
                    <div className="flex gap-2">
                      <Input
                        placeholder="Search"
                        className="w-4/12"
                        value={monitorMessageFilter}
                        onChange={(evt) =>
                          setMonitorMessageFilter(evt.target.value)
                        }
                      />

                      <Button
                        data-testid="clear-messages-btn"
                        variant="outline"
                        onClick={clearMessages}
                      >
                        <TrashIcon className="mr-2 h-4 w-4" />
                        Clear Messages
                      </Button>
                    </div>
                    <div className="my-4 max-w-full text-sm">
                      {wsInfo?.messages?.length ? (
                        <ScrollArea
                          className="h-[50vh] w-full px-6"
                          type="always"
                        >
                          {wsInfo.messages
                            .filter((message) => {
                              let pass = true

                              if (
                                monitorMessageFilter &&
                                typeof message.data === 'string'
                              ) {
                                pass = message.data
                                  .toLowerCase()
                                  .includes(monitorMessageFilter.toLowerCase())
                              }

                              return pass
                            })
                            .map((message, i) => {
                              const shouldBeJSON = /^[{[]/.test(
                                message.data.trim(),
                              )

                              return (
                                <Accordion type="multiple" key={i}>
                                  <AccordionItem value={message.time}>
                                    <AccordionTrigger className="flex justify-between">
                                      <div>
                                        <MessageIcon
                                          type={
                                            message.data ===
                                            'Binary messages are not currently supported by AWS'
                                              ? 'error'
                                              : 'message-in'
                                          }
                                        />
                                      </div>
                                      <span
                                        data-testid={`accordion-message-${i}`}
                                        className="max-w-3xl truncate px-2"
                                      >
                                        {message.data}
                                      </span>
                                      <span className="ml-auto px-2">
                                        {format(
                                          new Date(message.time),
                                          'HH:mm:ss',
                                        )}
                                      </span>
                                    </AccordionTrigger>
                                    <AccordionContent>
                                      {message.data ===
                                      'Binary messages are not currently supported by AWS' ? (
                                        <p>
                                          Binary messages are not currently
                                          supported by AWS. Util this is
                                          supported, use a text-based payload.
                                        </p>
                                      ) : (
                                        <CodeEditor
                                          id="message-viewer"
                                          contentType={
                                            shouldBeJSON
                                              ? 'application/json'
                                              : 'text/html'
                                          }
                                          readOnly
                                          value={
                                            shouldBeJSON
                                              ? formatJSON(message.data)
                                              : message.data
                                          }
                                          height="208px"
                                          className="h-52"
                                        />
                                      )}
                                    </AccordionContent>
                                  </AccordionItem>
                                </Accordion>
                              )
                            })}
                        </ScrollArea>
                      ) : (
                        <span className="text-lg text-gray-500">
                          Send a message to get a response.
                        </span>
                      )}
                    </div>
                  </SectionCard>
                </TabsContent>
                <TabsContent value="send-messages" className="space-y-10">
                  <SectionCard className="mt-4" title="Query Params">
                    <div className="w-full">
                      <FieldRows
                        rows={queryParams}
                        readOnly={connected}
                        addRowLabel="Add Query Param"
                        testId="query"
                        setRows={(rows) => {
                          setQueryParams(rows)
                        }}
                      />
                    </div>
                  </SectionCard>

                  <SectionCard
                    className="mt-4"
                    title="Message"
                    footer={
                      <>
                        <Select
                          value={payloadType}
                          onValueChange={setPayloadType}
                        >
                          <SelectTrigger className="w-[150px]">
                            <SelectValue placeholder="Select Message Type" />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="text">Text</SelectItem>
                            <SelectItem value="json">JSON</SelectItem>
                            <SelectItem value="xml">XML</SelectItem>
                            <SelectItem value="html">HTML</SelectItem>
                          </SelectContent>
                        </Select>
                        <Button
                          className="ml-auto"
                          data-testid="send-message-btn"
                          disabled={!currentPayload || !connected}
                          onClick={sendMessage}
                        >
                          Send
                        </Button>
                      </>
                    }
                  >
                    {payloadType === 'text' && (
                      <Textarea
                        placeholder="Enter message"
                        data-testid="message-text-input"
                        value={currentPayload}
                        onChange={(evt) => setCurrentPayload(evt.target.value)}
                      />
                    )}
                    {['json', 'xml', 'html'].includes(payloadType) && (
                      <CodeEditor
                        id="message-editor"
                        contentType={
                          {
                            json: 'application/json',
                            xml: 'application/xml',
                            html: 'text/html',
                          }[payloadType] || ''
                        }
                        value={
                          typeof currentPayload === 'string'
                            ? currentPayload
                            : ''
                        }
                        height="208px"
                        className="h-52"
                        includeLinters
                        onChange={(value) => {
                          setCurrentPayload(value)
                        }}
                      />
                    )}
                  </SectionCard>

                  <SectionCard
                    className="mt-4"
                    title="Messages"
                    headerSiblings={
                      <div className="flex gap-2">
                        <Badge
                          data-testid="connected-status"
                          className="font-semibold uppercase"
                          variant={connected ? 'success' : 'destructive'}
                        >
                          {connected ? 'Connected' : 'Disconnected'}
                        </Badge>
                      </div>
                    }
                  >
                    <div className="my-4 flex gap-2 pt-4">
                      <Input
                        placeholder="Search"
                        className="w-4/12"
                        onChange={(evt) => setMessageFilter(evt.target.value)}
                      />
                      <Select
                        value={messageTypeFilter}
                        onValueChange={setMessageTypeFilter}
                      >
                        <SelectTrigger className="w-[150px]">
                          <SelectValue placeholder="Select" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="all">All Messages</SelectItem>
                          <SelectItem value="out">Sent</SelectItem>
                          <SelectItem value="in">Recieved</SelectItem>
                        </SelectContent>
                      </Select>
                      <Button variant="outline" onClick={() => setMessages([])}>
                        <TrashIcon className="mr-2 h-4 w-4" />
                        Clear Messages
                      </Button>
                    </div>
                    <div className="-mx-4 my-4 max-w-full text-sm">
                      {messages.length ? (
                        <ScrollArea className="h-[30vh] px-6" type="always">
                          {messages
                            .filter((message) => {
                              let pass = false

                              if (messageTypeFilter === 'in') {
                                pass = message.type === 'message-in'
                              } else if (messageTypeFilter === 'out') {
                                pass = message.type === 'message-out'
                              } else {
                                pass = true
                              }

                              if (
                                messageFilter &&
                                typeof message.data === 'string'
                              ) {
                                pass = message.data
                                  .toLowerCase()
                                  .includes(messageFilter.toLowerCase())
                              }

                              return pass
                            })
                            .map((message, i) => {
                              const shouldBeJSON = /^[{[]/.test(
                                message.data.trim(),
                              )

                              return (
                                <Accordion type="multiple" key={i}>
                                  <AccordionItem value={message.ts.toString()}>
                                    <AccordionTrigger className="flex justify-between">
                                      <div>
                                        <MessageIcon type={message.type} />
                                      </div>
                                      <span
                                        data-testid={`accordion-message-${i}`}
                                        className="truncate px-2"
                                      >
                                        {message.data}
                                      </span>
                                      <span className="ml-auto px-2">
                                        {format(
                                          new Date(message.ts),
                                          'HH:mm:ss',
                                        )}
                                      </span>
                                    </AccordionTrigger>
                                    <AccordionContent>
                                      <CodeEditor
                                        id="message-viewer"
                                        contentType={
                                          shouldBeJSON
                                            ? 'application/json'
                                            : 'text/html'
                                        }
                                        readOnly
                                        value={
                                          shouldBeJSON
                                            ? formatJSON(message.data)
                                            : message.data
                                        }
                                        height="208px"
                                        className="h-52"
                                      />
                                    </AccordionContent>
                                  </AccordionItem>
                                </Accordion>
                              )
                            })}
                        </ScrollArea>
                      ) : (
                        <span className="text-lg text-gray-500">
                          Send a message to get a response.
                        </span>
                      )}
                    </div>
                  </SectionCard>
                </TabsContent>
              </Tabs>
            </div>
          </div>
        ) : (
          <div>
            Please refer to our documentation on{' '}
            <a
              className="underline"
              target="_blank"
              href="https://nitric.io/docs/websockets"
              rel="noreferrer"
            >
              creating WebSockets
            </a>{' '}
            as we are unable to find any existing WebSockets.
          </div>
        )}
      </Loading>
    </AppLayout>
  )
}

export default WSExplorer
