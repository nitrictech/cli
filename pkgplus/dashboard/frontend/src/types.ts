import type { OpenAPIV3 } from "openapi-types";
import type { FieldRow } from "./components/shared/FieldRows";

export type APIDoc = OpenAPIV3.Document;

export type Method =
  | "GET"
  | "PUT"
  | "POST"
  | "DELETE"
  | "OPTIONS"
  | "HEAD"
  | "PATCH"
  | "TRACE";

export interface WorkerResource {
  workerKey: string;
  topicKey: string;
}
export type Schedule = WorkerResource;

export type Subscription = WorkerResource;

export type Topic = Schedule;

export interface History {
  apis: ApiHistoryItem[];
  schedules: EventHistoryItem[];
  topics: EventHistoryItem[];
}

export interface WebSocket {
  name: string;
  events: ("connect" | "disconnect" | "message")[];
}

export interface WebSocketInfo {
  connectionCount: number;
  messages: {
    data: string;
    time: string;
    connectionId: string;
  }[];
}

export interface WebSocketsInfo {
  [socket: string]: WebSocketInfo;
}
export interface WebSocketResponse {
  projectName: string;
  buckets: string[];
  apis: APIDoc[];
  schedules: WorkerResource[];
  topics: WorkerResource[];
  subscriptions: WorkerResource[];
  websockets: WebSocket[];
  triggerAddress: string;
  apiAddresses: Record<string, string>;
  websocketAddresses: Record<string, string>;
  storageAddress: string; // has http:// prefix
  currentVersion: string;
  latestVersion: string;
  connected: boolean;
}

export interface Param {
  path: string;
  value: OpenAPIV3.ParameterObject[];
}

export interface Endpoint {
  id: string;
  api: string;
  path: string;
  method: Method;
  params?: Param[];
  doc: OpenAPIV3.Document<Record<string, any>>;
}

export interface APIRequest {
  path?: string;
  method?: Method;
  pathParams: FieldRow[] | [];
  queryParams: FieldRow[];
  headers: FieldRow[];
  body?: BodyInit | null;
}

export interface APIResponse {
  data?: any;
  status?: number;
  time?: number;
  size?: number;
  headers?: Record<string, string>;
}

export interface BucketFile {
  key: string;
}

export interface TopicRequest {
  id: string;
  payloadType: string;
  payload: Record<string, string>;
}

// HISTORY //

/** Used only in local storage to store the last used params in a request */
export interface LocalStorageHistoryItem {
  request: APIRequest;
  JSONBody: string;
}

/** History that is received from the CLI web socket */
export interface HistoryItem<T> {
  time: number;
  event: T;
}

export type EventHistoryItem = HistoryItem<{
  event: WorkerResource;
  payload: string;
}>;

export type ApiHistoryItem = HistoryItem<{
  api: string;
  request: RequestHistory;
  response: APIResponse;
}>;

export interface RequestHistory {
  path?: string;
  method?: Method;
  pathParams: FieldRow[] | [];
  queryParams: FieldRow[] | [];
  headers: Record<string, string[]>;
  body?: BodyInit | null;
}
