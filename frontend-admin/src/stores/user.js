import { defineStore } from "pinia";
import { ref } from "vue";
import { login as loginApi, logout as logoutApi, getUserInfo as getUserInfoApi } from "@/api/auth";

export const useUserStore = defineStore("user", () => {
  const token = ref(localStorage.getItem("token") || "");
  const userInfo = ref(JSON.parse(localStorage.getItem("userInfo") || "{}"));

  const login = async (loginForm) => {
    const data = await loginApi(loginForm);
    token.value = data.token;
    userInfo.value = data.user || data;
    localStorage.setItem("token", data.token);
    localStorage.setItem("userInfo", JSON.stringify(userInfo.value));
    return data;
  };

  const logout = async () => {
    if (token.value) {
      try { await logoutApi(); } catch { /* token is cleared locally even if the server is unavailable */ }
    }
    token.value = "";
    userInfo.value = {};
    localStorage.removeItem("token");
    localStorage.removeItem("userInfo");
  };

  const fetchUserInfo = async () => {
    const data = await getUserInfoApi();
    userInfo.value = { ...userInfo.value, ...data };
    localStorage.setItem("userInfo", JSON.stringify(userInfo.value));
  };

  return { token, userInfo, login, logout, fetchUserInfo };
});
