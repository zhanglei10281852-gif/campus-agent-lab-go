import request from "./request";

const toStudent = (row) => ({ ...row, studentNo: row.student_no, birthDate: row.birth_date, classId: row.cohort_id, className: row.cohort_name, gender: row.gender === 'female' ? 2 : 1, status: ({ active: 1, suspended: 0, completed: 2 }[row.status] || 1) })
const toPayload = (data) => ({
  student_no: String(data.studentNo || '').trim(),
  name: String(data.name || '').trim(),
  gender: data.gender === 2 ? 'female' : 'male',
  birth_date: data.birthDate || '',
  phone: data.phone || '',
  email: String(data.email || '').trim().toLowerCase(),
  cohort_id: data.classId || '',
  status: ({ 0: 'suspended', 1: 'active', 2: 'completed' }[data.status] || 'active'),
})
export const getStudentPage = async (params) => {
  const data = await request.get("/trainees", { params: { ...params, student_no: params.studentNo, cohort_id: params.classId } })
  return { ...data, records: (data.records || []).map(toStudent) }
}
export const getStudentById = async (id) => toStudent(await request.get(`/trainees/${id}`))
export const createStudent = (data) => request.post("/trainees", toPayload(data));
export const updateStudent = (id, data) => request.put(`/trainees/${id}`, { ...toPayload(data), version: data.version });
export const deleteStudent = (id, version) => request.delete(`/trainees/${id}`, { params: { version } });
