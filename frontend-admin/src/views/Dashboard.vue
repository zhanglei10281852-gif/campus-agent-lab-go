<template>
  <div class="page-container dashboard">
    <section class="welcome-section">
      <div><p class="eyebrow">CAMPUS AGENT LAB</p><h1>实训运行总览</h1><p class="welcome-copy">把班级、学员和智能体工具放在同一条可追溯的工作流里。</p></div>
      <div class="welcome-date"><el-icon><Calendar /></el-icon><span>{{ currentDate }}</span></div>
    </section>
    <el-alert v-if="errorMessage" :title="errorMessage" type="error" show-icon closable @close="errorMessage = ''" />
    <el-row :gutter="16" class="stats-row">
      <el-col :xs="12" :sm="6" v-for="item in stats" :key="item.title">
        <el-card shadow="never" class="stat-card"><div class="stat-icon" :class="item.tone"><el-icon :size="22"><component :is="item.icon" /></el-icon></div><div class="stat-value">{{ loading ? '—' : item.value }}</div><div class="stat-title">{{ item.title }}</div></el-card>
      </el-col>
    </el-row>
    <el-row :gutter="16">
      <el-col :xs="24" :lg="14"><el-card shadow="never" class="data-card">
        <template #header><div class="card-header"><span><el-icon><User /></el-icon> 最近加入实训的学员</span><el-button link type="primary" @click="$router.push('/students')">查看全部 <el-icon><ArrowRight /></el-icon></el-button></div></template>
        <el-table v-loading="loading" :data="recentTrainees" empty-text="还没有学员记录">
          <el-table-column prop="student_no" label="编号" width="130" /><el-table-column prop="name" label="姓名" min-width="120" />
          <el-table-column prop="cohort_name" label="实训班级" min-width="150"><template #default="{ row }"><el-tag type="info" effect="plain">{{ row.cohort_name || '未分组' }}</el-tag></template></el-table-column>
          <el-table-column prop="status" label="状态" width="100"><template #default="{ row }"><el-tag :type="row.status === 'active' ? 'success' : 'warning'">{{ statusLabel(row.status) }}</el-tag></template></el-table-column>
        </el-table>
      </el-card></el-col>
      <el-col :xs="24" :lg="10"><el-card shadow="never" class="data-card">
        <template #header><div class="card-header"><span><el-icon><Connection /></el-icon> 执行队列</span><el-button link type="primary" @click="refresh">刷新</el-button></div></template>
        <el-skeleton v-if="loading" :rows="4" animated /><el-empty v-else-if="!requests.length" description="当前没有待处理执行" :image-size="70" />
        <div v-else class="request-list"><div v-for="item in requests" :key="item.id" class="request-item"><div><strong>{{ item.request_key || item.id }}</strong><span>{{ item.state }}</span></div><el-tag size="small" :type="requestTone(item.state)">{{ requestLabel(item.state) }}</el-tag></div></div>
      </el-card></el-col>
    </el-row>
  </div>
</template>
<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Calendar, User, School, Connection, DataAnalysis, ArrowRight } from '@element-plus/icons-vue'
import { api } from '@/api/client'
const loading = ref(true), errorMessage = ref(''), recentTrainees = ref([]), requests = ref([])
const summary = ref({ cohorts: 0, active_cohorts: 0, trainees: 0, active_trainees: 0 })
const currentDate = computed(() => new Intl.DateTimeFormat('zh-CN', { dateStyle: 'full' }).format(new Date()))
const stats = computed(() => [{ title:'实训班级', value:summary.value.cohorts, icon:School, tone:'blue' }, { title:'进行中的班级', value:summary.value.active_cohorts, icon:DataAnalysis, tone:'green' }, { title:'实训学员', value:summary.value.trainees, icon:User, tone:'amber' }, { title:'活跃学员', value:summary.value.active_trainees, icon:Connection, tone:'red' }])
const statusLabel = (s) => ({active:'进行中',suspended:'已暂停',completed:'已完成'}[s] || s)
const requestLabel = (s) => ({submitted:'待授权',authorized:'已授权',executing:'执行中',completed:'已完成',cancelled:'已取消'}[s] || s)
const requestTone = (s) => s === 'executing' ? 'warning' : s === 'completed' ? 'success' : 'info'
async function refresh() { loading.value=true; errorMessage.value=''; try { const [training, trainees, execution] = await Promise.all([api.trainingSummary(), api.trainees.page({current:1,size:6,sort:'created_at',desc:true}), api.requests()]); summary.value=training; recentTrainees.value=trainees.records; requests.value=execution.items || execution.Items || [] } catch (e) { errorMessage.value=e.message || '数据加载失败'; ElMessage.error(errorMessage.value) } finally { loading.value=false } }
onMounted(refresh)
</script>
<style scoped lang="scss">
.dashboard{padding:24px}.welcome-section{display:flex;justify-content:space-between;align-items:center;margin-bottom:18px;padding:24px 28px;background:#172554;border-radius:10px;color:#fff}.eyebrow{margin:0 0 8px;font-size:11px;letter-spacing:1.4px;opacity:.72}h1{margin:0;font-size:25px}.welcome-copy{margin:8px 0 0;color:#c7d2fe;font-size:14px}.welcome-date{display:flex;align-items:center;gap:8px;padding:10px 14px;background:rgba(255,255,255,.12);border-radius:6px;font-size:13px;white-space:nowrap}.stats-row{margin:16px 0}.stat-card{border:1px solid #e5e7eb;min-height:125px}.stat-icon{width:40px;height:40px;border-radius:8px;display:flex;align-items:center;justify-content:center;color:#fff}.blue{background:#2563eb}.green{background:#059669}.amber{background:#d97706}.red{background:#dc2626}.stat-value{margin-top:13px;font-size:27px;font-weight:700;color:#111827}.stat-title{margin-top:3px;color:#6b7280;font-size:13px}.data-card{margin-bottom:16px;border:1px solid #e5e7eb}.card-header{display:flex;align-items:center;justify-content:space-between;font-weight:600}.card-header>span{display:flex;align-items:center;gap:7px}.request-item{display:flex;justify-content:space-between;align-items:center;padding:13px 4px;border-bottom:1px solid #f1f5f9}.request-item strong{display:block;font-size:13px}.request-item span{display:block;margin-top:4px;color:#64748b;font-size:12px}@media(max-width:680px){.dashboard{padding:14px}.welcome-section{align-items:flex-start;flex-direction:column;gap:16px}}
</style>
