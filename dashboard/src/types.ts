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
export interface Schedule extends WorkerResource {}

export interface Subscription extends WorkerResource {}

export interface Topic extends Schedule {}

export interface History {
  apis: ApiHistoryItem[];
  schedules: EventHistoryItem[];
  topics: EventHistoryItem[];
}

export interface WebSocketResponse {
  projectName: string;
  buckets: string[];
  apis: APIDoc[];
  schedules: WorkerResource[];
  topics: WorkerResource[];
  subscriptions: WorkerResource[];
  triggerAddress: string;
  apiAddresses: Record<string, string>;
  storageAddress: string; // has http:// prefix
}

export interface Param {
  path: string;
  value: OpenAPIV3.ParameterObject[];
}

export interface Endpoint {
  id: string;
  api: string;
  path: string;
  methods: Method[];
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
  Key: string;
}

// HISTORY //

/** Used only in local storage to store the last used params in a request */
export interface LocalStorageHistoryItem {
  request: APIRequest;
  JSONBody: string;
}

/** History that is received from the CLI web socket */
export interface HistoryItem {
  time: number;
  success: boolean;
}

export interface EventHistoryItem extends HistoryItem {
  event: WorkerResource;
}

export interface ApiHistoryItem extends HistoryItem {
  api: string;
  request: RequestHistory;
  response: ResponseHistory;
}

export type RequestHistory = APIRequest;

export type ResponseHistory = APIResponse;
