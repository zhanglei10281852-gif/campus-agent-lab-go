import request from "./request";

export const login = (data) => request.post("/auth/login", {
  email: data.email || data.username,
  password: data.password,
});
export const logout = () => request.post("/auth/logout");
export const getUserInfo = () => request.get("/auth/me");
export const updateProfile = (data) => request.put("/auth/profile", {
  display_name: data.nickname || data.displayName || "",
});
export const updatePassword = (data) => request.put("/auth/password", {
  old_password: data.oldPassword,
  new_password: data.newPassword,
});
