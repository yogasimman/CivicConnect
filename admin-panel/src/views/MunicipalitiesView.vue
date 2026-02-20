<template>
  <div class="p-6">
    <!-- Header -->
    <div class="mb-6 pb-4 border-b border-navy-100 flex items-center justify-between">
      <div>
        <h1 class="page-title">Municipal Corporations</h1>
        <p class="page-subtitle">Manage municipalities across the CivicConnect platform</p>
      </div>
      <button @click="openModal()" class="btn-primary">
        <i class="bi bi-plus-lg mr-1"></i> Add Municipality
      </button>
    </div>

    <!-- Stats Row -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
      <div class="card p-4">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 bg-navy-100 rounded-lg flex items-center justify-center">
            <i class="bi bi-building text-navy-600"></i>
          </div>
          <div>
            <p class="text-xs font-semibold uppercase tracking-wider text-navy-400">Municipalities</p>
            <p class="text-2xl font-bold text-navy-800">{{ municipalities.length }}</p>
          </div>
        </div>
      </div>
      <div class="card p-4">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 bg-green-100 rounded-lg flex items-center justify-center">
            <i class="bi bi-shield-check text-green-600"></i>
          </div>
          <div>
            <p class="text-xs font-semibold uppercase tracking-wider text-navy-400">Total Managers</p>
            <p class="text-2xl font-bold text-navy-800">{{ totalManagers }}</p>
          </div>
        </div>
      </div>
      <div class="card p-4">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 bg-blue-100 rounded-lg flex items-center justify-center">
            <i class="bi bi-diagram-3 text-blue-600"></i>
          </div>
          <div>
            <p class="text-xs font-semibold uppercase tracking-wider text-navy-400">Total Departments</p>
            <p class="text-2xl font-bold text-navy-800">{{ totalDepartments }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="text-center py-12 text-navy-300">Loading municipalities...</div>

    <!-- Empty State -->
    <div v-else-if="municipalities.length === 0" class="card p-12 text-center">
      <i class="bi bi-building text-4xl text-navy-200 block mb-3"></i>
      <p class="text-navy-400">No municipalities yet. Add the first one.</p>
    </div>

    <!-- Card Grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
      <div v-for="m in municipalities" :key="m.id" class="card hover:shadow-md transition">
        <div class="p-5">
          <div class="flex items-start gap-4">
            <div class="w-14 h-14 rounded-lg bg-navy-100 flex items-center justify-center flex-shrink-0 overflow-hidden">
              <img v-if="m.logo" :src="m.logo" :alt="m.name" class="w-full h-full object-cover" />
              <i v-else class="bi bi-building text-2xl text-navy-400"></i>
            </div>
            <div class="flex-1 min-w-0">
              <h3 class="font-serif font-bold text-navy-800 text-lg truncate">{{ m.name }}</h3>
              <p v-if="m.jurisdiction" class="text-sm text-navy-500 flex items-center gap-1 mt-1">
                <i class="bi bi-geo-alt text-navy-400"></i> {{ m.jurisdiction }}
              </p>
            </div>
          </div>

          <div class="mt-4 grid grid-cols-2 gap-3 text-sm">
            <div class="flex items-center gap-2 text-navy-500">
              <i class="bi bi-shield-check text-navy-400"></i>
              <span>{{ m.manager_count || 0 }} manager{{ (m.manager_count || 0) !== 1 ? 's' : '' }}</span>
            </div>
            <div class="flex items-center gap-2 text-navy-500">
              <i class="bi bi-diagram-3 text-navy-400"></i>
              <span>{{ m.department_count || 0 }} dept{{ (m.department_count || 0) !== 1 ? 's' : '' }}</span>
            </div>
          </div>
        </div>

        <!-- Actions -->
        <div class="px-5 py-3 border-t border-navy-100 flex justify-end gap-2">
          <button @click="openModal(m)" class="text-navy-500 hover:text-navy-700 text-sm font-medium flex items-center gap-1">
            <i class="bi bi-pencil"></i> Edit
          </button>
          <button @click="confirmDelete(m)" class="text-red-500 hover:text-red-700 text-sm font-medium flex items-center gap-1">
            <i class="bi bi-trash"></i> Delete
          </button>
        </div>
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" @click.self="showModal = false">
      <div class="bg-white rounded-xl shadow-xl max-w-lg w-full max-h-[90vh] overflow-y-auto">
        <div class="p-6 border-b border-navy-100 flex items-center justify-between">
          <h2 class="font-serif font-bold text-navy-800 text-lg">{{ editingId ? 'Edit Municipality' : 'New Municipality' }}</h2>
          <button @click="showModal = false" class="text-navy-400 hover:text-navy-600 text-xl">&times;</button>
        </div>
        <form @submit.prevent="saveMunicipality" class="p-6 space-y-4">
          <div>
            <label class="form-label">Municipality Name</label>
            <input v-model="form.name" required class="form-input" placeholder="e.g. Chennai Corporation" />
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="form-label">Jurisdiction</label>
              <input v-model="form.jurisdiction" class="form-input" placeholder="e.g. Chennai" />
            </div>
            <div>
              <label class="form-label">State</label>
              <input v-model="form.state" class="form-input" placeholder="e.g. Tamil Nadu" />
            </div>
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="form-label">Email</label>
              <input v-model="form.email" type="email" class="form-input" placeholder="contact@municipality.gov.in" />
            </div>
            <div>
              <label class="form-label">Phone</label>
              <input v-model="form.phone" class="form-input" placeholder="+91 44 12345678" />
            </div>
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
        <h2 class="font-serif font-bold text-navy-800 text-lg mb-2">Delete Municipality</h2>
        <p class="text-navy-500 text-sm mb-5">Are you sure you want to delete <strong>{{ deleteTarget?.name }}</strong>? All associated managers, departments and data will be removed.</p>
        <div class="flex justify-end gap-3">
          <button @click="showDelete = false" class="btn-secondary">Cancel</button>
          <button @click="deleteMunicipality" class="btn-danger">Delete</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import api from '../api'
import FileUpload from '../components/FileUpload.vue'

const authStore = useAuthStore()
const municipalities = ref([])
const loading = ref(true)
const showModal = ref(false)
const showDelete = ref(false)
const editingId = ref(null)
const deleteTarget = ref(null)
const formError = ref('')
const form = ref({ name: '', jurisdiction: '', state: '', email: '', phone: '', logo: '' })

const totalManagers = computed(() => municipalities.value.reduce((sum, m) => sum + (m.manager_count || 0), 0))
const totalDepartments = computed(() => municipalities.value.reduce((sum, m) => sum + (m.department_count || 0), 0))

function openModal(muni = null) {
  formError.value = ''
  if (muni) {
    editingId.value = muni.id
    form.value = { name: muni.name, jurisdiction: muni.jurisdiction || '', state: muni.state || '', email: muni.email || '', phone: muni.phone || '', logo: muni.logo || '' }
  } else {
    editingId.value = null
    form.value = { name: '', jurisdiction: '', state: '', email: '', phone: '', logo: '' }
  }
  showModal.value = true
}

function confirmDelete(muni) {
  deleteTarget.value = muni
  showDelete.value = true
}

async function loadMunicipalities() {
  loading.value = true
  try {
    const { data } = await api.get('/api/v1/admin/municipalities')
    municipalities.value = data.municipalities || data || []
  } catch {} finally {
    loading.value = false
  }
}

async function saveMunicipality() {
  formError.value = ''
  try {
    if (editingId.value) {
      await api.put('/api/v1/admin/municipalities/' + editingId.value, form.value)
    } else {
      await api.post('/api/v1/admin/municipalities', form.value)
    }
    showModal.value = false
    await loadMunicipalities()
  } catch (e) {
    formError.value = e.response?.data?.error || 'Failed to save municipality'
  }
}

async function deleteMunicipality() {
  if (!deleteTarget.value) return
  try {
    await api.delete('/api/v1/admin/municipalities/' + deleteTarget.value.id)
    showDelete.value = false
    deleteTarget.value = null
    await loadMunicipalities()
  } catch {}
}

onMounted(loadMunicipalities)
</script>
