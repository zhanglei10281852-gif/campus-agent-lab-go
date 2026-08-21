import request from "./request";

const toLog = (row) => ({
  ...row,
  username: row.actor || row.Actor,
  module: row.entity_type || row.EntityType,
  action: row.action || row.Action,
  method: row.entity_id || row.EntityID,
  costTime: 0,
  createTime: row.created_at || row.CreatedAt,
});

export const getLogPage = async (params) => {
  const data = await request.get("/audit", {
    params: {
      limit: params.size,
      offset: Math.max(0, (params.current - 1) * params.size),
      actor: params.username || "",
    },
  });
  const items = data.items || data.Items || [];
  const total = data.total ?? data.Total ?? 0;
  return { records: items.map(toLog), total, current: params.current, size: params.size };
};

export const getTodayLogCount = async () => {
  const data = await request.get("/audit", { params: { limit: 1, offset: 0 } });
  return data.total ?? data.Total ?? 0;
};
