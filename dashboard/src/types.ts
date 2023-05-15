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

export interface Schedule {
  workerKey: string;
  topicKey: string;
}

export interface WebSocketResponse {
  projectName: string;
  buckets: string[];
  apis: APIDoc[];
  schedules: Schedule[];
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

export interface HistoryItem {
  request: APIRequest;
  JSONBody: string;
}

export interface RequestHistoryItem {
  endpoint: Endpoint;
  request: APIRequest;
  response: APIResponse;
  time: number;
}

export interface ScheduleHistoryItem {
  schedule: Schedule;
  success: boolean;
  time: number;
}

export interface BucketFile {
  Key: string;
}
