<template>
  <div class="page-container execution-page">
    <div class="page-header">
      <div>
        <h2>执行治理</h2>
        <p>创建实训执行场景，并推进授权、运行与归档状态。</p>
      </div>
      <el-button type="primary" :icon="Plus" data-testid="open-scenario" @click="openScenarioDialog">发起实训执行</el-button>
    </div>

    <el-alert v-if="loadError" class="page-alert" :title="loadError" type="error" show-icon :closable="false">
      <template #default><el-button link type="danger" data-testid="retry-list" @click="loadRequests">重新加载</el-button></template>
    </el-alert>

    <section class="filter-band" aria-label="执行筛选">
      <el-select v-model="filters.state" placeholder="全部状态" clearable aria-label="执行状态" @change="resetAndLoad">
        <el-option v-for="item in states" :key="item.value" :label="item.label" :value="item.value" />
      </el-select>
      <el-button :icon="Refresh" :loading="loading" @click="loadRequests">刷新</el-button>
    </section>

    <section class="table-surface">
      <el-table v-loading="loading" :data="rows" row-key="id" empty-text="还没有执行记录" data-testid="execution-table">
        <el-table-column prop="request_key" label="执行编号" min-width="180" />
        <el-table-column label="状态" width="112">
          <template #default="{ row }"><el-tag :type="stateTone(row.state)">{{ stateLabel(row.state) }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="total_requested_units" label="资源单元" width="110" align="right" />
        <el-table-column label="计划时间" min-width="220">
          <template #default="{ row }"><span class="time-range">{{ formatTime(row.scheduled_start_at) }}<br>{{ formatTime(row.expected_finish_at) }}</span></template>
        </el-table-column>
        <el-table-column label="操作" min-width="210" fixed="right">
          <template #default="{ row }">
            <div class="row-actions">
              <el-button v-if="nextAction(row.state)" size="small" type="primary" :loading="transitioningId === row.id" :data-testid="`advance-${row.id}`" @click="advance(row)">
                {{ nextAction(row.state)?.label }}
              </el-button>
              <el-button v-if="row.state === 'submitted' || row.state === 'authorized'" size="small" type="danger" plain :loading="transitioningId === row.id" @click="cancel(row)">取消</el-button>
              <span v-if="!nextAction(row.state) && row.state !== 'submitted' && row.state !== 'authorized'" class="terminal-text">流程已结束</span>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-row">
        <span>共 {{ total }} 条</span>
        <el-pagination v-model:current-page="page" v-model:page-size="pageSize" :total="total" :page-sizes="[10, 20, 50]" layout="sizes, prev, pager, next" @change="loadRequests" />
      </div>
    </section>

    <el-dialog v-model="dialogVisible" title="发起实训执行" width="min(560px, 94vw)" destroy-on-close @closed="resetScenarioForm">
      <el-alert v-if="submitError" :title="submitError" :type="submitErrorKind === 'conflict' ? 'warning' : 'error'" show-icon :closable="false" class="dialog-alert">
        <template #default><el-button link :type="submitErrorKind === 'conflict' ? 'warning' : 'danger'" data-testid="retry-submit" @click="submitScenario">重试提交</el-button></template>
      </el-alert>
      <el-form ref="scenarioFormRef" :model="scenarioForm" :rules="scenarioRules" label-position="top" data-testid="scenario-form">
        <el-form-item label="实训场景名称" prop="name"><el-input v-model="scenarioForm.name" maxlength="120" show-word-limit placeholder="例如：工具调用安全复核" @input="scenarioValidationError = ''" /></el-form-item>
        <p v-if="scenarioValidationError" class="validation-message" role="alert">{{ scenarioValidationError }}</p>
        <el-form-item label="工具协议" prop="protocol_family">
          <el-select v-model="scenarioForm.protocol_family" style="width:100%"><el-option label="MCP" value="mcp" /><el-option label="OpenAPI" value="openapi" /><el-option label="Function Calling" value="function-calling" /></el-select>
        </el-form-item>
        <div class="number-grid">
          <el-form-item label="操作数量" prop="operation_count"><el-input-number v-model="scenarioForm.operation_count" :min="1" :max="5000" controls-position="right" /></el-form-item>
          <el-form-item label="资源单元" prop="requested_units"><el-input-number v-model="scenarioForm.requested_units" :min="1" :max="100000" controls-position="right" /></el-form-item>
          <el-form-item label="执行时长（分钟）" prop="duration_minutes"><el-input-number v-model="scenarioForm.duration_minutes" :min="15" :max="480" :step="15" controls-position="right" /></el-form-item>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" data-testid="submit-scenario" @click="submitScenario">创建并提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from "vue";
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from "element-plus";
import { Plus, Refresh } from "@element-plus/icons-vue";
import { api, ClientError, type ExecutionRequest, type ExecutionRequestState, type ScenarioInput } from "@/api/client";

const states: Array<{ label: string; value: ExecutionRequestState }> = [
  { label: "待授权", value: "submitted" }, { label: "已授权", value: "authorized" },
  { label: "执行中", value: "executing" }, { label: "已完成", value: "completed" },
  { label: "已归档", value: "archived" }, { label: "已取消", value: "cancelled" },
];
const filters = reactive<{ state: ExecutionRequestState | "" }>({ state: "" });
const rows = ref<ExecutionRequest[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = ref(10);
const loading = ref(false);
const loadError = ref("");
let listController: AbortController | undefined;

const dialogVisible = ref(false);
const scenarioFormRef = ref<FormInstance>();
const scenarioForm = reactive<ScenarioInput>({ name: "", protocol_family: "mcp", operation_count: 12, requested_units: 240, duration_minutes: 90 });
const scenarioRules: FormRules<ScenarioInput> = {
  name: [{ required: true, message: "请输入实训场景名称", trigger: "blur" }],
  protocol_family: [{ required: true, message: "请选择工具协议", trigger: "change" }],
};
const submitting = ref(false);
const submitError = ref("");
const scenarioValidationError = ref("");
const submitErrorKind = ref<"conflict" | "other">("other");
const idempotencyKey = ref("");
const transitioningId = ref("");

function resetAndLoad() { page.value = 1; void loadRequests(); }
async function loadRequests() {
  listController?.abort();
  listController = new AbortController();
  loading.value = true;
  loadError.value = "";
  try {
    const result = await api.execution.page({ limit: pageSize.value, offset: (page.value - 1) * pageSize.value, state: filters.state || undefined }, listController.signal);
    rows.value = result.items;
    total.value = result.total;
  } catch (error) {
    if (error instanceof ClientError && error.kind === "cancelled") return;
    loadError.value = error instanceof Error ? error.message : "执行记录加载失败";
  } finally {
    loading.value = false;
  }
}

function openScenarioDialog() {
  resetScenarioForm();
  idempotencyKey.value = crypto.randomUUID();
  dialogVisible.value = true;
}
function resetScenarioForm() {
  Object.assign(scenarioForm, { name: "", protocol_family: "mcp", operation_count: 12, requested_units: 240, duration_minutes: 90 });
  submitError.value = "";
  scenarioValidationError.value = "";
  scenarioFormRef.value?.clearValidate();
}
async function submitScenario() {
  if (!scenarioFormRef.value) return;
  if (!scenarioForm.name.trim()) {
    scenarioValidationError.value = "请输入实训场景名称";
    await scenarioFormRef.value.validateField("name").catch(() => undefined);
    return;
  }
  scenarioValidationError.value = "";
  if (!(await scenarioFormRef.value.validate().catch(() => false))) return;
  if (!idempotencyKey.value) idempotencyKey.value = crypto.randomUUID();
  submitting.value = true;
  submitError.value = "";
  try {
    const result = await api.execution.createScenario({ ...scenarioForm }, idempotencyKey.value);
    rows.value = [result.request, ...rows.value.filter((item) => item.id !== result.request.id)];
    total.value = Math.max(total.value + 1, rows.value.length);
    dialogVisible.value = false;
    ElMessage.success("执行场景已提交，等待授权");
  } catch (error) {
    const clientError = error as ClientError;
    submitErrorKind.value = clientError.status === 409 || clientError.code === "conflict" || clientError.code === "version_conflict" ? "conflict" : "other";
    submitError.value = error instanceof Error ? error.message : "场景提交失败";
  } finally {
    submitting.value = false;
  }
}

const transitions: Partial<Record<ExecutionRequestState, { action: "authorize" | "begin" | "complete" | "archive"; label: string }>> = {
  submitted: { action: "authorize", label: "授权" }, authorized: { action: "begin", label: "开始执行" },
  executing: { action: "complete", label: "完成" }, completed: { action: "archive", label: "归档" },
};
function nextAction(state: ExecutionRequestState) { return transitions[state]; }
async function advance(row: ExecutionRequest) {
  const next = nextAction(row.state);
  if (!next) return;
  transitioningId.value = row.id;
  try {
    const updated = await api.execution.transition(row.id, next.action);
    Object.assign(row, updated);
    ElMessage.success(`执行请求已${next.label}`);
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : "状态更新失败");
  } finally {
    transitioningId.value = "";
  }
}
async function cancel(row: ExecutionRequest) {
  await ElMessageBox.confirm("取消后会释放执行池和工具版本，是否继续？", "取消执行", { type: "warning" });
  transitioningId.value = row.id;
  try {
    const updated = await api.execution.cancel(row.id, "由实训管理员取消");
    Object.assign(row, updated);
    ElMessage.success("执行请求已取消");
  } finally {
    transitioningId.value = "";
  }
}

function stateLabel(state: ExecutionRequestState) { return states.find((item) => item.value === state)?.label || state; }
function stateTone(state: ExecutionRequestState) { return state === "executing" ? "warning" : state === "completed" || state === "archived" ? "success" : state === "cancelled" ? "danger" : "info"; }
function formatTime(value: string) { return new Intl.DateTimeFormat("zh-CN", { dateStyle: "short", timeStyle: "short", hour12: false }).format(new Date(value)); }

onMounted(loadRequests);
onBeforeUnmount(() => listController?.abort());
</script>

<style scoped lang="scss">
.execution-page { padding: 24px; }
.page-header { display:flex; justify-content:space-between; align-items:center; gap:16px; margin-bottom:18px; }
.page-header h2 { margin:0 0 5px; font-size:21px; color:#111827; }
.page-header p { margin:0; color:#64748b; font-size:14px; }
.page-alert,.dialog-alert { margin-bottom:16px; }
.filter-band { display:flex; align-items:center; gap:10px; padding:14px 0; border-top:1px solid #e5e7eb; }
.filter-band .el-select { width:180px; }
.table-surface { background:#fff; border:1px solid #e5e7eb; border-radius:8px; overflow:hidden; }
.pagination-row { display:flex; align-items:center; justify-content:space-between; gap:16px; padding:14px 16px; color:#64748b; font-size:13px; border-top:1px solid #e5e7eb; }
.row-actions { display:flex; align-items:center; gap:8px; min-height:32px; }
.time-range { color:#475569; font-size:12px; line-height:1.65; white-space:nowrap; }
.terminal-text { color:#94a3b8; font-size:13px; }
.number-grid { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:12px; }
.number-grid :deep(.el-input-number) { width:100%; }
.validation-message { margin:-12px 0 16px; color:var(--el-color-danger); font-size:12px; }
@media (max-width: 720px) {
  .execution-page { padding:14px; }
  .page-header { align-items:flex-start; flex-direction:column; }
  .page-header .el-button { width:100%; }
  .number-grid { grid-template-columns:1fr; gap:0; }
  .pagination-row { align-items:flex-start; flex-direction:column; overflow-x:auto; }
  .table-surface { overflow-x:auto; }
  .table-surface :deep(.el-table) { min-width:760px; }
}
</style>
