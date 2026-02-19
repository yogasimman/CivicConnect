<template>
  <div class="p-6">
    <div class="mb-6">
      <h1 class="text-2xl font-bold text-gray-900">Dashboard</h1>
      <p class="text-gray-500 text-sm mt-1">Welcome back, {{ authStore.user?.name }}.</p>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-5 mb-8">
      <div v-for="stat in stats" :key="stat.label" class="bg-white rounded-xl border border-gray-200 p-5 hover:shadow-md transition">
        <div class="flex items-center justify-between mb-3">
          <div :class="stat.iconBg" class="w-10 h-10 rounded-lg flex items-center justify-center">
            <span class="text-white text-lg">{{ stat.emoji }}</span>
          </div>
        </div>
        <p class="text-2xl font-bold text-gray-900">{{ stat.value }}</p>
        <p class="text-sm text-gray-500 mt-1">{{ stat.label }}</p>
      </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
      <div v-if="authStore.canManageComplaints" class="bg-white rounded-xl border border-gray-200 p-5">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-lg font-semibold text-gray-900">Recent Complaints</h2>
          <router-link to="/complaints" class="text-civic-600 text-sm font-medium hover:text-civic-700">View All &rarr;</router-link>
        </div>
        <div v-if="complaints.length === 0" class="text-gray-400 text-sm py-8 text-center">No complaints yet</div>
        <div v-else class="space-y-3">
          <div v-for="c in complaints.slice(0, 5)" :key="c.id" class="flex items-center justify-between py-2 border-b border-gray-100 last:border-0">
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium text-gray-900 truncate">{{ c.category }}</p>
              <p class="text-xs text-gray-500 truncate">{{ c.description?.substring(0, 60) }}...</p>
            </div>
            <span :class="statusClass(c.status)" class="text-xs px-2.5 py-1 rounded-full font-medium ml-3 whitespace-nowrap">{{ c.status }}</span>
          </div>
        </div>
      </div>

      <div class="bg-white rounded-xl border border-gray-200 p-5">
        <h2 class="text-lg font-semibold text-gray-900 mb-4">Quick Actions</h2>
        <div class="grid grid-cols-2 gap-3">
          <router-link v-if="authStore.canManageComplaints" to="/complaints" class="flex items-center p-3 bg-orange-50 rounded-lg hover:bg-orange-100 transition">
            <div class="w-8 h-8 bg-orange-500 rounded-lg flex items-center justify-center mr-3 text-white text-sm">!</div>
            <span class="text-sm font-medium text-gray-700">Complaints</span>
          </router-link>
          <router-link v-if="authStore.canManageArticles" to="/articles" class="flex items-center p-3 bg-blue-50 rounded-lg hover:bg-blue-100 transition">
            <div class="w-8 h-8 bg-blue-500 rounded-lg flex items-center justify-center mr-3 text-white text-sm">A</div>
            <span class="text-sm font-medium text-gray-700">Articles</span>
          </router-link>
          <router-link v-if="authStore.canManageDepartments" to="/departments" class="flex items-center p-3 bg-purple-50 rounded-lg hover:bg-purple-100 transition">
            <div class="w-8 h-8 bg-purple-500 rounded-lg flex items-center justify-center mr-3 text-white text-sm">D</div>
            <span class="text-sm font-medium text-gray-700">Departments</span>
          </router-link>
          <router-link to="/settings" class="flex items-center p-3 bg-gray-50 rounded-lg hover:bg-gray-100 transition">
            <div class="w-8 h-8 bg-gray-500 rounded-lg flex items-center justify-center mr-3 text-white text-sm">S</div>
            <span class="text-sm font-medium text-gray-700">Settings</span>
          </router-link>
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
const dashboard = ref({})
const complaints = ref([])

const stats = computed(() => {
  const d = dashboard.value
  const role = authStore.role
  if (role === 'super_admin') {
    return [
      { label: 'Governments', value: d.governments || 0, iconBg: 'bg-civic-600', emoji: '??' },
      { label: 'Departments', value: d.departments || 0, iconBg: 'bg-purple-500', emoji: '??' },
      { label: 'Admin Accounts', value: d.admins || 0, iconBg: 'bg-emerald-500', emoji: '??' },
      { label: 'Registered Citizens', value: d.users || 0, iconBg: 'bg-amber-500', emoji: '??' },
    ]
  }
  if (role === 'manager') {
    return [
      { label: 'Pending Complaints', value: d.pending_complaints || 0, iconBg: 'bg-orange-500', emoji: '?' },
      { label: 'In Progress', value: d.in_progress_complaints || 0, iconBg: 'bg-blue-500', emoji: '?' },
      { label: 'Resolved', value: d.resolved_complaints || 0, iconBg: 'bg-green-500', emoji: '?' },
      { label: 'Departments', value: d.departments || 0, iconBg: 'bg-purple-500', emoji: '??' },
    ]
  }
  return [
    { label: 'Pending Complaints', value: d.pending_complaints || 0, iconBg: 'bg-orange-500', emoji: '?' },
    { label: 'Resolved', value: d.resolved_complaints || 0, iconBg: 'bg-green-500', emoji: '?' },
  ]
})

function statusClass(status) {
  const c = { pending: 'bg-yellow-100 text-yellow-800', in_progress: 'bg-blue-100 text-blue-800', resolved: 'bg-green-100 text-green-800', rejected: 'bg-red-100 text-red-800' }
  return c[status] || 'bg-gray-100 text-gray-800'
}

onMounted(async () => {
  try {
    const { data } = await api.get('/api/v1/admin/dashboard')
    dashboard.value = data
  } catch {}
  if (authStore.canManageComplaints) {
    try {
      const govId = authStore.user?.government_id
      const deptId = authStore.user?.department_id
      let params = {}
      if (govId) params.government_id = govId
      if (authStore.isDeptManager && deptId) params.department_id = deptId
      const { data } = await api.get('/api/v1/complaints/', { params })
      complaints.value = Array.isArray(data) ? data.slice(0, 10) : []
    } catch {}
  }
})
</script>
