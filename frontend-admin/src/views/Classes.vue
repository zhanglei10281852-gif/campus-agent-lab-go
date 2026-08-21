<template>
  <div class="page-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <div class="header-info">
        <h2>班级管理</h2>
        <p>管理系统中的所有班级信息</p>
      </div>
      <el-button type="primary" @click="openDialog()" :icon="Plus" class="add-btn">
        新增班级
      </el-button>
    </div>
    
    <!-- 搜索区域 -->
    <el-card class="search-card" shadow="never">
      <el-form :inline="true" :model="searchForm" class="search-form">
        <el-form-item label="班级名称">
          <el-input v-model="searchForm.className" placeholder="请输入班级名称" clearable :prefix-icon="School" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="fetchData" :icon="Search">搜索</el-button>
          <el-button @click="resetSearch" :icon="RefreshRight">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 班级卡片列表 -->
    <div class="class-grid" v-loading="loading">
      <div v-for="item in tableData" :key="item.id" class="class-card">
        <div class="card-header">
          <div class="class-icon">
            <el-icon :size="24"><School /></el-icon>
          </div>
          <div class="card-actions">
            <el-button type="primary" link :icon="Edit" @click="openDialog(item)" />
            <el-popconfirm title="确定删除该班级吗？" @confirm="handleDelete(item.id, item.version)">
              <template #reference>
                <el-button type="danger" link :icon="Delete" />
              </template>
            </el-popconfirm>
          </div>
        </div>
        <div class="card-body">
          <h3 class="class-name">{{ item.className }}</h3>
          <div class="class-meta">
            <span class="grade">{{ item.grade || '未设置年级' }}</span>
          </div>
        </div>
        <div class="card-footer">
          <div class="footer-item">
            <el-icon><User /></el-icon>
            <span>班主任: {{ item.teacher || '未分配' }}</span>
          </div>
          <div class="student-count">
            <span class="count">{{ item.studentCount || 0 }}</span>
            <span class="label">学生</span>
          </div>
        </div>
      </div>
      
      <!-- 空状态 -->
      <el-empty v-if="!loading && !tableData.length" description="暂无班级数据" class="empty-state" />
    </div>
    
    <!-- 分页 -->
    <div class="pagination-container" v-if="pagination.total > 0">
      <el-pagination 
        v-model:current-page="pagination.current" 
        v-model:page-size="pagination.size"
        :total="pagination.total" 
        :page-sizes="[12, 24, 48]" 
        layout="total, sizes, prev, pager, next"
        @size-change="fetchData" 
        @current-change="fetchData" 
      />
    </div>

    <!-- 新增/编辑弹窗 -->
    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑班级' : '新增班级'" width="450px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-form-item label="班级名称" prop="className">
          <el-input v-model="form.className" placeholder="请输入班级名称" />
        </el-form-item>
        <el-form-item label="年级">
          <el-input v-model="form.grade" placeholder="请输入年级，如：2024级" />
        </el-form-item>
        <el-form-item label="班主任">
          <el-input v-model="form.teacher" placeholder="请输入班主任姓名" />
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
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, RefreshRight, Plus, Edit, Delete, School, User } from '@element-plus/icons-vue'
import { getClassPage, createClass, updateClass, deleteClass } from '@/api/class'

const loading = ref(false)
const submitLoading = ref(false)
const dialogVisible = ref(false)
const formRef = ref()
const tableData = ref([])

const searchForm = reactive({ className: '' })
const pagination = reactive({ current: 1, size: 12, total: 0 })
const form = reactive({ id: null, className: '', grade: '', teacher: '' })

const rules = {
  className: [{ required: true, message: '请输入班级名称', trigger: 'blur' }]
}

const fetchData = async () => {
  loading.value = true
  try {
    const data = await getClassPage({ ...searchForm, current: pagination.current, size: pagination.size })
    tableData.value = data.records || []
    pagination.total = data.total || 0
  } finally {
    loading.value = false
  }
}

const resetSearch = () => {
  searchForm.className = ''
  pagination.current = 1
  fetchData()
}

const openDialog = (row) => {
  if (row) {
    Object.assign(form, row)
  } else {
    Object.assign(form, { id: null, className: '', grade: '', teacher: '' })
  }
  dialogVisible.value = true
}

const handleSubmit = async () => {
  await formRef.value.validate()
  submitLoading.value = true
  try {
    if (form.id) {
      await updateClass(form.id, form)
      ElMessage.success('更新成功')
    } else {
      await createClass(form)
      ElMessage.success('新增成功')
    }
    dialogVisible.value = false
    fetchData()
  } finally {
    submitLoading.value = false
  }
}

const handleDelete = async (id, version) => {
    await deleteClass(id, version)
  ElMessage.success('删除成功')
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

.class-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
  min-height: 200px;
}

.class-card {
  background: #fff;
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  padding: 24px;
  transition: all 0.3s ease;
  display: flex;
  flex-direction: column;
  
  &:hover {
    transform: translateY(-4px);
    box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.1);
    border-color: #a5b4fc;
  }
  
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
    
    .class-icon {
      width: 56px;
      height: 56px;
      background: linear-gradient(135deg, #10b981 0%, #34d399 100%);
      border-radius: 14px;
      display: flex;
      align-items: center;
      justify-content: center;
      color: #fff;
      flex-shrink: 0;
      
      .el-icon {
        font-size: 28px;
      }
    }
    
    .card-actions {
      display: flex;
      gap: 4px;
      
      .el-button {
        padding: 8px;
      }
    }
  }
  
  .card-body {
    margin-bottom: 20px;
    flex: 1;
    
    .class-name {
      font-size: 18px;
      font-weight: 700;
      color: #1e293b;
      margin-bottom: 10px;
      line-height: 1.4;
    }
    
    .class-meta {
      .grade {
        display: inline-block;
        font-size: 13px;
        color: #64748b;
        background: #f1f5f9;
        padding: 6px 12px;
        border-radius: 20px;
      }
    }
  }
  
  .card-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding-top: 16px;
    border-top: 1px solid #e2e8f0;
    
    .footer-item {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 14px;
      color: #64748b;
      
      .el-icon {
        color: #94a3b8;
        font-size: 16px;
      }
    }
    
    .student-count {
      display: flex;
      align-items: baseline;
      gap: 4px;
      
      .count {
        font-size: 28px;
        font-weight: 700;
        color: #6366f1;
        line-height: 1;
      }
      
      .label {
        font-size: 13px;
        color: #94a3b8;
      }
    }
  }
}

.empty-state {
  grid-column: 1 / -1;
}

.pagination-container {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 12px;
  border: 1px solid #e2e8f0;
}
</style>
