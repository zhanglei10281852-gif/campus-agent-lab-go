<template>
  <div class="page-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="header-info">
        <h2>操作日志</h2>
        <p>查看系统中的所有操作记录</p>
      </div>
    </div>
    
    <!-- 搜索区域 -->
    <el-card class="search-card" shadow="never">
      <el-form :inline="true" :model="searchForm" class="search-form">
        <el-form-item label="操作人">
          <el-input v-model="searchForm.username" placeholder="请输入操作人" clearable :prefix-icon="User" />
        </el-form-item>
        <el-form-item label="模块">
          <el-select v-model="searchForm.module" placeholder="全部模块" clearable style="width: 140px">
            <el-option label="学生管理" value="学生管理" />
            <el-option label="班级管理" value="班级管理" />
            <el-option label="用户管理" value="用户管理" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchData" :icon="Search">搜索</el-button>
          <el-button @click="resetSearch" :icon="RefreshRight">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 日志列表 -->
    <el-card class="table-card" shadow="never">
      <el-table :data="tableData" v-loading="loading" row-key="id">
        <el-table-column prop="username" label="操作人" width="140">
          <template #default="{ row }">
            <div class="operator-cell">
              <el-avatar :size="32" class="operator-avatar">
                {{ row.username?.charAt(0).toUpperCase() }}
              </el-avatar>
              <span>{{ row.username }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="module" label="模块" width="120">
          <template #default="{ row }">
            <el-tag :type="getModuleType(row.module)" effect="light" size="small">
              {{ row.module }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="action" label="操作" width="140">
          <template #default="{ row }">
            <span class="action-text">{{ row.action }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="method" label="方法" min-width="280">
          <template #default="{ row }">
            <el-tooltip :content="row.method" placement="top" :show-after="500">
              <code class="method-code">{{ formatMethod(row.method) }}</code>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="ip" label="IP地址" width="140">
          <template #default="{ row }">
            <span class="ip-text">{{ row.ip || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="costTime" label="耗时" width="100" align="center">
          <template #default="{ row }">
            <el-tag 
              :type="getCostTimeType(row.costTime)" 
              effect="plain" 
              size="small"
              class="cost-tag"
            >
              {{ row.costTime }}ms
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createTime" label="操作时间" width="180">
          <template #default="{ row }">
            <div class="time-cell">
              <el-icon><Clock /></el-icon>
              <span>{{ row.createTime }}</span>
            </div>
          </template>
        </el-table-column>
      </el-table>
      
      <div class="pagination-wrapper">
        <div class="pagination-info">
          共 <span class="total">{{ pagination.total }}</span> 条记录
        </div>
        <el-pagination 
          v-model:current-page="pagination.current" 
          v-model:page-size="pagination.size"
          :total="pagination.total" 
          :page-sizes="[10, 20, 50, 100]" 
          layout="sizes, prev, pager, next"
          @size-change="fetchData" 
          @current-change="fetchData" 
        />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { Search, RefreshRight, User, Clock } from '@element-plus/icons-vue'
import { getLogPage } from '@/api/log'

const loading = ref(false)
const tableData = ref([])

const searchForm = reactive({ username: '', module: '' })
const pagination = reactive({ current: 1, size: 10, total: 0 })

const fetchData = async () => {
  loading.value = true
  try {
    const data = await getLogPage({ ...searchForm, current: pagination.current, size: pagination.size })
    tableData.value = data.records || []
    pagination.total = data.total || 0
  } finally {
    loading.value = false
  }
}

const resetSearch = () => {
  Object.assign(searchForm, { username: '', module: '' })
  pagination.current = 1
  fetchData()
}

const getModuleType = (module) => {
  const types = {
    '学生管理': 'primary',
    '班级管理': 'success',
    '用户管理': 'warning'
  }
  return types[module] || 'info'
}

const getCostTimeType = (time) => {
  if (time < 100) return 'success'
  if (time < 500) return 'warning'
  return 'danger'
}

const formatMethod = (method) => {
  if (!method) return '-'
  const parts = method.split('.')
  if (parts.length >= 2) {
    return `${parts[parts.length - 2]}.${parts[parts.length - 1]}`
  }
  return method
}

onMounted(() => fetchData())
</script>

<style lang="scss" scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  
  .header-info {
    h2 {
      font-size: 20px;
      font-weight: 700;
      color: var(--text-primary);
      margin-bottom: 4px;
    }
    
    p {
      font-size: 14px;
      color: var(--text-secondary);
    }
  }
}

.search-card {
  margin-bottom: 20px;
  
  .search-form {
    .el-form-item {
      margin-bottom: 0;
      margin-right: 12px;
    }
  }
}

.table-card {
  :deep(.el-card__body) {
    padding: 0;
  }
}

.operator-cell {
  display: flex;
  align-items: center;
  gap: 10px;
  
  .operator-avatar {
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    color: #fff;
    font-size: 12px;
    font-weight: 600;
    flex-shrink: 0;
  }
}

.action-text {
  font-weight: 500;
  color: var(--text-primary);
}

.method-code {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 12px;
  color: var(--text-secondary);
  background: var(--bg-color-secondary);
  padding: 4px 8px;
  border-radius: 4px;
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ip-text {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 13px;
  color: var(--text-secondary);
}

.cost-tag {
  font-family: 'SF Mono', Monaco, monospace;
  font-weight: 500;
}

.time-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-secondary);
  
  .el-icon {
    color: var(--text-muted);
  }
}

.pagination-wrapper {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
  
  .pagination-info {
    font-size: 13px;
    color: var(--text-secondary);
    
    .total {
      font-weight: 600;
      color: var(--el-color-primary);
    }
  }
}
</style>
