<template>
  <div class="p-6">
    <!-- Header -->
    <div class="mb-6 pb-4 border-b border-navy-100">
      <h1 class="page-title">Settings</h1>
      <p class="page-subtitle">Manage your profile and account preferences</p>
    </div>

    <!-- Profile + Password Grid -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-5 mb-6">
      <!-- Profile Card (wider) -->
      <div class="lg:col-span-2 card">
        <div class="px-5 py-4 border-b border-navy-100 flex items-center gap-2">
          <i class="bi bi-person-circle text-navy-500"></i>
          <h2 class="font-serif font-bold text-navy-800">Profile</h2>
        </div>
        <div class="p-5 space-y-4">
          <div class="flex items-center gap-4 mb-4">
            <div class="w-16 h-16 bg-navy-100 rounded-full flex items-center justify-center text-2xl font-bold text-navy-600">
              {{ authStore.user?.name?.[0]?.toUpperCase() || '?' }}
            </div>
            <div>
              <h3 class="text-lg font-semibold text-navy-800">{{ authStore.user?.name }}</h3>
              <span class="badge" :class="roleClass(authStore.role)">{{ roleLabel(authStore.role) }}</span>
            </div>
          </div>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <p class="text-xs font-semibold uppercase tracking-wider text-navy-400 mb-1">Email</p>
              <p class="text-navy-700">{{ authStore.user?.email }}</p>
            </div>
            <div>
              <p class="text-xs font-semibold uppercase tracking-wider text-navy-400 mb-1">Role</p>
              <p class="text-navy-700">{{ roleLabel(authStore.role) }}</p>
            </div>
            <div v-if="authStore.governmentName">
              <p class="text-xs font-semibold uppercase tracking-wider text-navy-400 mb-1">Municipality</p>
              <p class="text-navy-700">{{ authStore.governmentName }}</p>
            </div>
            <div v-if="authStore.departmentName">
              <p class="text-xs font-semibold uppercase tracking-wider text-navy-400 mb-1">Department</p>
              <p class="text-navy-700">{{ authStore.departmentName }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Change Password Card -->
      <div class="card">
        <div class="px-5 py-4 border-b border-navy-100 flex items-center gap-2">
          <i class="bi bi-shield-lock text-navy-500"></i>
          <h2 class="font-serif font-bold text-navy-800">Change Password</h2>
        </div>
        <form @submit.prevent="changePassword" class="p-5 space-y-4">
          <div>
            <label class="form-label">Old Password</label>
            <input v-model="pwForm.old_password" type="password" required class="form-input" placeholder="Current password" />
          </div>
          <div>
            <label class="form-label">New Password</label>
            <input v-model="pwForm.new_password" type="password" required class="form-input" placeholder="Minimum 8 characters" />
          </div>
          <div>
            <label class="form-label">Confirm Password</label>
            <input v-model="pwForm.confirm_password" type="password" required class="form-input" placeholder="Repeat new password" />
          </div>
          <p v-if="pwError" class="text-sm text-red-600">{{ pwError }}</p>
          <p v-if="pwSuccess" class="text-sm text-green-600">{{ pwSuccess }}</p>
          <button type="submit" class="btn-primary w-full justify-center">
            <i class="bi bi-shield-lock mr-1"></i> Update Password
          </button>
        </form>
      </div>
    </div>

    <!-- Municipality Info (Manager Only) -->
    <div v-if="authStore.isManager" class="card">
      <div class="px-5 py-4 border-b border-navy-100 flex items-center gap-2">
        <i class="bi bi-building text-navy-500"></i>
        <h2 class="font-serif font-bold text-navy-800">Municipality Information</h2>
      </div>
      <form @submit.prevent="updateGovernment" class="p-5">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="form-label">Name</label>
            <input v-model="govForm.name" required class="form-input" placeholder="Municipality name" />
          </div>
          <div>
            <label class="form-label">Jurisdiction</label>
            <input v-model="govForm.jurisdiction" class="form-input" placeholder="e.g. Chennai" />
          </div>
          <div>
            <label class="form-label">State</label>
            <input v-model="govForm.state" class="form-input" placeholder="e.g. Tamil Nadu" />
          </div>
          <div>
            <label class="form-label">Contact Email</label>
            <input v-model="govForm.email" type="email" class="form-input" placeholder="contact@municipality.gov.in" />
          </div>
          <div>
            <label class="form-label">Phone</label>
            <input v-model="govForm.phone" class="form-input" placeholder="+91 44 12345678" />
          </div>
        </div>
        <div class="mt-4">
          <label class="form-label">Logo</label>
          <FileUpload v-model="govForm.logo" />
        </div>
        <p v-if="govError" class="text-sm text-red-600 mt-3">{{ govError }}</p>
        <p v-if="govSuccess" class="text-sm text-green-600 mt-3">{{ govSuccess }}</p>
        <div class="flex justify-end mt-5">
          <button type="submit" class="btn-primary">
            <i class="bi bi-gear mr-1"></i> Update Municipality
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import api from '../api'
import FileUpload from '../components/FileUpload.vue'

const authStore = useAuthStore()

const pwForm = ref({ old_password: '', new_password: '', confirm_password: '' })
const pwError = ref('')
const pwSuccess = ref('')

const govForm = ref({ name: '', jurisdiction: '', state: '', email: '', phone: '', logo: '' })
const govError = ref('')
const govSuccess = ref('')

function roleClass(r) {
  const map = { super_admin: 'badge-danger', manager: 'badge-info', dept_manager: 'badge-success' }
  return map[r] || 'badge-info'
}

function roleLabel(r) {
  const map = { super_admin: 'Super Admin', manager: 'Manager', dept_manager: 'Dept Manager' }
  return map[r] || r
}

async function changePassword() {
  pwError.value = ''
  pwSuccess.value = ''
  if (pwForm.value.new_password !== pwForm.value.confirm_password) {
    pwError.value = 'Passwords do not match'
    return
  }
  if (pwForm.value.new_password.length < 8) {
    pwError.value = 'Password must be at least 8 characters'
    return
  }
  try {
    await api.post('/api/v1/admin/change-password', {
      old_password: pwForm.value.old_password,
      new_password: pwForm.value.new_password
    })
    pwSuccess.value = 'Password updated successfully'
    pwForm.value = { old_password: '', new_password: '', confirm_password: '' }
  } catch (e) {
    pwError.value = e.response?.data?.error || 'Failed to change password'
  }
}

async function updateGovernment() {
  govError.value = ''
  govSuccess.value = ''
  try {
    await api.put('/api/v1/admin/government', govForm.value)
    govSuccess.value = 'Municipality info updated'
    await authStore.fetchMe()
  } catch (e) {
    govError.value = e.response?.data?.error || 'Failed to update municipality info'
  }
}

async function loadGovernmentInfo() {
  if (!authStore.isManager) return
  try {
    const { data } = await api.get('/api/v1/admin/me')
    govForm.value = {
      name: data.government_name || '',
      jurisdiction: data.jurisdiction || '',
      state: data.state || '',
      email: data.government_email || '',
      phone: data.government_phone || '',
      logo: data.government_logo || ''
    }
  } catch {}
}

onMounted(loadGovernmentInfo)
</script>
