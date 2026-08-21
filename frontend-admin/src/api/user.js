import request from "./request";

const toUser = (row) => ({
  ...row,
  username: row.email,
  nickname: row.display_name,
  role: row.role === "agent_developer" ? "ADMIN" : "USER",
  status: row.status === "active" ? 1 : 0,
  createTime: row.created_at,
});

const toPayload = (data) => ({
  email: String(data.email || data.username || "").trim().toLowerCase(),
  display_name: String(data.displayName || data.nickname || "").trim(),
  password: data.password || undefined,
  role: data.role === "ADMIN" ? "agent_developer" : "tool_operator",
  status: data.status === 0 ? "disabled" : "active",
  version: data.version,
});

export const getUserPage = async (params) => {
  const data = await request.get("/users", { params });
  return { ...data, records: (data.records || []).map(toUser) };
};
export const createUser = (data) => {
  const payload = toPayload(data);
  delete payload.status;
  delete payload.version;
  return request.post("/users", payload);
};
export const updateUser = (id, data) => request.put("/users/" + id, toPayload(data));
export const deleteUser = (id, version) => request.delete("/users/" + id, { params: { version } });
export const updateUserStatus = (id, status, version) =>
  request.put("/users/" + id + "/status", null, { params: { status, version } });
