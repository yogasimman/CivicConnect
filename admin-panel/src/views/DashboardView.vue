<template>
  <div class="p-6">
    <!-- Header -->
    <div class="mb-6 pb-4 border-b border-navy-100">
      <h1 class="page-title">Dashboard</h1>
      <p class="page-subtitle">Overview of {{ authStore.governmentName || 'your administration' }}</p>
    </div>

    <!-- Stats Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-5 mb-8">
      <div v-for="stat in stats" :key="stat.label" class="card p-5 hover:shadow-md transition">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-xs font-semibold uppercase tracking-wider text-navy-400">{{ stat.label }}</p>
            <p class="text-3xl font-bold text-navy-800 mt-1 font-sans">{{ stat.value }}</p>
          </div>
          <div class="w-12 h-12 rounded-lg flex items-center justify-center" :class="stat.bg">
            <i :class="stat.icon" class="text-xl"></i>
          </div>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-5">
      <!-- Recent Complaints -->
      <div class="lg:col-span-2 card">
        <div class="px-5 py-4 border-b border-navy-100 flex items-center justify-between">
          <h2 class="font-serif font-bold text-navy-800">Recent Complaints</h2>
          <router-link v-if="authStore.canManageComplaints" to="/complaints" class="text-sm text-navy-500 hover:text-navy-700 font-medium">View All <i class="bi bi-arrow-right"></i></router-link>
        </div>
        <div v-if="recentComplaints.length === 0" class="p-8 text-center text-navy-300">
          <i class="bi bi-inbox text-3xl mb-2 block"></i>
          <p class="text-sm">No complaints yet</p>
        </div>
        <div v-else class="divide-y divide-navy-100">
          <div v-for="c in recentComplaints" :key="c.id || c.ID" class="px-5 py-3 flex items-center justify-between hover:bg-navy-50 transition">
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium text-navy-700 truncate">{{ c.description?.substring(0, 60) || 'No description' }}...</p>
              <p class="text-xs text-navy-400 mt-0.5">{{ c.category }} &middot; {{ formatDate(c.created_at || c.CreatedAt) }}</p>
            </div>
            <span class="badge ml-3" :class="statusClass(c.status || c.Status)">{{ c.status || c.Status }}</span>
          </div>
        </div>
      </div>

      <!-- Quick Actions -->
      <div class="card">
        <div class="px-5 py-4 border-b border-navy-100">
          <h2 class="font-serif font-bold text-navy-800">Quick Actions</h2>
        </div>
        <div class="p-4 space-y-2">
          <router-link v-for="action in quickActions" :key="action.to" :to="action.to" class="flex items-center px-4 py-3 rounded-md hover:bg-navy-50 transition group">
            <div class="w-9 h-9 rounded-lg flex items-center justify-center bg-navy-100 group-hover:bg-navy-200 transition mr-3">
              <i :class="action.icon" class="text-navy-600"></i>
            </div>
            <div>
              <p class="text-sm font-medium text-navy-700">{{ action.label }}</p>
              <p class="text-xs text-navy-400">{{ action.desc }}</p>
            </div>
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
const dashboard = ref({ municipalities: 0, departments: 0, managers: 0, users: 0, pending: 0, resolved: 0, in_progress: 0 })
const recentComplaints = ref([])

const stats = computed(() => {
  const d = dashboard.value
  if (authStore.isSuperAdmin) {
    return [
      { label: 'Municipalities', value: d.municipalities || 0, icon: 'bi bi-building text-navy-600', bg: 'bg-navy-100' },
      { label: 'Managers', value: d.managers || 0, icon: 'bi bi-shield-check text-green-600', bg: 'bg-green-100' },
      { label: 'Departments', value: d.departments || 0, icon: 'bi bi-diagram-3 text-blue-600', bg: 'bg-blue-100' },
      { label: 'Citizens', value: d.users || 0, icon: 'bi bi-people text-amber-600', bg: 'bg-amber-100' },
    ]
  }
  return [
    { label: 'Departments', value: d.departments || 0, icon: 'bi bi-diagram-3 text-navy-600', bg: 'bg-navy-100' },
    { label: 'Pending', value: d.pending || 0, icon: 'bi bi-clock text-amber-600', bg: 'bg-amber-100' },
    { label: 'In Progress', value: d.in_progress || 0, icon: 'bi bi-arrow-repeat text-blue-600', bg: 'bg-blue-100' },
    { label: 'Resolved', value: d.resolved || 0, icon: 'bi bi-check-circle text-green-600', bg: 'bg-green-100' },
  ]
})

const quickActions = computed(() => {
  const actions = []
  if (authStore.canManageComplaints) actions.push({ to: '/complaints', label: 'Manage Complaints', desc: 'Review and respond', icon: 'bi bi-exclamation-triangle' })
  if (authStore.canManageArticles) actions.push({ to: '/articles', label: 'Manage Articles', desc: 'Create and edit', icon: 'bi bi-newspaper' })
  if (authStore.canManageDepartments) actions.push({ to: '/departments', label: 'Departments', desc: 'View departments', icon: 'bi bi-diagram-3' })
  if (authStore.canManageAdmins) actions.push({ to: '/admins', label: authStore.isSuperAdmin ? 'Managers' : 'Staff', desc: 'Manage personnel', icon: 'bi bi-shield-check' })
  if (authStore.canManageMunicipalities) actions.push({ to: '/municipalities', label: 'Municipalities', desc: 'Manage corporations', icon: 'bi bi-building' })
  actions.push({ to: '/settings', label: 'Settings', desc: 'Profile & preferences', icon: 'bi bi-gear' })
  return actions
})

function formatDate(d) {
  if (!d) return ''
  return new Date(d).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' })
}

function statusClass(s) {
  const map = { pending: 'badge-warning', in_progress: 'badge-info', resolved: 'badge-success', rejected: 'badge-danger' }
  return map[s] || 'badge-info'
}

onMounted(async () => {
  try {
    const { data } = await api.get('/api/v1/admin/dashboard')
    dashboard.value = data
    recentComplaints.value = data.recent_complaints || []
  } catch {}
})
</script>
