import { afterEach, describe, expect, it, vi } from "vitest";
import { AxiosError, CanceledError, type AxiosAdapter, type AxiosRequestConfig, type AxiosResponse } from "axios";
import { api, client } from "./client";

const originalAdapter = client.defaults.adapter;

function responseAdapter(data: unknown, status = 200): AxiosAdapter {
  return async (config): Promise<AxiosResponse> => ({ data, status, statusText: "OK", headers: {}, config });
}

function rejectedAdapter(status: number, data: unknown): AxiosAdapter {
  return async (config) => {
    const response: AxiosResponse = { data, status, statusText: "Error", headers: {}, config };
    throw new AxiosError("request failed", "ERR_BAD_RESPONSE", config, undefined, response);
  };
}

afterEach(() => {
  client.defaults.adapter = originalAdapter;
  localStorage.clear();
  vi.restoreAllMocks();
});

describe("typed API client", () => {
  it("adds the session token and normalizes execution pagination", async () => {
    localStorage.setItem("token", "session-token");
    const adapter = vi.fn(responseAdapter({ Items: [{ id: "run_1", state: "submitted" }], Total: 1 }));
    client.defaults.adapter = adapter;
    const page = await api.execution.page({ limit: 10, offset: 0 });
    expect(page.total).toBe(1);
    expect(page.items[0].id).toBe("run_1");
    expect((adapter.mock.calls[0][0] as AxiosRequestConfig).headers?.Authorization).toBe("Bearer session-token");
  });

  it("maps structured server errors with code, request ID and status", async () => {
    client.defaults.adapter = rejectedAdapter(409, { error: { code: "version_conflict", message: "请刷新后重试", request_id: "req_test" } });
    await expect(api.trainingSummary()).rejects.toMatchObject({
      name: "ClientError", kind: "server", code: "version_conflict", requestId: "req_test", status: 409,
    });
  });

  it("aborts an obsolete list request", async () => {
    client.defaults.adapter = (config) => new Promise((_resolve, reject) => {
      config.signal?.addEventListener?.("abort", () => reject(new CanceledError("cancelled", config)));
    });
    const controller = new AbortController();
    const pending = api.execution.page({ limit: 10 }, controller.signal);
    controller.abort();
    await expect(pending).rejects.toEqual(expect.objectContaining({ kind: "cancelled", message: "请求已取消" }));
  });

  it("distinguishes request timeout from a general network failure", async () => {
    client.defaults.adapter = async (config) => { throw new AxiosError("timeout", "ECONNABORTED", config); };
    await expect(api.trainingSummary()).rejects.toMatchObject({ kind: "timeout", message: "请求超时，请重试" });
  });
});
