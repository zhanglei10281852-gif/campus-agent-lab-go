<template>
  <div class="login-container">
    <!-- 背景装饰 -->
    <div class="bg-decoration">
      <div class="circle circle-1"></div>
      <div class="circle circle-2"></div>
      <div class="circle circle-3"></div>
    </div>
    
    <div class="login-wrapper">
      <!-- 左侧品牌区 -->
      <div class="brand-section">
        <div class="brand-content">
          <div class="brand-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M22 10v6M2 10l10-5 10 5-10 5z"/>
              <path d="M6 12v5c3 3 9 3 12 0v-5"/>
            </svg>
          </div>
          <h1 class="brand-title">智能体实训台</h1>
          <p class="brand-desc">Campus Agent Lab</p>
          <div class="brand-features">
            <div class="feature-item">
              <span class="feature-icon">✓</span>
              <span>统一管理实训班级与学员</span>
            </div>
            <div class="feature-item">
              <span class="feature-icon">✓</span>
              <span>追踪工具版本与执行配额</span>
            </div>
            <div class="feature-item">
              <span class="feature-icon">✓</span>
              <span>保留审批、审计与运行证据</span>
            </div>
          </div>
        </div>
      </div>
      
      <!-- 右侧登录区 -->
      <div class="login-section">
        <div class="login-card">
          <div class="login-header">
            <h2>欢迎回来</h2>
            <p>请登录您的账户继续</p>
          </div>
          
          <el-form ref="formRef" :model="form" :rules="rules" @keyup.enter="handleLogin" class="login-form">
            <el-form-item prop="username">
              <el-input 
                v-model="form.username" 
                placeholder="请输入邮箱" 
                size="large"
                :prefix-icon="User"
              />
            </el-form-item>
            <el-form-item prop="password">
              <el-input 
                v-model="form.password" 
                type="password" 
                placeholder="请输入密码" 
                size="large"
                :prefix-icon="Lock"
                show-password 
              />
            </el-form-item>
            
            <div class="login-options">
              <el-checkbox v-model="rememberMe">记住我</el-checkbox>
            </div>
            
            <el-form-item>
              <el-button 
                type="primary" 
                size="large" 
                :loading="loading" 
                @click="handleLogin"
                class="login-btn"
              >
                <span v-if="!loading">登 录</span>
                <span v-else>登录中...</span>
              </el-button>
            </el-form-item>
          </el-form>
          
          <div class="login-footer">
            <div class="divider">
              <span>测试账号</span>
            </div>
            <div class="test-account">
              <code>operator@example.test</code> / <code>very-secure-password</code>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()
const formRef = ref()
const loading = ref(false)
const rememberMe = ref(false)

const form = reactive({
  username: 'operator@example.test',
  password: 'very-secure-password'
})

const rules = {
  username: [{ required: true, type: 'email', message: '请输入有效邮箱', trigger: 'blur' }],
  password: [{ required: true, min: 12, message: '密码至少需要 12 位', trigger: 'blur' }]
}

const handleLogin = async () => {
  await formRef.value.validate()
  loading.value = true
  try {
    await userStore.login(form)
    ElMessage.success('登录成功')
    router.push('/')
  } finally {
    loading.value = false
  }
}
</script>

<style lang="scss" scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  position: relative;
  overflow: hidden;
}

.bg-decoration {
  position: absolute;
  inset: 0;
  overflow: hidden;
  
  .circle {
    position: absolute;
    border-radius: 50%;
    background: rgba(255, 255, 255, 0.1);
  }
  
  .circle-1 {
    width: 600px;
    height: 600px;
    top: -200px;
    right: -100px;
  }
  
  .circle-2 {
    width: 400px;
    height: 400px;
    bottom: -100px;
    left: -100px;
  }
  
  .circle-3 {
    width: 200px;
    height: 200px;
    top: 50%;
    left: 30%;
    background: rgba(255, 255, 255, 0.05);
  }
}

.login-wrapper {
  display: flex;
  width: 900px;
  min-height: 560px;
  background: #fff;
  border-radius: 24px;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
  overflow: hidden;
  position: relative;
  z-index: 1;
}

.brand-section {
  flex: 1;
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
  padding: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  
  &::before {
    content: '';
    position: absolute;
    inset: 0;
    background: url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23ffffff' fill-opacity='0.05'%3E%3Cpath d='M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E");
  }
}

.brand-content {
  text-align: center;
  color: #fff;
  position: relative;
  z-index: 1;
}

.brand-icon {
  width: 80px;
  height: 80px;
  margin: 0 auto 24px;
  background: rgba(255, 255, 255, 0.2);
  border-radius: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  
  svg {
    width: 48px;
    height: 48px;
  }
}

.brand-title {
  font-size: 28px;
  font-weight: 700;
  margin-bottom: 8px;
  letter-spacing: -0.5px;
}

.brand-desc {
  font-size: 14px;
  opacity: 0.8;
  margin-bottom: 40px;
}

.brand-features {
  text-align: left;
  
  .feature-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 0;
    font-size: 14px;
    opacity: 0.9;
    
    .feature-icon {
      width: 20px;
      height: 20px;
      background: rgba(255, 255, 255, 0.2);
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 12px;
    }
  }
}

.login-section {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px;
  background: #fff;
}

.login-card {
  width: 100%;
  max-width: 360px;
}

.login-header {
  margin-bottom: 32px;
  
  h2 {
    font-size: 24px;
    font-weight: 700;
    color: #1e293b;
    margin-bottom: 8px;
  }
  
  p {
    color: #64748b;
    font-size: 14px;
  }
}

.login-form {
  .el-form-item {
    margin-bottom: 20px;
  }
  
  :deep(.el-input) {
    --el-input-border-radius: 10px;
    
    .el-input__wrapper {
      padding: 4px 16px;
      box-shadow: 0 0 0 1px #e2e8f0 inset;
      
      &:hover {
        box-shadow: 0 0 0 1px #cbd5e1 inset;
      }
      
      &.is-focus {
        box-shadow: 0 0 0 2px #6366f1 inset;
      }
    }
    
    .el-input__inner {
      height: 44px;
    }
  }
}

.login-options {
  margin-bottom: 24px;
}

.login-btn {
  width: 100%;
  height: 48px;
  font-size: 16px;
  font-weight: 600;
  border-radius: 10px;
  background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
  border: none;
  transition: all 0.3s ease;
  
  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 10px 20px rgba(99, 102, 241, 0.3);
  }
  
  &:active {
    transform: translateY(0);
  }
}

.login-footer {
  margin-top: 32px;
  
  .divider {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 16px;
    
    &::before,
    &::after {
      content: '';
      flex: 1;
      height: 1px;
      background: #e2e8f0;
    }
    
    span {
      color: #94a3b8;
      font-size: 12px;
    }
  }
  
  .test-account {
    text-align: center;
    color: #64748b;
    font-size: 13px;
    
    code {
      background: #f1f5f9;
      padding: 2px 8px;
      border-radius: 4px;
      font-family: 'SF Mono', Monaco, monospace;
      color: #6366f1;
    }
  }
}

@media (max-width: 768px) {
  .login-wrapper {
    flex-direction: column;
    width: 90%;
    max-width: 400px;
  }
  
  .brand-section {
    padding: 32px;
    
    .brand-features {
      display: none;
    }
  }
}
</style>
