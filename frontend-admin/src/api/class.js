import request from "./request";

const toClass = (row) => ({ ...row, className: row.name, teacher: row.instructor, studentCount: row.student_count })
export const getClassPage = async (params) => {
  const data = await request.get("/cohorts", { params: { ...params, search: params.className || "" } })
  return { ...data, records: (data.records || []).map(toClass) }
}
export const getAllClasses = async () => (await request.get("/cohorts/all") || []).map(toClass)
export const createClass = (data) => request.post("/cohorts", {
  code: data.code || data.className, name: data.className, grade: data.grade,
  instructor: data.teacher, capacity: Number(data.capacity || 40), status: "active",
});
export const updateClass = (id, data) => request.put(`/cohorts/${id}`, {
  code: data.code || data.className, name: data.className, grade: data.grade,
  instructor: data.teacher, capacity: Number(data.capacity || 40), status: data.status || "active",
  version: data.version,
});
export const deleteClass = (id, version) => request.delete(`/cohorts/${id}`, { params: { version } });
