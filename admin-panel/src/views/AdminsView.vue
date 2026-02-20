<template>
  <div class="p-6">
    <!-- Header -->
    <div class="mb-6 pb-4 border-b border-navy-100 flex items-center justify-between">
      <div>
        <h1 class="page-title">{{ authStore.isSuperAdmin ? 'Managers' : 'Department Managers' }}</h1>
        <p class="page-subtitle">{{ authStore.isSuperAdmin ? 'Manage municipality managers across the platform' : 'Manage department managers for ' + (authStore.governmentName || 'your municipality') }}</p>
      </div>
      <button @click="openModal()" class="btn-primary">
        <i class="bi bi-plus-lg mr-1"></i> Add {{ authStore.isSuperAdmin ? 'Manager' : 'Dept Manager' }}
      </button>
    </div>

    <!-- Table -->
    <div class="card overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="table-header">
            <th class="text-left px-4 py-3">Name</th>
            <th class="text-left px-4 py-3">Email</th>
            <th class="text-left px-4 py-3">Role</th>
            <th class="text-left px-4 py-3">{{ authStore.isSuperAdmin ? 'Municipality' : 'Department' }}</th>
            <th class="text-left px-4 py-3">Created</th>
            <th class="text-left px-4 py-3">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in admins" :key="a.id" class="table-row">
            <td class="px-4 py-3">
              <div class="flex items-center gap-3">
                <div class="w-8 h-8 bg-navy-100 rounded-full flex items-center justify-center text-xs font-bold text-navy-600">
                  {{ a.name?.[0]?.toUpperCase() || '?' }}
                </div>
                <span class="font-medium text-navy-700">{{ a.name }}</span>
              </div>
            </td>
            <td class="px-4 py-3 text-navy-500">{{ a.email }}</td>
            <td class="px-4 py-3">
              <span class="badge" :class="roleClass(a.role)">{{ roleLabel(a.role) }}</span>
            </td>
            <td class="px-4 py-3 text-navy-500 text-sm">{{ a.government_name || a.department_name || '—' }}</td>
            <td class="px-4 py-3 text-navy-400 text-xs">{{ formatDate(a.created_at) }}</td>
            <td class="px-4 py-3">
              <button @click="confirmDelete(a)" class="text-red-500 hover:text-red-700" title="Delete">
                <i class="bi bi-trash"></i>
              </button>
            </td>
          </tr>
          <tr v-if="!admins.length">
            <td colspan="6" class="text-center py-12 text-navy-300">
              <i class="bi bi-shield-check text-3xl mb-2 block"></i>
              <p class="text-sm">No managers found</p>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" @click.self="showModal = false">
      <div class="bg-white rounded-xl shadow-xl max-w-lg w-full">
        <div class="p-6 border-b border-navy-100 flex items-center justify-between">
          <h2 class="font-serif font-bold text-navy-800 text-lg">{{ authStore.isSuperAdmin ? 'Add Manager' : 'Add Department Manager' }}</h2>
          <button @click="showModal = false" class="text-navy-400 hover:text-navy-600 text-xl">&times;</button>
        </div>
        <form @submit.prevent="createAdmin" class="p-6 space-y-4">
          <div>
            <label class="form-label">Full Name</label>
            <input v-model="form.name" required class="form-input" placeholder="Enter full name" />
          </div>
          <div>
            <label class="form-label">Email Address</label>
            <input v-model="form.email" type="email" required class="form-input" placeholder="manager@govt.in" />
          </div>
          <div>
            <label class="form-label">Password</label>
            <input v-model="form.password" type="password" required class="form-input" placeholder="Minimum 8 characters" />
          </div>
          <div>
            <label class="form-label">Role</label>
            <select v-model="form.role" required class="form-input">
              <option v-if="authStore.isSuperAdmin" value="manager">Manager</option>
              <option v-if="authStore.isManager" value="dept_manager">Department Manager</option>
            </select>
          </div>
          <!-- Government ID for super_admin creating managers -->
          <div v-if="authStore.isSuperAdmin && form.role === 'manager'">
            <label class="form-label">Municipality</label>
            <input v-model="form.government_id" class="form-input" placeholder="Government/Municipality ID" />
          </div>
          <!-- Department dropdown for manager creating dept_managers -->
          <div v-if="authStore.isManager && form.role === 'dept_manager'">
            <label class="form-label">Department</label>
            <select v-model="form.department_id" required class="form-input">
              <option value="">Select department</option>
              <option v-for="d in departments" :key="d.id" :value="d.id">{{ d.name }}</option>
            </select>
          </div>
          <p v-if="formError" class="text-sm text-red-600">{{ formError }}</p>
          <div class="flex justify-end gap-3 pt-2">
            <button type="button" @click="showModal = false" class="btn-secondary">Cancel</button>
            <button type="submit" class="btn-primary">
              <i class="bi bi-person-badge mr-1"></i> Create
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Delete Confirmation -->
    <div v-if="showDelete" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" @click.self="showDelete = false">
      <div class="bg-white rounded-xl shadow-xl max-w-sm w-full p-6">
        <h2 class="font-serif font-bold text-navy-800 text-lg mb-2">Remove Manager</h2>
        <p class="text-navy-500 text-sm mb-5">Are you sure you want to remove <strong>{{ deleteTarget?.name }}</strong>? They will lose all admin access.</p>
        <div class="flex justify-end gap-3">
          <button @click="showDelete = false" class="btn-secondary">Cancel</button>
          <button @click="deleteAdmin" class="btn-danger">Remove</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import api from '../api'

const authStore = useAuthStore()
const admins = ref([])
const departments = ref([])
const showModal = ref(false)
const showDelete = ref(false)
const deleteTarget = ref(null)
const formError = ref('')
const form = ref({ name: '', email: '', password: '', role: '', government_id: '', department_id: '' })

function roleClass(r) {
  const map = { super_admin: 'badge-danger', manager: 'badge-info', dept_manager: 'badge-success' }
  return map[r] || 'badge-info'
}

function roleLabel(r) {
  const map = { super_admin: 'Super Admin', manager: 'Manager', dept_manager: 'Dept Manager' }
  return map[r] || r
}

function formatDate(d) {
  if (!d) return ''
  return new Date(d).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' })
}

function openModal() {
  formError.value = ''
  form.value = {
    name: '', email: '', password: '',
    role: authStore.isSuperAdmin ? 'manager' : 'dept_manager',
    government_id: '', department_id: ''
  }
  showModal.value = true
}

function confirmDelete(admin) {
  deleteTarget.value = admin
  showDelete.value = true
}

async function loadAdmins() {
  try {
    const { data } = await api.get('/api/v1/admin/admins')
    admins.value = data.admins || data || []
  } catch {}
}

async function loadDepartments() {
  if (!authStore.isManager) return
  try {
    const { data } = await api.get('/api/v1/admin/departments')
    departments.value = data.departments || data || []
  } catch {}
}

async function createAdmin() {
  formError.value = ''
  const payload = { name: form.value.name, email: form.value.email, password: form.value.password, role: form.value.role }
  if (form.value.government_id) payload.government_id = form.value.government_id
  if (form.value.department_id) payload.department_id = form.value.department_id
  try {
    await api.post('/api/v1/admin/admins', payload)
    showModal.value = false
    await loadAdmins()
  } catch (e) {
    formError.value = e.response?.data?.error || 'Failed to create admin'
  }
}

async function deleteAdmin() {
  if (!deleteTarget.value) return
  try {
    await api.delete('/api/v1/admin/admins/' + deleteTarget.value.id)
    showDelete.value = false
    deleteTarget.value = null
    await loadAdmins()
  } catch {}
}

onMounted(() => {
  loadAdmins()
  loadDepartments()
})
</script>
