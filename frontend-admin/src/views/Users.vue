<template>
  <div class="page-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="header-info">
        <h2>用户管理</h2>
        <p>管理系统中的所有用户账号</p>
      </div>
      <el-button type="primary" @click="openDialog()" :icon="Plus" class="add-btn">
        新增用户
      </el-button>
    </div>
    
    <!-- 搜索区域 -->
    <el-card class="search-card" shadow="never">
      <el-form :inline="true" :model="searchForm" class="search-form">
        <el-form-item label="用户名">
          <el-input v-model="searchForm.username" placeholder="请输入用户名" clearable :prefix-icon="User" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchData" :icon="Search">搜索</el-button>
          <el-button @click="resetSearch" :icon="RefreshRight">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 数据表格 -->
    <el-card class="table-card" shadow="never">
      <el-table :data="tableData" v-loading="loading" row-key="id">
        <el-table-column prop="username" label="用户信息" min-width="200">
          <template #default="{ row }">
            <div class="user-cell">
              <el-avatar :size="40" class="user-avatar" :class="{ admin: row.role === 'ADMIN' }">
                {{ (row.nickname || row.username)?.charAt(0).toUpperCase() }}
              </el-avatar>
              <div class="user-detail">
                <span class="username">{{ row.username }}</span>
                <span class="nickname">{{ row.nickname || '未设置昵称' }}</span>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="role" label="角色" width="140" align="center">
          <template #default="{ row }">
            <div class="role-badge" :class="row.role === 'ADMIN' ? 'admin' : 'user'">
              <span class="role-icon">{{ row.role === 'ADMIN' ? '👑' : '👤' }}</span>
              <span class="role-text">{{ row.role === 'ADMIN' ? '管理员' : '普通用户' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-switch 
              :model-value="row.status === 1" 
              @change="handleStatusChange(row)" 
              :disabled="row.username === 'admin'"
              inline-prompt
              active-text="启"
              inactive-text="禁"
              style="--el-switch-on-color: #10b981"
            />
          </template>
        </el-table-column>
        <el-table-column prop="createTime" label="创建时间" width="180">
          <template #default="{ row }">
            <div class="time-cell">
              <el-icon><Clock /></el-icon>
              <span>{{ row.createTime }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-btns">
              <el-button type="primary" link size="small" @click="openDialog(row)">编辑</el-button>
              <el-popconfirm 
                title="确定删除该用户吗？" 
                @confirm="handleDelete(row.id, row.version)" 
                v-if="row.username !== 'admin'"
              >
                <template #reference>
                  <el-button type="danger" link size="small">删除</el-button>
                </template>
              </el-popconfirm>
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
          :page-sizes="[10, 20, 50]" 
          layout="sizes, prev, pager, next"
          @size-change="fetchData" 
          @current-change="fetchData" 
        />
      </div>
    </el-card>

    <!-- 新增/编辑弹窗 -->
    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑用户' : '新增用户'" width="450px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="computedRules" label-width="80px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" placeholder="请输入用户名" :disabled="!!form.id" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
        <el-input 
            v-model="form.password" 
            type="password" 
            :placeholder="form.id ? '不修改请留空' : '请输入密码（12-128位）'" 
            show-password 
          />
          <div class="form-tip" v-if="form.id">留空表示不修改密码</div>
        </el-form-item>
        <el-form-item label="昵称">
          <el-input v-model="form.nickname" placeholder="请输入昵称" />
        </el-form-item>
        <el-form-item label="角色">
          <el-radio-group v-model="form.role" :disabled="isAdminUser">
            <el-radio-button value="USER">
              <span>👤 普通用户</span>
            </el-radio-button>
            <el-radio-button value="ADMIN">
              <span>👑 管理员</span>
            </el-radio-button>
          </el-radio-group>
          <div class="form-tip warning" v-if="isAdminUser">管理员账号不能更改角色</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitLoading">
          {{ form.id ? '保存修改' : '确认添加' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, RefreshRight, Plus, Edit, Delete, User, Clock } from '@element-plus/icons-vue'
import { getUserPage, createUser, updateUser, deleteUser, updateUserStatus } from '@/api/user'

const loading = ref(false)
const submitLoading = ref(false)
const dialogVisible = ref(false)
const formRef = ref()
const tableData = ref([])

const searchForm = reactive({ username: '' })
const pagination = reactive({ current: 1, size: 10, total: 0 })
const form = reactive({ id: null, username: '', password: '', nickname: '', role: 'USER', originalUsername: '' })

// 判断是否是admin用户
const isAdminUser = computed(() => form.originalUsername === 'admin')

// 密码验证器
const validatePassword = (rule, value, callback) => {
  if (form.id) {
    // 编辑模式：密码可以为空，但如果填写了则需要验证长度
    if (value && (value.length < 12 || value.length > 128)) {
      callback(new Error('密码长度需在12-128位之间'))
    } else {
      callback()
    }
  } else {
    // 新增模式：密码必填且验证长度
    if (!value) {
      callback(new Error('请输入密码'))
    } else if (value.length < 12 || value.length > 128) {
      callback(new Error('密码长度需在12-128位之间'))
    } else {
      callback()
    }
  }
}

// 动态计算验证规则
const computedRules = computed(() => ({
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ validator: validatePassword, trigger: 'blur' }]
}))

const fetchData = async () => {
  loading.value = true
  try {
    const data = await getUserPage({ ...searchForm, current: pagination.current, size: pagination.size })
    tableData.value = data.records || []
    pagination.total = data.total || 0
  } finally {
    loading.value = false
  }
}

const resetSearch = () => {
  searchForm.username = ''
  pagination.current = 1
  fetchData()
}

const openDialog = (row) => {
  if (row) {
    Object.assign(form, { ...row, password: '', originalUsername: row.username })
  } else {
    Object.assign(form, { id: null, username: '', password: '', nickname: '', role: 'USER', originalUsername: '' })
  }
  dialogVisible.value = true
}

const handleSubmit = async () => {
  await formRef.value.validate()
  submitLoading.value = true
  try {
    const submitData = { ...form }
    // 编辑模式下，如果密码为空则不提交密码字段
    if (form.id && !form.password) {
      delete submitData.password
    }
    
    if (form.id) {
      await updateUser(form.id, submitData)
      ElMessage.success('更新成功')
    } else {
      await createUser(submitData)
      ElMessage.success('新增成功')
    }
    dialogVisible.value = false
    fetchData()
  } finally {
    submitLoading.value = false
  }
}

const handleDelete = async (id, version) => {
  await deleteUser(id, version)
  ElMessage.success('删除成功')
  fetchData()
}

const handleStatusChange = async (row) => {
  const newStatus = row.status === 1 ? 0 : 1
  await updateUserStatus(row.id, newStatus, row.version)
  ElMessage.success('状态更新成功')
  fetchData()
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
      color: #1e293b;
      margin-bottom: 4px;
    }
    
    p {
      font-size: 14px;
      color: #64748b;
    }
  }
  
  .add-btn {
    height: 40px;
    padding: 0 20px;
  }
}

.search-card {
  margin-bottom: 20px;
  border-radius: 12px;
  
  .search-form {
    .el-form-item {
      margin-bottom: 0;
      margin-right: 12px;
    }
  }
}

.table-card {
  border-radius: 12px;
  
  :deep(.el-card__body) {
    padding: 0;
  }
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 12px;
  
  .user-avatar {
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    color: #fff;
    font-weight: 600;
    flex-shrink: 0;
    
    &.admin {
      background: linear-gradient(135deg, #f59e0b 0%, #f97316 100%);
    }
  }
  
  .user-detail {
    display: flex;
    flex-direction: column;
    
    .username {
      font-weight: 600;
      color: #1e293b;
    }
    
    .nickname {
      font-size: 12px;
      color: #94a3b8;
    }
  }
}

.role-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 500;
  
  &.admin {
    background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
    color: #b45309;
    
    .role-icon {
      font-size: 14px;
    }
  }
  
  &.user {
    background: #f1f5f9;
    color: #64748b;
    
    .role-icon {
      font-size: 14px;
    }
  }
}

.time-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #64748b;
  
  .el-icon {
    color: #94a3b8;
  }
}

.action-btns {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  white-space: nowrap;
}

.pagination-wrapper {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-top: 1px solid #e2e8f0;
  
  .pagination-info {
    font-size: 13px;
    color: #64748b;
    
    .total {
      font-weight: 600;
      color: #6366f1;
    }
  }
}

.form-tip {
  font-size: 12px;
  color: #94a3b8;
  margin-top: 4px;
  
  &.warning {
    color: #f59e0b;
  }
}

:deep(.el-radio-button__inner) {
  display: flex;
  align-items: center;
  gap: 4px;
}
</style>
