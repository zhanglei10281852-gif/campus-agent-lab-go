<template>
  <div class="page-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="header-info">
        <h2>学生管理</h2>
        <p>管理系统中的所有学生信息</p>
      </div>
      <el-button type="primary" @click="openDialog()" :icon="Plus" class="add-btn">
        新增学生
      </el-button>
    </div>
    
    <!-- 搜索区域 -->
    <el-card class="search-card" shadow="never">
      <el-form :inline="true" :model="searchForm" class="search-form">
        <el-form-item label="姓名">
          <el-input v-model="searchForm.name" placeholder="请输入姓名" clearable :prefix-icon="User" />
        </el-form-item>
        <el-form-item label="学号">
          <el-input v-model="searchForm.studentNo" placeholder="请输入学号" clearable />
        </el-form-item>
        <el-form-item label="班级">
          <el-select v-model="searchForm.classId" placeholder="全部班级" clearable style="width: 180px">
            <el-option v-for="c in classList" :key="c.id" :label="c.className" :value="c.id" />
          </el-select>
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
        <el-table-column prop="studentNo" label="学号" width="120">
          <template #default="{ row }">
            <span class="student-no">{{ row.studentNo }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="学生信息" min-width="180">
          <template #default="{ row }">
            <div class="student-cell">
              <el-avatar :size="36" class="student-avatar">
                {{ row.name?.charAt(0) }}
              </el-avatar>
              <div class="student-detail">
                <span class="name">{{ row.name }}</span>
                <span class="gender">{{ row.gender === 1 ? '男' : '女' }}</span>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="phone" label="联系方式" width="200">
          <template #default="{ row }">
            <div class="contact-cell">
              <div v-if="row.phone" class="contact-item">
                <el-icon><Phone /></el-icon>
                <span>{{ row.phone }}</span>
              </div>
              <div v-if="row.email" class="contact-item email">
                <el-icon><Message /></el-icon>
                <span>{{ row.email }}</span>
              </div>
              <span v-if="!row.phone && !row.email" class="no-data">-</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="className" label="班级" width="150">
          <template #default="{ row }">
            <el-tag v-if="row.className" type="info" effect="plain" class="class-tag">
              {{ row.className }}
            </el-tag>
            <span v-else class="no-data">未分配</span>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="statusMap[row.status]?.type" effect="light">
              {{ statusMap[row.status]?.label }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-button type="primary" link :icon="Edit" @click="openDialog(row)">编辑</el-button>
              <el-popconfirm 
                title="确定删除该学生吗？" 
                confirm-button-text="确定"
                cancel-button-text="取消"
                @confirm="handleDelete(row.id, row.version)"
              >
                <template #reference>
                  <el-button type="danger" link :icon="Delete">删除</el-button>
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
    <el-dialog 
      v-model="dialogVisible" 
      :title="form.id ? '编辑学生' : '新增学生'" 
      width="520px" 
      destroy-on-close
      class="student-dialog"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="学号" prop="studentNo">
              <el-input v-model="form.studentNo" placeholder="请输入学号" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="姓名" prop="name">
              <el-input v-model="form.name" placeholder="请输入姓名" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="性别" prop="gender">
              <el-radio-group v-model="form.gender">
                <el-radio :value="1">男</el-radio>
                <el-radio :value="2">女</el-radio>
              </el-radio-group>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="出生日期">
              <el-date-picker 
                v-model="form.birthDate" 
                type="date" 
                placeholder="选择日期" 
                value-format="YYYY-MM-DD" 
                style="width: 100%" 
              />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="联系电话">
          <el-input v-model="form.phone" placeholder="请输入联系电话" :prefix-icon="Phone" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="form.email" placeholder="请输入邮箱" :prefix-icon="Message" />
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="班级">
              <el-select v-model="form.classId" placeholder="请选择班级" style="width: 100%">
                <el-option v-for="c in classList" :key="c.id" :label="c.className" :value="c.id" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="状态">
              <el-select v-model="form.status" placeholder="请选择状态" style="width: 100%">
                <el-option v-for="(item, key) in statusMap" :key="key" :label="item.label" :value="Number(key)" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
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
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, RefreshRight, Plus, Edit, Delete, User, Phone, Message, School } from '@element-plus/icons-vue'
import { getStudentPage, createStudent, updateStudent, deleteStudent } from '@/api/student'
import { getAllClasses } from '@/api/class'

const loading = ref(false)
const submitLoading = ref(false)
const dialogVisible = ref(false)
const formRef = ref()
const tableData = ref([])
const classList = ref([])

const statusMap = {
  0: { label: '休学', type: 'info' },
  1: { label: '在读', type: 'success' },
  2: { label: '毕业', type: 'warning' }
}

const searchForm = reactive({ name: '', studentNo: '', classId: null })
const pagination = reactive({ current: 1, size: 10, total: 0 })

const form = reactive({
  id: null, studentNo: '', name: '', gender: 1, birthDate: '', phone: '', email: '', classId: null, status: 1
})

const rules = {
  studentNo: [{ required: true, message: '请输入学号', trigger: 'blur' }],
  name: [{ required: true, message: '请输入姓名', trigger: 'blur' }],
  gender: [{ required: true, message: '请选择性别', trigger: 'change' }]
}

const fetchData = async () => {
  loading.value = true
  try {
    const data = await getStudentPage({ ...searchForm, current: pagination.current, size: pagination.size })
    tableData.value = data.records || []
    pagination.total = data.total || 0
  } finally {
    loading.value = false
  }
}

const fetchClasses = async () => {
  classList.value = await getAllClasses() || []
}

const resetSearch = () => {
  Object.assign(searchForm, { name: '', studentNo: '', classId: null })
  pagination.current = 1
  fetchData()
}

const openDialog = (row) => {
  if (row) {
    Object.assign(form, row)
  } else {
    Object.assign(form, { id: null, studentNo: '', name: '', gender: 1, birthDate: '', phone: '', email: '', classId: null, status: 1 })
  }
  dialogVisible.value = true
}

const handleSubmit = async () => {
  await formRef.value.validate()
  submitLoading.value = true
  try {
    if (form.id) {
      await updateStudent(form.id, form)
      ElMessage.success('更新成功')
    } else {
      await createStudent(form)
      ElMessage.success('新增成功')
    }
    dialogVisible.value = false
    fetchData()
  } finally {
    submitLoading.value = false
  }
}

const handleDelete = async (id, version) => {
    await deleteStudent(id, version)
  ElMessage.success('删除成功')
  fetchData()
}

onMounted(() => {
  fetchData()
  fetchClasses()
})
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
  
  .add-btn {
    height: 40px;
    padding: 0 20px;
  }
}

.search-card {
  margin-bottom: 20px;
  
  .search-form {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    
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
  
  :deep(.el-table) {
    width: 100%;
  }
}

.student-no {
  font-family: 'SF Mono', Monaco, monospace;
  font-size: 13px;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  padding: 4px 10px;
  border-radius: 6px;
}

.student-cell {
  display: flex;
  align-items: center;
  gap: 12px;
  
  .student-avatar {
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    color: #fff;
    font-weight: 600;
    flex-shrink: 0;
  }
  
  .student-detail {
    display: flex;
    flex-direction: column;
    
    .name {
      font-weight: 600;
      color: var(--text-primary);
    }
    
    .gender {
      font-size: 12px;
      color: var(--text-muted);
    }
  }
}

.contact-cell {
  .contact-item {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 13px;
    color: var(--text-secondary);
    
    .el-icon {
      font-size: 14px;
      color: var(--text-muted);
    }
    
    &.email {
      margin-top: 4px;
      font-size: 12px;
    }
  }
  
  .no-data {
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

.action-buttons {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  white-space: nowrap;
}

.student-dialog {
  :deep(.el-dialog__body) {
    padding-top: 20px;
  }
}

// 班级下拉选项样式
.class-option {
  display: flex;
  align-items: center;
  gap: 8px;
  
  .option-icon {
    font-size: 16px;
    line-height: 1;
  }
}
</style>
