import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount, type VueWrapper } from "@vue/test-utils";
import ElementPlus from "element-plus";
import ExecutionRequests from "./ExecutionRequests.vue";

const mocks = vi.hoisted(() => ({
  page: vi.fn(), createScenario: vi.fn(), transition: vi.fn(), cancel: vi.fn(),
}));

vi.mock("@/api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/api/client")>();
  return { ...actual, api: { ...actual.api, execution: mocks } };
});

const submitted = {
  id: "run_1", workspace_id: "workspace_1", requester_zone_id: "zone_1", execution_zone_id: "zone_2",
  execution_pool_id: "pool_1", request_key: "RUN-001", state: "submitted", scheduled_start_at: "2026-08-21T09:00:00Z",
  expected_finish_at: "2026-08-21T10:30:00Z", total_requested_units: 240, version: 1,
  created_at: "2026-08-21T08:00:00Z", updated_at: "2026-08-21T08:00:00Z",
} as const;

let wrapper: VueWrapper | undefined;
function click(selector: string) {
  const element = document.querySelector<HTMLElement>(selector);
  if (!element) throw new Error(`missing element ${selector}`);
  element.click();
}

beforeEach(() => {
  mocks.page.mockReset().mockResolvedValue({ items: [], total: 0 });
  mocks.createScenario.mockReset();
  mocks.transition.mockReset();
  mocks.cancel.mockReset();
});

afterEach(() => {
  wrapper?.unmount();
  wrapper = undefined;
  document.body.innerHTML = "";
});

async function mountPage() {
  wrapper = mount(ExecutionRequests, { attachTo: document.body, global: { plugins: [ElementPlus] } });
  await flushPromises();
  return wrapper;
}

describe("execution governance workflow", () => {
  it("shows a route-level load failure and recovers through retry", async () => {
    mocks.page.mockRejectedValueOnce(new Error("执行服务暂时不可用")).mockResolvedValueOnce({ items: [submitted], total: 1 });
    await mountPage();
    expect(document.body.textContent).toContain("执行服务暂时不可用");
    click('[data-testid="retry-list"]');
    await flushPromises();
    expect(document.body.textContent).toContain("RUN-001");
    expect(document.body.textContent).not.toContain("执行服务暂时不可用");
  });

  it("uses Element Plus form validation before submitting a scenario", async () => {
    await mountPage();
    click('[data-testid="open-scenario"]');
    await flushPromises();
    click('[data-testid="submit-scenario"]');
    await flushPromises();
    expect(document.body.textContent).toContain("请输入实训场景名称");
    expect(mocks.createScenario).not.toHaveBeenCalled();
  });

  it("keeps a conflict visible and retries the same idempotent submission", async () => {
    mocks.createScenario
      .mockRejectedValueOnce(Object.assign(new Error("同名场景刚刚被创建"), { status: 409, code: "conflict" }))
      .mockResolvedValueOnce({ workspace: { id: "workspace_1", name: "工具调用复核" }, tool_revision: { id: "revision_1", version_tag: "v1" }, request: submitted });
    await mountPage();
    click('[data-testid="open-scenario"]');
    await flushPromises();
    const nameInput = document.querySelector<HTMLInputElement>('[data-testid="scenario-form"] input');
    if (!nameInput) throw new Error("missing scenario name input");
    nameInput.value = "工具调用复核";
    nameInput.dispatchEvent(new Event("input", { bubbles: true }));
    click('[data-testid="submit-scenario"]');
    await flushPromises();
    expect(document.body.textContent).toContain("同名场景刚刚被创建");
    click('[data-testid="retry-submit"]');
    await flushPromises();
    expect(mocks.createScenario).toHaveBeenCalledTimes(2);
    expect(document.body.textContent).toContain("RUN-001");
  });

  it("advances the public request state from submitted to authorized", async () => {
    mocks.page.mockResolvedValue({ items: [{ ...submitted }], total: 1 });
    mocks.transition.mockResolvedValue({ ...submitted, state: "authorized", version: 2 });
    await mountPage();
    click('[data-testid="advance-run_1"]');
    await flushPromises();
    expect(mocks.transition).toHaveBeenCalledWith("run_1", "authorize");
    expect(document.body.textContent).toContain("已授权");
    expect(document.body.textContent).toContain("开始执行");
  });
});
