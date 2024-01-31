import type { OpenAPIV3 } from 'openapi-types'
import type { FieldRow } from './components/shared/FieldRows'

export type APIDoc = OpenAPIV3.Document

export type Method =
  | 'GET'
  | 'PUT'
  | 'POST'
  | 'DELETE'
  | 'OPTIONS'
  | 'HEAD'
  | 'PATCH'
  | 'TRACE'

export interface BaseResource {
  name: string
  filePath: string
  requestingServices: string[]
}

export interface Api extends BaseResource {
  spec: APIDoc
}

export interface Schedule extends BaseResource {
  expression?: string
  rate?: string
  target: string
}

export interface Topic extends BaseResource {
  subscriberCount: number
  subscribers: Map<string, number>
}

export type Service = BaseResource

export interface History {
  apis: ApiHistoryItem[]
  schedules: EventHistoryItem[]
  topics: EventHistoryItem[]
}

export interface WebSocket extends BaseResource {
  events: ('connect' | 'disconnect' | 'message')[]
}

export interface WebSocket {
  connectionCount: number
  targets: Map<string, string>
  messages: {
    data: string
    time: string
    connectionId: string
  }[]
}

export interface WebSocketsInfo {
  [socket: string]: WebSocket
}

export interface Bucket extends BaseResource {
  notificationCount: number
  notifiers: Map<string, number>
}

type ResourceType = 'bucket' | 'topic' | 'websocket' | 'collection' | 'secret'

type Notification = {
  bucket: string
  target: string
}

type Subscriber = {
  topic: string
  target: string
}

interface Resource {
  name: string
  type: ResourceType
}
export interface Policy extends BaseResource {
  principals: Resource[]
  actions: string[]
  resources: Resource[]
}
export interface WebSocketResponse {
  projectName: string
  buckets: Bucket[]
  apis: Api[]
  schedules: Schedule[]
  notifications: Notification[]
  subscriptions: Subscriber[]
  topics: Topic[]
  services: Service[]
  // subscriptions: string[];
  websockets: WebSocket[]
  policies: {
    [name: string]: Policy
  }
  triggerAddress: string
  apiAddresses: Record<string, string>
  websocketAddresses: Record<string, string>
  storageAddress: string // has http:// prefix
  currentVersion: string
  latestVersion: string
  connected: boolean
}

export interface Param {
  path: string
  value: OpenAPIV3.ParameterObject[]
}

export interface Endpoint {
  id: string
  api: string
  path: string
  method: Method
  params?: Param[]
  doc: Api['spec']
}

export interface APIRequest {
  path?: string
  method?: Method
  pathParams: FieldRow[] | []
  queryParams: FieldRow[]
  headers: FieldRow[]
  body?: BodyInit | null
}

export interface APIResponse {
  data?: any
  status?: number
  time?: number
  size?: number
  headers?: Record<string, string>
}

export interface BucketFile {
  key: string
}

// HISTORY //

/** Used only in local storage to store the last used params in a request */
export interface LocalStorageHistoryItem {
  request: APIRequest
  JSONBody: string
}

/** History that is received from the CLI web socket */
export interface HistoryItem<T> {
  time: number
  event: T
}

export type EventHistoryItem = TopicHistoryItem | ScheduleHistoryItem

export type TopicHistoryItem = HistoryItem<{
  name: string
  payload: string
  success: boolean
}>

export type ScheduleHistoryItem = HistoryItem<{
  name: string
  success: boolean
}>

export type ApiHistoryItem = HistoryItem<{
  api: string
  request: RequestHistory
  response: APIResponse
}>

export interface RequestHistory {
  path?: string
  method?: Method
  pathParams: FieldRow[] | []
  queryParams: FieldRow[] | []
  headers: Record<string, string[]>
  body?: BodyInit | null
}
