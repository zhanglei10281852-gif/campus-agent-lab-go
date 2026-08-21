<template>
  <div class="page-container">
    <div class="page-header">
      <div class="header-info">
        <h2>系统设置</h2>
        <p>查看系统信息</p>
      </div>
    </div>
    
    <el-row :gutter="20">
      <!-- 关于系统 -->
      <el-col :xs="24" :sm="24" :md="12">
        <el-card class="about-card" shadow="never">
          <template #header>
            <div class="card-title">
              <el-icon><InfoFilled /></el-icon>
              <span>关于系统</span>
            </div>
          </template>
          <div class="about-content">
            <div class="about-logo">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M22 10v6M2 10l10-5 10 5-10 5z"/>
                <path d="M6 12v5c3 3 9 3 12 0v-5"/>
              </svg>
            </div>
            <h3>学生管理系统</h3>
            <p class="version">Version 1.0.0</p>
            <div class="tech-tags">
              <el-tag effect="plain" size="small">Vue 3</el-tag>
              <el-tag effect="plain" size="small">Element Plus</el-tag>
              <el-tag effect="plain" size="small">Spring Boot 3</el-tag>
              <el-tag effect="plain" size="small">MySQL 8.0</el-tag>
            </div>
          </div>
        </el-card>
      </el-col>
      
      <!-- 快捷操作 -->
      <el-col :xs="24" :sm="24" :md="12">
        <el-card class="quick-card" shadow="never">
          <template #header>
            <div class="card-title">
              <el-icon><Operation /></el-icon>
              <span>快捷操作</span>
            </div>
          </template>
          <div class="quick-actions">
            <div class="action-item" @click="refreshPage">
              <div class="action-icon refresh">
                <el-icon><RefreshRight /></el-icon>
              </div>
              <div class="action-text">
                <span class="title">刷新页面</span>
                <span class="desc">重新加载当前页面</span>
              </div>
            </div>
            <div class="action-item danger" @click="clearCache">
              <div class="action-icon clear">
                <el-icon><Delete /></el-icon>
              </div>
              <div class="action-text">
                <span class="title">清除缓存并退出</span>
                <span class="desc">清除本地存储并重新登录</span>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ElMessage, ElMessageBox } from 'element-plus'
import { InfoFilled, Operation, RefreshRight, Delete } from '@element-plus/icons-vue'

const refreshPage = () => {
  window.location.reload()
}

const clearCache = () => {
  ElMessageBox.confirm('清除缓存后需要重新登录，确定继续吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(() => {
    localStorage.clear()
    ElMessage.success('缓存已清除')
    setTimeout(() => {
      window.location.href = '/login'
    }, 1000)
  }).catch(() => {})
}
</script>

<style lang="scss" scoped>
.page-header {
  margin-bottom: 24px;
  
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
}

.about-card, .quick-card {
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  
  :deep(.el-card__header) {
    padding: 16px 20px;
    border-bottom: 1px solid #e2e8f0;
  }
  
  :deep(.el-card__body) {
    padding: 24px;
  }
  
  .card-title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 600;
    color: #1e293b;
    
    .el-icon {
      color: #6366f1;
    }
  }
}

.about-card {
  .about-content {
    text-align: center;
    
    .about-logo {
      width: 72px;
      height: 72px;
      margin: 0 auto 20px;
      background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
      border-radius: 18px;
      display: flex;
      align-items: center;
      justify-content: center;
      
      svg {
        width: 40px;
        height: 40px;
        color: #fff;
      }
    }
    
    h3 {
      font-size: 20px;
      font-weight: 700;
      color: #1e293b;
      margin-bottom: 6px;
    }
    
    .version {
      font-size: 14px;
      color: #94a3b8;
      margin-bottom: 20px;
    }
    
    .tech-tags {
      display: flex;
      flex-wrap: wrap;
      justify-content: center;
      gap: 8px;
      
      .el-tag {
        border-radius: 6px;
      }
    }
  }
}

.quick-card {
  .quick-actions {
    display: flex;
    flex-direction: column;
    gap: 16px;
    
    .action-item {
      display: flex;
      align-items: center;
      gap: 16px;
      padding: 16px;
      background: #f8fafc;
      border-radius: 12px;
      cursor: pointer;
      transition: all 0.2s ease;
      
      &:hover {
        background: #f1f5f9;
        transform: translateX(4px);
      }
      
      &.danger:hover {
        background: #fef2f2;
      }
      
      .action-icon {
        width: 48px;
        height: 48px;
        border-radius: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
        
        &.refresh {
          background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
          color: #fff;
        }
        
        &.clear {
          background: linear-gradient(135deg, #ef4444 0%, #f87171 100%);
          color: #fff;
        }
        
        .el-icon {
          font-size: 22px;
        }
      }
      
      .action-text {
        flex: 1;
        
        .title {
          display: block;
          font-size: 15px;
          font-weight: 600;
          color: #1e293b;
          margin-bottom: 4px;
        }
        
        .desc {
          font-size: 13px;
          color: #94a3b8;
        }
      }
    }
  }
}

@media (max-width: 767px) {
  .el-col {
    margin-bottom: 20px;
    
    &:last-child {
      margin-bottom: 0;
    }
  }
}
</style>
