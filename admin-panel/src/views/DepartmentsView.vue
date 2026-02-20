<template>
  <div class="p-6">
    <!-- Header -->
    <div class="mb-6 pb-4 border-b border-navy-100 flex items-center justify-between">
      <div>
        <h1 class="page-title">Departments</h1>
        <p class="page-subtitle">Manage departments under {{ authStore.governmentName || 'your municipality' }}</p>
      </div>
      <button @click="openModal()" class="btn-primary">
        <i class="bi bi-plus-lg mr-1"></i> Add Department
      </button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="text-center py-12 text-navy-300">Loading departments...</div>

    <!-- Empty State -->
    <div v-else-if="departments.length === 0" class="card p-12 text-center">
      <i class="bi bi-diagram-3 text-4xl text-navy-200 block mb-3"></i>
      <p class="text-navy-400">No departments yet. Create your first department.</p>
    </div>

    <!-- Card Grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
      <div v-for="d in departments" :key="d.id" class="card hover:shadow-md transition">
        <div class="p-5">
          <div class="flex items-start gap-4">
            <div class="w-14 h-14 rounded-lg bg-navy-100 flex items-center justify-center flex-shrink-0 overflow-hidden">
              <img v-if="d.logo" :src="d.logo" :alt="d.name" class="w-full h-full object-cover" />
              <i v-else class="bi bi-diagram-3 text-2xl text-navy-400"></i>
            </div>
            <div class="flex-1 min-w-0">
              <h3 class="font-serif font-bold text-navy-800 text-lg truncate">{{ d.name }}</h3>
              <div class="mt-2 space-y-1 text-sm text-navy-500">
                <p v-if="d.email" class="flex items-center gap-2">
                  <i class="bi bi-envelope text-navy-400"></i>
                  <span class="truncate">{{ d.email }}</span>
                </p>
                <p v-if="d.phone" class="flex items-center gap-2">
                  <i class="bi bi-telephone text-navy-400"></i>
                  {{ d.phone }}
                </p>
              </div>
            </div>
          </div>

          <!-- Manager Count -->
          <div class="mt-4 flex items-center gap-2 text-xs text-navy-400">
            <i class="bi bi-person-badge"></i>
            <span>{{ d.manager_count || 0 }} manager{{ (d.manager_count || 0) !== 1 ? 's' : '' }}</span>
          </div>

          <!-- Services -->
          <div v-if="d.services && d.services.length" class="mt-3 flex flex-wrap gap-1">
            <span v-for="s in displayServices(d.services)" :key="s" class="px-2 py-0.5 bg-navy-100 text-navy-600 text-xs rounded-full">{{ s }}</span>
          </div>
        </div>

        <!-- Actions -->
        <div class="px-5 py-3 border-t border-navy-100 flex justify-end gap-2">
          <button @click="openModal(d)" class="text-navy-500 hover:text-navy-700 text-sm font-medium flex items-center gap-1">
            <i class="bi bi-pencil"></i> Edit
          </button>
          <button @click="confirmDelete(d)" class="text-red-500 hover:text-red-700 text-sm font-medium flex items-center gap-1">
            <i class="bi bi-trash"></i> Delete
          </button>
        </div>
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" @click.self="showModal = false">
      <div class="bg-white rounded-xl shadow-xl max-w-lg w-full">
        <div class="p-6 border-b border-navy-100 flex items-center justify-between">
          <h2 class="font-serif font-bold text-navy-800 text-lg">{{ editingId ? 'Edit Department' : 'New Department' }}</h2>
          <button @click="showModal = false" class="text-navy-400 hover:text-navy-600 text-xl">&times;</button>
        </div>
        <form @submit.prevent="saveDepartment" class="p-6 space-y-4">
          <div>
            <label class="form-label">Department Name</label>
            <input v-model="form.name" required class="form-input" placeholder="e.g. Public Works" />
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="form-label">Email</label>
              <input v-model="form.email" type="email" class="form-input" placeholder="dept@gov.in" />
            </div>
            <div>
              <label class="form-label">Phone</label>
              <input v-model="form.phone" class="form-input" placeholder="+91 44 1234567" />
            </div>
          </div>
          <div>
            <label class="form-label">Services (comma-separated)</label>
            <input v-model="form.services_text" class="form-input" placeholder="Roads, Drainage, Water Supply" />
          </div>
          <div>
            <label class="form-label">Logo</label>
            <FileUpload v-model="form.logo" />
          </div>
          <p v-if="formError" class="text-sm text-red-600">{{ formError }}</p>
          <div class="flex justify-end gap-3 pt-2">
            <button type="button" @click="showModal = false" class="btn-secondary">Cancel</button>
            <button type="submit" class="btn-primary">{{ editingId ? 'Update' : 'Create' }}</button>
          </div>
        </form>
      </div>
    </div>

    <!-- Delete Confirmation -->
    <div v-if="showDelete" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" @click.self="showDelete = false">
      <div class="bg-white rounded-xl shadow-xl max-w-sm w-full p-6">
        <h2 class="font-serif font-bold text-navy-800 text-lg mb-2">Delete Department</h2>
        <p class="text-navy-500 text-sm mb-5">Are you sure you want to delete <strong>{{ deleteTarget?.name }}</strong>? This action cannot be undone.</p>
        <div class="flex justify-end gap-3">
          <button @click="showDelete = false" class="btn-secondary">Cancel</button>
          <button @click="deleteDepartment" class="btn-danger">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import api from '../api'
import FileUpload from '../components/FileUpload.vue'

const authStore = useAuthStore()
const departments = ref([])
const loading = ref(true)
const showModal = ref(false)
const showDelete = ref(false)
const editingId = ref(null)
const deleteTarget = ref(null)
const formError = ref('')
const form = ref({ name: '', email: '', phone: '', services_text: '', logo: '' })

function displayServices(services) {
  if (typeof services === 'string') return services.split(',').map(s => s.trim()).filter(Boolean)
  if (Array.isArray(services)) return services
  return []
}

function openModal(dept = null) {
  formError.value = ''
  if (dept) {
    editingId.value = dept.id
    const svc = Array.isArray(dept.services) ? dept.services.join(', ') : (dept.services || '')
    form.value = { name: dept.name, email: dept.email || '', phone: dept.phone || '', services_text: svc, logo: dept.logo || '' }
  } else {
    editingId.value = null
    form.value = { name: '', email: '', phone: '', services_text: '', logo: '' }
  }
  showModal.value = true
}

function confirmDelete(dept) {
  deleteTarget.value = dept
  showDelete.value = true
}

async function loadDepartments() {
  loading.value = true
  try {
    const { data } = await api.get('/api/v1/admin/departments')
    departments.value = data.departments || data || []
  } catch {} finally {
    loading.value = false
  }
}

async function saveDepartment() {
  formError.value = ''
  const services = form.value.services_text.split(',').map(s => s.trim()).filter(Boolean)
  const payload = { name: form.value.name, email: form.value.email, phone: form.value.phone, services, logo: form.value.logo }
  try {
    if (editingId.value) {
      await api.put('/api/v1/admin/departments/' + editingId.value, payload)
    } else {
      await api.post('/api/v1/admin/departments', payload)
    }
    showModal.value = false
    await loadDepartments()
  } catch (e) {
    formError.value = e.response?.data?.error || 'Failed to save department'
  }
}

async function deleteDepartment() {
  if (!deleteTarget.value) return
  try {
    await api.delete('/api/v1/admin/departments/' + deleteTarget.value.id)
    showDelete.value = false
    deleteTarget.value = null
    await loadDepartments()
  } catch {}
}

onMounted(loadDepartments)
</script>
