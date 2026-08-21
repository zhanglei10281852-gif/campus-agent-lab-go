import { ElMessage } from "element-plus";
import { client } from "./client";

const unwrap = (promise) => promise.then((response) => response.data).catch((error) => {
  if (error.status === 401) {
    localStorage.removeItem("token");
    localStorage.removeItem("userInfo");
    window.dispatchEvent(new CustomEvent("session-expired"));
  }
  ElMessage.error(error.message || "请求失败");
  return Promise.reject(error);
});

const request = {
  get: (url, config) => unwrap(client.get(url, config)),
  post: (url, body, config) => unwrap(client.post(url, body, config)),
  put: (url, body, config) => unwrap(client.put(url, body, config)),
  delete: (url, config) => unwrap(client.delete(url, config)),
};

export default request;
