<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">Complaints</h1>
        <p class="text-gray-500 text-sm mt-1">{{ authStore.isDeptManager ? 'Your department complaints' : 'All government complaints' }}</p>
      </div>
    </div>

    <!-- Filters -->
    <div class="bg-white rounded-xl border border-gray-200 p-4 mb-6">
      <div class="flex flex-wrap gap-3 items-center">
        <div class="flex bg-gray-100 rounded-lg p-1">
          <button v-for="s in ['all', 'pending', 'in_progress', 'resolved', 'rejected']" :key="s"
            @click="statusFilter = s" :class="statusFilter === s ? 'bg-white shadow text-gray-900' : 'text-gray-500'"
            class="px-3 py-1.5 text-sm font-medium rounded-md transition capitalize">
            {{ s === 'all' ? 'All' : s.replace('_', ' ') }}
          </button>
        </div>
        <select v-if="authStore.isManager" v-model="deptFilter" class="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-civic-500 focus:border-civic-500">
          <option value="">All Departments</option>
          <option v-for="d in departments" :key="d.id" :value="d.id">{{ d.name }}</option>
        </select>
        <input v-model="search" type="text" placeholder="Search complaints..." class="border border-gray-300 rounded-lg px-3 py-2 text-sm flex-1 min-w-[200px] focus:ring-civic-500 focus:border-civic-500" />
      </div>
    </div>

    <!-- Complaints Table -->
    <div class="bg-white rounded-xl border border-gray-200 overflow-hidden">
      <div v-if="loading" class="p-8 text-center text-gray-400">Loading complaints...</div>
      <table v-else class="w-full text-sm">
        <thead class="bg-gray-50 border-b">
          <tr>
            <th class="text-left px-4 py-3 text-gray-600 font-medium">ID</th>
            <th class="text-left px-4 py-3 text-gray-600 font-medium">Category</th>
            <th class="text-left px-4 py-3 text-gray-600 font-medium">Description</th>
            <th class="text-left px-4 py-3 text-gray-600 font-medium">Status</th>
            <th class="text-left px-4 py-3 text-gray-600 font-medium">Priority</th>
            <th class="text-left px-4 py-3 text-gray-600 font-medium">Date</th>
            <th class="text-left px-4 py-3 text-gray-600 font-medium">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in filteredComplaints" :key="c.id" class="border-t hover:bg-gray-50 cursor-pointer" @click="openDetail(c)">
            <td class="px-4 py-3 font-mono text-gray-800">#{{ c.id }}</td>
            <td class="px-4 py-3"><span class="px-2 py-0.5 rounded-full text-xs bg-blue-100 text-blue-700">{{ c.category }}</span></td>
            <td class="px-4 py-3 text-gray-700 max-w-xs truncate">{{ c.description }}</td>
            <td class="px-4 py-3">
              <select v-model="c.status" @click.stop @change="updateStatus(c)" class="text-xs border rounded px-2 py-1" :class="statusClass(c.status)">
                <option value="pending">Pending</option>
                <option value="in_progress">In Progress</option>
                <option value="resolved">Resolved</option>
                <option value="rejected">Rejected</option>
              </select>
            </td>
            <td class="px-4 py-3">
              <span class="font-mono text-sm" :class="(c.upvotes - c.downvotes * 2) > 0 ? 'text-green-600' : 'text-red-600'">{{ c.upvotes - (c.downvotes * 2) }}</span>
            </td>
            <td class="px-4 py-3 text-gray-500 text-xs">{{ new Date(c.created_at).toLocaleDateString() }}</td>
            <td class="px-4 py-3" @click.stop>
              <button @click="openDetail(c)" class="text-civic-600 hover:text-civic-700 text-sm font-medium">View</button>
            </td>
          </tr>
          <tr v-if="filteredComplaints.length === 0"><td colspan="7" class="px-4 py-12 text-center text-gray-400">No complaints found</td></tr>
        </tbody>
      </table>
    </div>

    <!-- Detail Modal -->
    <div v-if="selectedComplaint" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" @click.self="selectedComplaint = null">
      <div class="bg-white rounded-2xl shadow-xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        <div class="p-6 border-b flex items-center justify-between">
          <h2 class="text-xl font-bold text-gray-900">Complaint #{{ selectedComplaint.id }}</h2>
          <button @click="selectedComplaint = null" class="text-gray-400 hover:text-gray-600">✕</button>
        </div>
        <div class="p-6 space-y-4">
          <div><label class="text-sm font-medium text-gray-500">Category</label><p class="text-gray-900 mt-1">{{ selectedComplaint.category }}</p></div>
          <div><label class="text-sm font-medium text-gray-500">Description</label><p class="text-gray-900 mt-1 whitespace-pre-wrap">{{ selectedComplaint.description }}</p></div>
          <div v-if="selectedComplaint.ai_analysis" class="bg-purple-50 border border-purple-200 rounded-lg p-4">
            <label class="text-sm font-medium text-purple-700">AI Analysis</label>
            <p class="text-purple-900 mt-1 text-sm">{{ selectedComplaint.ai_analysis }}</p>
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div><label class="text-sm font-medium text-gray-500">Status</label><p class="mt-1"><span :class="statusClass(selectedComplaint.status)" class="text-xs px-2.5 py-1 rounded-full font-medium">{{ selectedComplaint.status }}</span></p></div>
            <div><label class="text-sm font-medium text-gray-500">Priority Score</label><p class="text-gray-900 mt-1 font-mono">{{ selectedComplaint.upvotes - (selectedComplaint.downvotes * 2) }}</p></div>
          </div>
          <div v-if="selectedComplaint.manual_location"><label class="text-sm font-medium text-gray-500">Location</label><p class="text-gray-900 mt-1">{{ selectedComplaint.manual_location }}</p></div>

          <!-- Actions Taken -->
          <div class="border-t pt-4 mt-4">
            <h3 class="text-lg font-semibold text-gray-900 mb-3">Actions Taken</h3>
            <div v-if="complaintActions.length === 0" class="text-gray-400 text-sm">No actions yet</div>
            <div v-for="a in complaintActions" :key="a.id" class="bg-gray-50 rounded-lg p-3 mb-2">
              <p class="text-sm text-gray-900">{{ a.action_details }}</p>
              <div class="flex items-center mt-2">
                <div class="flex-1 bg-gray-200 rounded-full h-2 mr-3"><div class="bg-civic-600 h-2 rounded-full" :style="{ width: a.completion_percentage + '%' }"></div></div>
                <span class="text-xs text-gray-600 font-medium">{{ a.completion_percentage }}%</span>
              </div>
            </div>

            <div v-if="authStore.isDeptManager || authStore.isManager" class="mt-4 bg-blue-50 rounded-lg p-4">
              <h4 class="text-sm font-semibold text-gray-900 mb-3">Add Action</h4>
              <textarea v-model="newAction.details" placeholder="Describe the action taken..." class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm mb-3" rows="2"></textarea>
              <div class="flex items-center gap-3">
                <label class="text-sm text-gray-600">Completion:</label>
                <input v-model.number="newAction.completion" type="range" min="0" max="100" step="10" class="flex-1" />
                <span class="text-sm font-medium text-gray-700 w-10 text-right">{{ newAction.completion }}%</span>
                <button @click="addAction" :disabled="!newAction.details" class="bg-civic-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-civic-700 disabled:opacity-50 transition">Add</button>
              </div>
            </div>
          </div>
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
const loading = ref(true)
const complaints = ref([])
const departments = ref([])
const statusFilter = ref('all')
const deptFilter = ref('')
const search = ref('')
const selectedComplaint = ref(null)
const complaintActions = ref([])
const newAction = ref({ details: '', completion: 0 })

const filteredComplaints = computed(() => {
  return complaints.value.filter(c => {
    if (statusFilter.value !== 'all' && c.status !== statusFilter.value) return false
    if (deptFilter.value && c.department_id != deptFilter.value) return false
    if (search.value) {
      const q = search.value.toLowerCase()
      return c.category?.toLowerCase().includes(q) || c.description?.toLowerCase().includes(q)
    }
    return true
  })
})

function statusClass(status) {
  const c = { pending: 'bg-yellow-100 text-yellow-800', in_progress: 'bg-blue-100 text-blue-800', resolved: 'bg-green-100 text-green-800', rejected: 'bg-red-100 text-red-800' }
  return c[status] || 'bg-gray-100 text-gray-800'
}

async function updateStatus(complaint) {
  try { await api.put(`/api/v1/complaints/${complaint.id}`, { status: complaint.status }) } catch { alert('Failed to update status') }
}

async function openDetail(complaint) {
  selectedComplaint.value = complaint
  try {
    const { data } = await api.get(`/api/v1/complaints/${complaint.id}/actions`)
    complaintActions.value = Array.isArray(data) ? data : []
  } catch { complaintActions.value = [] }
}

async function addAction() {
  if (!newAction.value.details || !selectedComplaint.value) return
  try {
    await api.post(`/api/v1/complaints/${selectedComplaint.value.id}/actions`, {
      government_id: authStore.user?.government_id,
      admin_id: authStore.user?.id,
      action_details: newAction.value.details,
      completion_percentage: newAction.value.completion,
    })
    newAction.value = { details: '', completion: 0 }
    const { data } = await api.get(`/api/v1/complaints/${selectedComplaint.value.id}/actions`)
    complaintActions.value = Array.isArray(data) ? data : []
    fetchComplaints()
  } catch { alert('Failed to add action') }
}

async function fetchComplaints() {
  loading.value = true
  try {
    const params = {}
    const govId = authStore.user?.government_id
    if (govId) params.government_id = govId
    if (authStore.isDeptManager && authStore.user?.department_id) params.department_id = authStore.user.department_id
    const { data } = await api.get('/api/v1/complaints/', { params })
    complaints.value = Array.isArray(data) ? data : []
  } catch { complaints.value = [] }
  finally { loading.value = false }
}

onMounted(async () => {
  fetchComplaints()
  if (authStore.isManager) {
    try { const { data } = await api.get('/api/v1/admin/departments'); departments.value = Array.isArray(data) ? data : [] } catch {}
  }
})
</script>
