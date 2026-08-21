import axios, { AxiosInstance, AxiosRequestConfig } from "axios";

export type Role = "agent_developer" | "tool_operator" | "security_reviewer" | "compliance_auditor";
export type CohortStatus = "draft" | "active" | "closed";
export type TraineeStatus = "active" | "suspended" | "completed";

export interface Principal { user_id: string; email: string; display_name: string; role: Role; }
export interface Cohort {
  id: string; code: string; name: string; grade: string; instructor: string; workspace_id?: string;
  capacity: number; student_count: number; status: CohortStatus; version: number;
  created_at: string; updated_at: string;
}
export interface Trainee {
  id: string; student_no: string; name: string; gender?: string; birth_date?: string; phone?: string;
  email: string; cohort_id: string; cohort_name?: string; status: TraineeStatus; version: number;
  created_at: string; updated_at: string;
}
export interface Page<T> { records: T[]; total: number; current: number; size: number; }
export interface TrainingSummary { cohorts: number; active_cohorts: number; trainees: number; active_trainees: number; }
export interface ApiError { error?: { code: string; message: string; request_id: string }; message?: string; }

export type ClientErrorKind = "cancelled" | "timeout" | "server" | "network";

export class ClientError extends Error {
  code?: string;
  requestId?: string;
  status?: number;
  kind: ClientErrorKind;

  constructor(message: string, kind: ClientErrorKind) {
    super(message);
    this.name = "ClientError";
    this.kind = kind;
  }
}

export type ExecutionRequestState = "submitted" | "authorized" | "executing" | "completed" | "archived" | "cancelled";
export interface ExecutionRequest {
  id: string; workspace_id: string; requester_zone_id: string; execution_zone_id: string; execution_pool_id: string;
  request_key: string; state: ExecutionRequestState; scheduled_start_at: string; expected_finish_at: string;
  total_requested_units: number; version: number; created_at: string; updated_at: string;
}
export interface ScenarioInput {
  name: string; protocol_family: string; operation_count: number; requested_units: number; duration_minutes: number;
}
export interface ScenarioResult { workspace: { id: string; name: string }; tool_revision: { id: string; version_tag: string }; request: ExecutionRequest; }
export interface ExecutionPage { items: ExecutionRequest[]; total: number; }

export const client: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_URL || "/api/v1",
  timeout: 12000,
});

client.interceptors.request.use((config) => {
  const token = localStorage.getItem("token");
  if (token) config.headers.Authorization = "Bearer " + token;
  return config;
});

client.interceptors.response.use((response) => response, (error) => {
  if (axios.isCancel(error) || error.code === "ERR_CANCELED") {
    return Promise.reject(new ClientError("请求已取消", "cancelled"));
  }
  if (error.code === "ECONNABORTED") {
    return Promise.reject(new ClientError("请求超时，请重试", "timeout"));
  }
  const payload = error.response?.data as ApiError | undefined;
  const message = payload?.error?.message || payload?.message || (error.response ? "服务暂时不可用，请稍后重试" : "网络连接失败，请检查连接后重试");
  const enriched = new ClientError(message, error.response ? "server" : "network");
  enriched.code = payload?.error?.code;
  enriched.requestId = payload?.error?.request_id;
  enriched.status = error.response?.status;
  return Promise.reject(enriched);
});

export function get<T>(url: string, config?: AxiosRequestConfig) { return client.get<T>(url, config).then(r => r.data); }
export function post<T>(url: string, body?: unknown, config?: AxiosRequestConfig) { return client.post<T>(url, body, config).then(r => r.data); }
export function put<T>(url: string, body?: unknown, config?: AxiosRequestConfig) { return client.put<T>(url, body, config).then(r => r.data); }
export function del<T>(url: string, config?: AxiosRequestConfig) { return client.delete<T>(url, config).then(r => r.data); }

export async function listExecutionRequests(params: Record<string, unknown>, signal?: AbortSignal): Promise<ExecutionPage> {
  const page = await get<{ Items?: ExecutionRequest[]; Total?: number; items?: ExecutionRequest[]; total?: number }>("/execution-requests", { params, signal });
  return { items: page.items || page.Items || [], total: page.total ?? page.Total ?? 0 };
}

export const api = {
  login: (email: string, password: string) => post<{ token: string; expires_at: string; user: Principal }>("/auth/login", { email, password }),
  me: () => get<Principal>("/auth/me"),
  logout: () => post<void>("/auth/logout"),
  cohorts: {
    page: (params: Record<string, unknown>) => get<Page<Cohort>>("/cohorts", { params }),
    all: () => get<Cohort[]>("/cohorts/all"),
    create: (body: Partial<Cohort>) => post<Cohort>("/cohorts", body),
    update: (id: string, body: Partial<Cohort>) => put<Cohort>("/cohorts/" + id, body),
    remove: (id: string, version: number) => del<void>("/cohorts/" + id, { params: { version } }),
  },
  trainees: {
    page: (params: Record<string, unknown>) => get<Page<Trainee>>("/trainees", { params }),
    create: (body: Partial<Trainee>) => post<Trainee>("/trainees", body),
    update: (id: string, body: Partial<Trainee>) => put<Trainee>("/trainees/" + id, body),
    remove: (id: string, version: number) => del<void>("/trainees/" + id, { params: { version } }),
  },
  trainingSummary: () => get<TrainingSummary>("/training/summary"),
  execution: {
    page: listExecutionRequests,
    createScenario: (body: ScenarioInput, idempotencyKey: string) => post<ScenarioResult>("/training/execution-scenarios", body, { headers: { "Idempotency-Key": idempotencyKey } }),
    transition: (id: string, action: "authorize" | "begin" | "complete" | "archive") => post<ExecutionRequest>(`/execution-requests/${id}/${action}`),
    cancel: (id: string, note: string) => post<ExecutionRequest>(`/execution-requests/${id}/cancel`, { note }),
  },
  requests: () => listExecutionRequests({ limit: 8, offset: 0 }),
  summary: () => get<Record<string, number>>("/summary"),
};
