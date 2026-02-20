<template>
  <div class="p-6">
    <!-- Header -->
    <div class="mb-6 pb-4 border-b border-navy-100">
      <h1 class="page-title">Complaints</h1>
      <p class="page-subtitle">Review, manage and resolve citizen complaints</p>
    </div>

    <!-- Status Filter Bar -->
    <div class="flex flex-wrap items-center gap-3 mb-6">
      <div class="flex bg-navy-100 rounded-lg p-1 gap-1">
        <button v-for="s in statusFilters" :key="s.value" @click="statusFilter = s.value"
          :class="statusFilter === s.value ? 'bg-white text-navy-800 shadow-sm' : 'text-navy-500 hover:text-navy-700'"
          class="px-4 py-1.5 text-sm font-medium rounded-md transition">
          {{ s.label }}
        </button>
      </div>
      <select v-if="authStore.canManageDepartments" v-model="deptFilter" @change="loadComplaints"
        class="form-input w-auto text-sm">
        <option value="">All Departments</option>
        <option v-for="d in departments" :key="d.id" :value="d.id">{{ d.name }}</option>
      </select>
    </div>

    <!-- Table -->
    <div class="card overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="table-header">
            <th class="text-left px-4 py-3">ID</th>
            <th class="text-left px-4 py-3">Category</th>
            <th class="text-left px-4 py-3">Description</th>
            <th class="text-left px-4 py-3">Status</th>
            <th class="text-left px-4 py-3">Priority</th>
            <th class="text-left px-4 py-3">Date</th>
            <th class="text-left px-4 py-3">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in filteredComplaints" :key="c.id || c.ID" class="table-row">
            <td class="px-4 py-3 font-mono text-xs text-navy-500">#{{ c.id || c.ID }}</td>
            <td class="px-4 py-3 font-medium text-navy-700">{{ c.category }}</td>
            <td class="px-4 py-3 text-navy-600">{{ truncate(c.description, 60) }}</td>
            <td class="px-4 py-3">
              <span class="badge" :class="statusClass(c.status || c.Status)">{{ c.status || c.Status }}</span>
            </td>
            <td class="px-4 py-3">
              <span class="text-xs font-semibold" :class="priorityColor(c.priority)">{{ c.priority || 'Normal' }}</span>
            </td>
            <td class="px-4 py-3 text-navy-400 text-xs">{{ formatDate(c.created_at || c.CreatedAt) }}</td>
            <td class="px-4 py-3">
              <div class="flex gap-2">
                <button @click="openDetail(c)" class="text-navy-500 hover:text-navy-700" title="View Details">
                  <i class="bi bi-eye"></i>
                </button>
                <button v-if="authStore.canManageDepartments" @click="openReassign(c)" class="text-navy-500 hover:text-navy-700" title="Reassign">
                  <i class="bi bi-arrow-clockwise"></i>
                </button>
              </div>
            </td>
          </tr>
          <tr v-if="!filteredComplaints.length">
            <td colspan="7" class="text-center py-12 text-navy-300">
              <i class="bi bi-exclamation-triangle text-3xl mb-2 block"></i>
              <p class="text-sm">No complaints found</p>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Detail Modal -->
    <div v-if="showDetail" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" @click.self="showDetail = false">
      <div class="bg-white rounded-xl shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        <div class="p-6 border-b border-navy-100 flex items-center justify-between">
          <div>
            <h2 class="font-serif font-bold text-navy-800 text-lg">Complaint #{{ selected.id || selected.ID }}</h2>
            <span class="badge mt-1" :class="statusClass(selected.status || selected.Status)">{{ selected.status || selected.Status }}</span>
          </div>
          <button @click="showDetail = false" class="text-navy-400 hover:text-navy-600 text-xl">&times;</button>
        </div>
        <div class="p-6 space-y-5">
          <!-- Info -->
          <div class="grid grid-cols-2 gap-4 text-sm">
            <div>
              <p class="text-navy-400 text-xs uppercase tracking-wider mb-1">Category</p>
              <p class="text-navy-700 font-medium">{{ selected.category }}</p>
            </div>
            <div>
              <p class="text-navy-400 text-xs uppercase tracking-wider mb-1">Priority</p>
              <p class="font-medium" :class="priorityColor(selected.priority)">{{ selected.priority || 'Normal' }}</p>
            </div>
            <div class="col-span-2">
              <p class="text-navy-400 text-xs uppercase tracking-wider mb-1">Description</p>
              <p class="text-navy-700">{{ selected.description }}</p>
            </div>
            <div v-if="selected.location">
              <p class="text-navy-400 text-xs uppercase tracking-wider mb-1">Location</p>
              <p class="text-navy-600 text-sm">{{ selected.location }}</p>
            </div>
            <div>
              <p class="text-navy-400 text-xs uppercase tracking-wider mb-1">Filed On</p>
              <p class="text-navy-600 text-sm">{{ formatDate(selected.created_at || selected.CreatedAt) }}</p>
            </div>
          </div>

          <!-- Status Update -->
          <div class="border-t border-navy-100 pt-4">
            <label class="form-label">Update Status</label>
            <div class="flex gap-2 mt-1">
              <select v-model="updateStatus" class="form-input flex-1">
                <option value="pending">Pending</option>
                <option value="in_progress">In Progress</option>
                <option value="resolved">Resolved</option>
                <option value="rejected">Rejected</option>
              </select>
              <button @click="changeStatus" class="btn-primary">Update</button>
            </div>
          </div>

          <!-- Action History Timeline -->
          <div class="border-t border-navy-100 pt-4">
            <h3 class="font-serif font-bold text-navy-800 mb-3">Action History</h3>
            <div v-if="actions.length === 0" class="text-navy-300 text-sm">No actions recorded yet</div>
            <div v-else class="space-y-3">
              <div v-for="(a, i) in actions" :key="i" class="flex gap-3">
                <div class="flex flex-col items-center">
                  <div class="w-3 h-3 rounded-full bg-navy-400 mt-1"></div>
                  <div v-if="i < actions.length - 1" class="w-0.5 flex-1 bg-navy-200"></div>
                </div>
                <div class="pb-3">
                  <p class="text-sm text-navy-700">{{ a.description }}</p>
                  <div class="flex items-center gap-3 mt-1">
                    <span class="text-xs text-navy-400">{{ formatDate(a.created_at) }}</span>
                    <span v-if="a.completion != null" class="text-xs font-medium text-navy-500">{{ a.completion }}% complete</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Add Action -->
          <div class="border-t border-navy-100 pt-4">
            <h3 class="font-serif font-bold text-navy-800 mb-3">Add Action</h3>
            <div class="space-y-3">
              <div>
                <label class="form-label">Action Description</label>
                <textarea v-model="actionForm.description" rows="2" class="form-input" placeholder="Describe action taken..."></textarea>
              </div>
              <div>
                <label class="form-label">Completion %</label>
                <input v-model.number="actionForm.completion" type="number" min="0" max="100" class="form-input w-32" placeholder="0-100" />
              </div>
              <button @click="addAction" class="btn-primary">
                <i class="bi bi-plus-lg mr-1"></i> Add Action
              </button>
            </div>
          </div>

          <!-- Reassign (Manager only) -->
          <div v-if="authStore.canManageDepartments" class="border-t border-navy-100 pt-4">
            <h3 class="font-serif font-bold text-navy-800 mb-3">Reassign Department</h3>
            <div class="flex gap-2">
              <select v-model="reassignDept" class="form-input flex-1">
                <option value="">Select department</option>
                <option v-for="d in departments" :key="d.id" :value="d.id">{{ d.name }}</option>
              </select>
              <button @click="reassignComplaint" class="btn-secondary">
                <i class="bi bi-arrow-clockwise mr-1"></i> Reassign
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Reassign Quick Modal -->
    <div v-if="showReassign" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" @click.self="showReassign = false">
      <div class="bg-white rounded-xl shadow-xl max-w-md w-full p-6">
        <h2 class="font-serif font-bold text-navy-800 text-lg mb-4">Reassign Complaint #{{ reassignTarget?.id || reassignTarget?.ID }}</h2>
        <div>
          <label class="form-label">Target Department</label>
          <select v-model="reassignDept" class="form-input w-full">
            <option value="">Select department</option>
            <option v-for="d in departments" :key="d.id" :value="d.id">{{ d.name }}</option>
          </select>
        </div>
        <div class="flex justify-end gap-3 mt-5">
          <button @click="showReassign = false" class="btn-secondary">Cancel</button>
          <button @click="reassignFromModal" class="btn-primary">
            <i class="bi bi-arrow-clockwise mr-1"></i> Reassign
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import api from '../api'

const authStore = useAuthStore()
const complaints = ref([])
const departments = ref([])
const statusFilter = ref('')
const deptFilter = ref('')
const showDetail = ref(false)
const showReassign = ref(false)
const selected = ref({})
const reassignTarget = ref(null)
const reassignDept = ref('')
const updateStatus = ref('')
const actions = ref([])
const actionForm = ref({ description: '', completion: 0 })

const statusFilters = [
  { label: 'All', value: '' },
  { label: 'Pending', value: 'pending' },
  { label: 'In Progress', value: 'in_progress' },
  { label: 'Resolved', value: 'resolved' },
  { label: 'Rejected', value: 'rejected' },
]

const filteredComplaints = computed(() => {
  if (!statusFilter.value) return complaints.value
  return complaints.value.filter(c => (c.status || c.Status) === statusFilter.value)
})

function truncate(str, len) {
  if (!str) return ''
  return str.length > len ? str.substring(0, len) + '...' : str
}

function statusClass(s) {
  const map = { pending: 'badge-warning', in_progress: 'badge-info', resolved: 'badge-success', rejected: 'badge-danger' }
  return map[s] || 'badge-info'
}

function priorityColor(p) {
  const map = { high: 'text-red-600', medium: 'text-amber-600', low: 'text-green-600' }
  return map[p?.toLowerCase()] || 'text-navy-500'
}

function formatDate(d) {
  if (!d) return ''
  return new Date(d).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' })
}

function openDetail(c) {
  selected.value = { ...c }
  updateStatus.value = c.status || c.Status || 'pending'
  reassignDept.value = ''
  actionForm.value = { description: '', completion: 0 }
  loadActions(c.id || c.ID)
  showDetail.value = true
}

function openReassign(c) {
  reassignTarget.value = c
  reassignDept.value = ''
  showReassign.value = true
}

async function loadComplaints() {
  try {
    const params = {}
    if (deptFilter.value) params.department_id = deptFilter.value
    const { data } = await api.get('/api/v1/complaints', { params })
    complaints.value = data.complaints || data || []
  } catch {}
}

async function loadDepartments() {
  if (!authStore.canManageDepartments) return
  try {
    const { data } = await api.get('/api/v1/admin/departments')
    departments.value = data.departments || data || []
  } catch {}
}

async function loadActions(id) {
  actions.value = []
  try {
    const { data } = await api.get('/api/v1/complaints/' + id)
    actions.value = data.actions || []
  } catch {}
}

async function changeStatus() {
  const id = selected.value.id || selected.value.ID
  try {
    await api.put('/api/v1/complaints/' + id, { status: updateStatus.value })
    selected.value.status = updateStatus.value
    selected.value.Status = updateStatus.value
    await loadComplaints()
  } catch {}
}

async function addAction() {
  if (!actionForm.value.description) return
  const id = selected.value.id || selected.value.ID
  try {
    await api.post('/api/v1/complaints/' + id + '/actions', actionForm.value)
    actionForm.value = { description: '', completion: 0 }
    await loadActions(id)
  } catch {}
}

async function reassignComplaint() {
  if (!reassignDept.value) return
  const id = selected.value.id || selected.value.ID
  try {
    await api.put('/api/v1/complaints/' + id + '/reassign', { department_id: reassignDept.value })
    showDetail.value = false
    await loadComplaints()
  } catch {}
}

async function reassignFromModal() {
  if (!reassignDept.value || !reassignTarget.value) return
  const id = reassignTarget.value.id || reassignTarget.value.ID
  try {
    await api.put('/api/v1/complaints/' + id + '/reassign', { department_id: reassignDept.value })
    showReassign.value = false
    await loadComplaints()
  } catch {}
}

onMounted(() => {
  loadComplaints()
  loadDepartments()
})
</script>
