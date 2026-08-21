import { createRouter, createWebHistory } from "vue-router";
import { useUserStore } from "@/stores/user";

const routes = [
  {
    path: "/login",
    name: "Login",
    component: () => import("@/views/Login.vue"),
    meta: { title: "登录" },
  },
  {
    path: "/",
    component: () => import("@/views/Layout.vue"),
    redirect: "/dashboard",
    children: [
      {
        path: "dashboard",
        name: "Dashboard",
        component: () => import("@/views/Dashboard.vue"),
        meta: { title: "仪表盘", icon: "Odometer" },
      },
      {
        path: "students",
        name: "Students",
        component: () => import("@/views/Students.vue"),
        meta: { title: "学生管理", icon: "User" },
      },
      {
        path: "classes",
        name: "Classes",
        component: () => import("@/views/Classes.vue"),
        meta: { title: "班级管理", icon: "School" },
      },
      {
        path: "executions",
        name: "Executions",
        component: () => import("@/views/ExecutionRequests.vue"),
        meta: { title: "执行治理", icon: "Monitor" },
      },
      {
        path: "users",
        name: "Users",
        component: () => import("@/views/Users.vue"),
        meta: { title: "用户管理", icon: "UserFilled" },
      },
      {
        path: "logs",
        name: "Logs",
        component: () => import("@/views/Logs.vue"),
        meta: { title: "操作日志", icon: "Document" },
      },
      {
        path: "profile",
        name: "Profile",
        component: () => import("@/views/Profile.vue"),
        meta: { title: "个人中心" },
      },
      {
        path: "settings",
        name: "Settings",
        component: () => import("@/views/Settings.vue"),
        meta: { title: "系统设置" },
      },
    ],
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach((to, from, next) => {
  document.title = `${to.meta.title || ""} - 智能体实训台`;
  const userStore = useUserStore();
  if (to.path !== "/login" && !userStore.token) {
    next("/login");
  } else {
    next();
  }
});

export default router;
