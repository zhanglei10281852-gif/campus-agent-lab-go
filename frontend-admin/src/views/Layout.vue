<template>
  <el-container class="layout-container">
    <!-- 侧边栏 -->
    <el-aside :width="isCollapsed ? '64px' : '240px'" class="aside">
      <div class="logo" @click="router.push('/')">
        <div class="logo-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 10v6M2 10l10-5 10 5-10 5z"/>
            <path d="M6 12v5c3 3 9 3 12 0v-5"/>
          </svg>
        </div>
        <transition name="fade">
          <span v-if="!isCollapsed" class="logo-text">智能体实训台</span>
        </transition>
      </div>
      
      <el-menu 
        :default-active="route.path" 
        router 
        :collapse="isCollapsed"
        class="sidebar-menu"
      >
        <el-menu-item v-for="item in menuItems" :key="item.path" :index="item.path">
          <el-icon><component :is="item.icon" /></el-icon>
          <template #title>{{ item.title }}</template>
        </el-menu-item>
      </el-menu>
      
      <div class="sidebar-footer">
        <el-button 
          :icon="isCollapsed ? Expand : Fold" 
          text 
          @click="isCollapsed = !isCollapsed"
          class="collapse-btn"
        />
      </div>
    </el-aside>
    
    <!-- 主内容区 -->
    <el-container class="main-container">
      <!-- 顶部导航 -->
      <el-header class="header">
        <div class="header-left">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/' }">
              <el-icon><HomeFilled /></el-icon>
            </el-breadcrumb-item>
            <el-breadcrumb-item>{{ route.meta.title }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        
        <div class="header-right">
          <!-- 用户信息 -->
          <el-dropdown @command="handleCommand" trigger="click">
            <div class="user-info">
              <el-avatar :size="36" class="user-avatar">
                {{ (userStore.userInfo.nickname || userStore.userInfo.username || 'U').charAt(0).toUpperCase() }}
              </el-avatar>
              <div class="user-detail">
                <span class="user-name">{{ userStore.userInfo.display_name || userStore.userInfo.email }}</span>
                <span class="user-role">{{ roleLabel(userStore.userInfo.role) }}</span>
              </div>
              <el-icon class="dropdown-icon"><ArrowDown /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item :icon="User" command="profile">个人中心</el-dropdown-item>
                <el-dropdown-item :icon="Setting" command="settings">系统设置</el-dropdown-item>
                <el-dropdown-item divided :icon="SwitchButton" command="logout">
                  退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>
      
      <!-- 内容区 -->
      <el-main class="main">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { ElMessage } from 'element-plus'
import { 
  Odometer, User, School, UserFilled, Document, Monitor,
  HomeFilled, ArrowDown, Setting, SwitchButton,
  Expand, Fold
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const isCollapsed = ref(false)

const menuItems = [
  { path: '/dashboard', title: '运行总览', icon: Odometer },
  { path: '/students', title: '实训学员', icon: User },
  { path: '/classes', title: '实训班级', icon: School },
  { path: '/executions', title: '执行治理', icon: Monitor },
  { path: '/users', title: '治理成员', icon: UserFilled },
  { path: '/logs', title: '审计记录', icon: Document }
]

const roleLabel = (role) => ({
  agent_developer: '平台管理员',
  tool_operator: '实训运营',
  security_reviewer: '安全复核',
  compliance_auditor: '审计查看',
}[role] || '成员')

const handleCommand = (command) => {
  if (command === 'logout') {
    userStore.logout()
    ElMessage.success('已退出登录')
    router.push('/login')
  } else if (command === 'profile') {
    router.push('/profile')
  } else if (command === 'settings') {
    router.push('/settings')
  }
}
</script>

<style lang="scss" scoped>
.layout-container {
  height: 100vh;
  background: var(--bg-color);
}

.aside {
  background: linear-gradient(180deg, #1e293b 0%, #0f172a 100%);
  display: flex;
  flex-direction: column;
  transition: width 0.3s ease;
  box-shadow: 4px 0 10px rgba(0, 0, 0, 0.1);
  z-index: 100;
}

.logo {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  cursor: pointer;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  padding: 0 16px;
  
  .logo-icon {
    width: 36px;
    height: 36px;
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    border-radius: 10px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    
    svg {
      width: 22px;
      height: 22px;
      color: #fff;
    }
  }
  
  .logo-text {
    font-size: 18px;
    font-weight: 700;
    color: #fff;
    white-space: nowrap;
  }
}

.sidebar-menu {
  flex: 1;
  border-right: none;
  background: transparent;
  padding: 12px 8px;
  
  --el-menu-bg-color: transparent;
  --el-menu-text-color: #94a3b8;
  --el-menu-hover-bg-color: rgba(99, 102, 241, 0.1);
  --el-menu-active-color: #fff;
  
  :deep(.el-menu-item) {
    height: 48px;
    line-height: 48px;
    margin: 4px 0;
    border-radius: 10px;
    transition: all 0.2s ease;
    
    &:hover {
      background: rgba(99, 102, 241, 0.15);
      color: #fff;
    }
    
    &.is-active {
      background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
      color: #fff;
      box-shadow: 0 4px 12px rgba(99, 102, 241, 0.4);
      
      .el-icon {
        color: #fff;
      }
    }
    
    .el-icon {
      font-size: 20px;
      margin-right: 12px;
    }
  }
  
  &.el-menu--collapse {
    padding: 12px 10px;
    
    :deep(.el-menu-item) {
      padding: 0 !important;
      display: flex !important;
      justify-content: center !important;
      align-items: center !important;
      width: 44px;
      margin: 4px auto;
      
      .el-icon {
        margin: 0 !important;
      }
      
      .el-menu-tooltip__trigger {
        display: flex !important;
        justify-content: center !important;
        align-items: center !important;
        width: 100%;
        height: 100%;
      }
    }
  }
}

.sidebar-footer {
  padding: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
  
  .collapse-btn {
    width: 100%;
    color: #94a3b8;
    
    &:hover {
      color: #fff;
      background: rgba(255, 255, 255, 0.1);
    }
  }
}

.main-container {
  flex-direction: column;
}

.header {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  background: #fff;
  border-bottom: 1px solid var(--border-color);
  box-shadow: var(--shadow-sm);
}

.header-left {
  :deep(.el-breadcrumb) {
    font-size: 14px;
    
    .el-breadcrumb__inner {
      color: var(--text-secondary);
      
      &.is-link:hover {
        color: var(--el-color-primary);
      }
    }
    
    .el-breadcrumb__item:last-child .el-breadcrumb__inner {
      color: var(--text-primary);
      font-weight: 500;
    }
  }
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 6px 12px;
  border-radius: 10px;
  cursor: pointer;
  transition: background 0.2s;
  
  &:hover {
    background: var(--bg-color-secondary);
  }
  
  .user-avatar {
    background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
    color: #fff;
    font-weight: 600;
  }
  
  .user-detail {
    display: flex;
    flex-direction: column;
    
    .user-name {
      font-size: 14px;
      font-weight: 600;
      color: var(--text-primary);
      line-height: 1.2;
    }
    
    .user-role {
      font-size: 12px;
      color: var(--text-muted);
    }
  }
  
  .dropdown-icon {
    color: var(--text-muted);
    font-size: 12px;
  }
}

.main {
  background: var(--bg-color);
  padding: 0;
  overflow-y: auto;
  min-width: 0;
}

@media (max-width: 760px) {
  .aside { width: 64px !important; }
  .main-container { min-width: 0; }
  .header { min-width: 0; padding: 0 12px; }
  .user-info .user-detail, .user-info .dropdown-icon { display: none; }
  .header-right { flex-shrink: 0; }
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.fade-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.fade-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}
</style>
