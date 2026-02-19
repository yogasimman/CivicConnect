<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold text-gray-900 mb-6">Settings</h1>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Profile Info -->
      <div class="bg-white rounded-xl border border-gray-200 p-6">
        <h2 class="text-lg font-semibold text-gray-900 mb-4">Profile</h2>
        <div class="space-y-3">
          <div><label class="text-sm text-gray-500">Name</label><p class="text-gray-900 font-medium">{{ authStore.user?.full_name }}</p></div>
          <div><label class="text-sm text-gray-500">Email</label><p class="text-gray-900">{{ authStore.user?.email }}</p></div>
          <div><label class="text-sm text-gray-500">Role</label>
            <p><span class="text-xs px-2 py-0.5 rounded-full font-medium"
              :class="{ 'bg-red-100 text-red-700': authStore.isSuperAdmin, 'bg-purple-100 text-purple-700': authStore.isManager, 'bg-blue-100 text-blue-700': authStore.isDeptManager }">
              {{ authStore.isSuperAdmin ? 'Super Admin' : authStore.isManager ? 'Manager' : 'Dept Manager' }}
            </span></p>
          </div>
          <div><label class="text-sm text-gray-500">Government ID</label><p class="text-gray-900 font-mono">{{ authStore.user?.government_id || '—' }}</p></div>
          <div v-if="authStore.user?.department_id"><label class="text-sm text-gray-500">Department ID</label><p class="text-gray-900 font-mono">{{ authStore.user.department_id }}</p></div>
        </div>
      </div>

      <!-- Change Password -->
      <div class="bg-white rounded-xl border border-gray-200 p-6">
        <h2 class="text-lg font-semibold text-gray-900 mb-4">Change Password</h2>
        <form @submit.prevent="changePassword" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Current Password</label>
            <input v-model="pwForm.current" type="password" required class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-civic-500 focus:border-civic-500" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">New Password</label>
            <input v-model="pwForm.newPass" type="password" required minlength="6" class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-civic-500 focus:border-civic-500" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Confirm New Password</label>
            <input v-model="pwForm.confirm" type="password" required class="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-civic-500 focus:border-civic-500" />
          </div>
          <p v-if="pwError" class="text-sm text-red-600">{{ pwError }}</p>
          <p v-if="pwSuccess" class="text-sm text-green-600">{{ pwSuccess }}</p>
          <button type="submit" class="bg-civic-600 text-white px-6 py-2 rounded-lg text-sm font-medium hover:bg-civic-700 transition">
            Update Password
          </button>
        </form>
      </div>

      <!-- Government Info (super_admin / manager only) -->
      <div v-if="authStore.isSuperAdmin || authStore.isManager" class="bg-white rounded-xl border border-gray-200 p-6 lg:col-span-2">
        <h2 class="text-lg font-semibold text-gray-900 mb-4">Government Info</h2>
        <div v-if="govLoading" class="text-gray-400 text-sm">Loading...</div>
        <div v-else-if="govInfo" class="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div><label class="text-sm text-gray-500">Name</label><p class="text-gray-900 font-medium">{{ govInfo.name }}</p></div>
          <div><label class="text-sm text-gray-500">Jurisdiction</label><p class="text-gray-900">{{ govInfo.jurisdiction }}</p></div>
          <div><label class="text-sm text-gray-500">State</label><p class="text-gray-900">{{ govInfo.state || '—' }}</p></div>
          <div><label class="text-sm text-gray-500">Contact Email</label><p class="text-gray-900">{{ govInfo.contact_email || '—' }}</p></div>
          <div><label class="text-sm text-gray-500">Contact Phone</label><p class="text-gray-900">{{ govInfo.contact_phone || '—' }}</p></div>
          <div><label class="text-sm text-gray-500">Followers</label><p class="text-gray-900 font-mono">{{ govInfo.follower_count || 0 }}</p></div>
        </div>
        <p v-else class="text-gray-400 text-sm">No government info available</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import api from '../api'

const authStore = useAuthStore()
const pwForm = ref({ current: '', newPass: '', confirm: '' })
const pwError = ref('')
const pwSuccess = ref('')
const govInfo = ref(null)
const govLoading = ref(false)

async function changePassword() {
  pwError.value = ''
  pwSuccess.value = ''
  if (pwForm.value.newPass !== pwForm.value.confirm) {
    pwError.value = 'Passwords do not match'
    return
  }
  try {
    await api.put('/api/v1/admin/me/password', { current_password: pwForm.value.current, new_password: pwForm.value.newPass })
    pwSuccess.value = 'Password updated successfully'
    pwForm.value = { current: '', newPass: '', confirm: '' }
  } catch (e) {
    pwError.value = e.response?.data?.error || 'Failed to update password'
  }
}

onMounted(async () => {
  if ((authStore.isSuperAdmin || authStore.isManager) && authStore.user?.government_id) {
    govLoading.value = true
    try {
      const { data } = await api.get(`/api/v1/admin/governments/${authStore.user.government_id}`)
      govInfo.value = data
    } catch { govInfo.value = null }
    finally { govLoading.value = false }
  }
})
</script>
