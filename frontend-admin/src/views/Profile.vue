<template>
  <div class="page-container">
    <div class="page-header">
      <div class="header-info">
        <h2>个人中心</h2>
        <p>管理您的个人信息和账户安全</p>
      </div>
    </div>
    
    <el-row :gutter="20">
      <!-- 左侧个人信息卡片 -->
      <el-col :xs="24" :lg="8">
        <el-card class="profile-card" shadow="never">
          <div class="profile-header">
            <el-avatar :size="80" class="profile-avatar">
              {{ (userStore.userInfo.display_name || userStore.userInfo.email || 'U').charAt(0).toUpperCase() }}
            </el-avatar>
            <h3 class="profile-name">{{ userStore.userInfo.display_name || userStore.userInfo.email }}</h3>
            <el-tag :type="userStore.userInfo.role === 'agent_developer' ? 'danger' : 'info'" effect="light">
              {{ userStore.userInfo.role === 'agent_developer' ? '平台管理员' : '实训运营' }}
            </el-tag>
          </div>
          <div class="profile-info">
            <div class="info-item">
              <el-icon><User /></el-icon>
              <span class="label">用户名</span>
              <span class="value">{{ userStore.userInfo.email }}</span>
            </div>
            <div class="info-item">
              <el-icon><Clock /></el-icon>
              <span class="label">注册时间</span>
              <span class="value">{{ userStore.userInfo.created_at || '-' }}</span>
            </div>
          </div>
        </el-card>
      </el-col>
      
      <!-- 右侧编辑表单 -->
      <el-col :xs="24" :lg="16">
        <el-card class="edit-card" shadow="never">
          <template #header>
            <span>编辑资料</span>
          </template>
          <el-form ref="profileFormRef" :model="profileForm" :rules="profileRules" label-width="100px">
            <el-form-item label="昵称" prop="nickname">
              <el-input v-model="profileForm.nickname" placeholder="请输入昵称" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleUpdateProfile" :loading="profileLoading">
                保存修改
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
        
        <el-card class="edit-card" shadow="never" style="margin-top: 20px;">
          <template #header>
            <span>修改密码</span>
          </template>
          <el-form ref="passwordFormRef" :model="passwordForm" :rules="passwordRules" label-width="100px">
            <el-form-item label="当前密码" prop="oldPassword">
              <el-input v-model="passwordForm.oldPassword" type="password" placeholder="请输入当前密码" show-password />
            </el-form-item>
            <el-form-item label="新密码" prop="newPassword">
              <el-input v-model="passwordForm.newPassword" type="password" placeholder="请输入新密码" show-password />
            </el-form-item>
            <el-form-item label="确认密码" prop="confirmPassword">
              <el-input v-model="passwordForm.confirmPassword" type="password" placeholder="请再次输入新密码" show-password />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="handleUpdatePassword" :loading="passwordLoading">
                修改密码
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { User, Clock } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { updateProfile, updatePassword } from '@/api/auth'

const userStore = useUserStore()
const profileFormRef = ref()
const passwordFormRef = ref()
const profileLoading = ref(false)
const passwordLoading = ref(false)

const profileForm = reactive({
  nickname: ''
})

const passwordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

const profileRules = {
  nickname: [{ required: true, message: '请输入昵称', trigger: 'blur' }]
}

const validateConfirmPassword = (rule, value, callback) => {
  if (value !== passwordForm.newPassword) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const passwordRules = {
  oldPassword: [{ required: true, message: '请输入当前密码', trigger: 'blur' }],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 12, message: '密码长度不能少于12位', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

const handleUpdateProfile = async () => {
  await profileFormRef.value.validate()
  profileLoading.value = true
  try {
    await updateProfile(profileForm)
    userStore.userInfo.display_name = profileForm.nickname
    localStorage.setItem('userInfo', JSON.stringify(userStore.userInfo))
    ElMessage.success('资料更新成功')
  } finally {
    profileLoading.value = false
  }
}

const handleUpdatePassword = async () => {
  await passwordFormRef.value.validate()
  passwordLoading.value = true
  try {
    await updatePassword({
      oldPassword: passwordForm.oldPassword,
      newPassword: passwordForm.newPassword
    })
    ElMessage.success('密码修改成功，请重新登录')
    passwordFormRef.value.resetFields()
  } finally {
    passwordLoading.value = false
  }
}

onMounted(() => {
  profileForm.nickname = userStore.userInfo.display_name || ''
})
</script>

<style lang="scss" scoped>
.page-header {
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

.profile-card {
  border-radius: var(--radius-md);
  border: 1px solid var(--border-color);
  
  .profile-header {
    text-align: center;
    padding: 24px 0;
    border-bottom: 1px solid var(--border-color);
    
    .profile-avatar {
      background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
      color: #fff;
      font-size: 32px;
      font-weight: 600;
    }
    
    .profile-name {
      margin: 16px 0 8px;
      font-size: 18px;
      font-weight: 600;
      color: var(--text-primary);
    }
  }
  
  .profile-info {
    padding: 16px 0;
    
    .info-item {
      display: flex;
      align-items: center;
      padding: 12px 20px;
      
      .el-icon {
        color: var(--text-muted);
        margin-right: 12px;
      }
      
      .label {
        color: var(--text-secondary);
        width: 80px;
      }
      
      .value {
        color: var(--text-primary);
        font-weight: 500;
      }
    }
  }
}

.edit-card {
  border-radius: var(--radius-md);
  border: 1px solid var(--border-color);
  
  :deep(.el-card__header) {
    font-weight: 600;
    color: var(--text-primary);
  }
  
  :deep(.el-form) {
    max-width: 400px;
  }
}
</style>
