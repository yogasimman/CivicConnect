<template>
  <div class="p-6">
    <!-- Header -->
    <div class="mb-6 pb-4 border-b border-navy-100">
      <h1 class="page-title">Citizens</h1>
      <p class="page-subtitle">{{ total }} registered citizens</p>
    </div>

    <!-- Search & Sort -->
    <div class="card p-4 mb-6">
      <div class="flex flex-wrap gap-3 items-center">
        <div class="relative flex-1 min-w-[250px]">
          <i class="bi bi-search absolute left-3 top-2.5 text-navy-300"></i>
          <input v-model="searchQuery" @input="debouncedSearch" type="text"
            placeholder="Search by name, email, or last 4 Aadhar digits..."
            class="form-input pl-9" />
        </div>
        <select v-model="sortBy" @change="loadUsers" class="form-input w-auto">
          <option value="created_at">Newest First</option>
          <option value="name">Name A-Z</option>
        </select>
      </div>
    </div>

    <!-- Users Table -->
    <div class="card overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="table-header">
            <th class="text-left px-4 py-3">Name</th>
            <th class="text-left px-4 py-3">Aadhar No</th>
            <th class="text-left px-4 py-3">Email</th>
            <th class="text-left px-4 py-3">Location</th>
            <th class="text-left px-4 py-3">Joined</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.id" class="table-row">
            <td class="px-4 py-3">
              <div class="flex items-center gap-3">
                <div class="w-8 h-8 bg-navy-100 rounded-full flex items-center justify-center text-xs font-bold text-navy-600">
                  {{ u.name?.[0]?.toUpperCase() || '?' }}
                </div>
                <span class="font-medium text-navy-700">{{ u.name }}</span>
              </div>
            </td>
            <td class="px-4 py-3 font-mono text-navy-500 text-xs">{{ maskAadhar(u.aadhar_no) }}</td>
            <td class="px-4 py-3 text-navy-500">{{ u.email || '—' }}</td>
            <td class="px-4 py-3 text-navy-400 text-xs">{{ u.location || '—' }}</td>
            <td class="px-4 py-3 text-navy-400 text-xs">{{ formatDate(u.created_at) }}</td>
          </tr>
          <tr v-if="!users.length">
            <td colspan="5" class="text-center py-12 text-navy-300">
              <i class="bi bi-people text-3xl mb-2 block"></i>
              <p class="text-sm">No citizens found</p>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Pagination -->
    <div v-if="totalPages > 1" class="flex items-center justify-between mt-4">
      <p class="text-sm text-navy-400">Page {{ page }} of {{ totalPages }} &middot; {{ total }} total</p>
      <div class="flex gap-1">
        <button @click="goPage(page - 1)" :disabled="page <= 1"
          class="px-3 py-1.5 text-sm border border-navy-200 rounded-md hover:bg-navy-50 disabled:opacity-40 transition flex items-center gap-1">
          <i class="bi bi-chevron-left text-xs"></i> Previous
        </button>
        <button v-for="p in visiblePages" :key="p" @click="goPage(p)"
          :class="p === page ? 'bg-navy-700 text-white border-navy-700' : 'border-navy-200 hover:bg-navy-50 text-navy-600'"
          class="px-3 py-1.5 text-sm border rounded-md transition">{{ p }}</button>
        <button @click="goPage(page + 1)" :disabled="page >= totalPages"
          class="px-3 py-1.5 text-sm border border-navy-200 rounded-md hover:bg-navy-50 disabled:opacity-40 transition flex items-center gap-1">
          Next <i class="bi bi-chevron-right text-xs"></i>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import api from '../api'

const authStore = useAuthStore()
const users = ref([])
const page = ref(1)
const limit = ref(20)
const total = ref(0)
const totalPages = ref(0)
const searchQuery = ref('')
const sortBy = ref('created_at')
let searchTimeout = null

const visiblePages = computed(() => {
  const pages = []
  const start = Math.max(1, page.value - 2)
  const end = Math.min(totalPages.value, start + 4)
  for (let i = start; i <= end; i++) pages.push(i)
  return pages
})

function maskAadhar(aadhar) {
  if (!aadhar || aadhar.length < 4) return '****-****-****'
  return '****-****-' + aadhar.slice(-4)
}

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' })
}

function debouncedSearch() {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => { page.value = 1; loadUsers() }, 300)
}

function goPage(p) {
  if (p < 1 || p > totalPages.value) return
  page.value = p
  loadUsers()
}

async function loadUsers() {
  const order = sortBy.value === 'name' ? 'asc' : 'desc'
  try {
    const { data } = await api.get('/api/v1/admin/users', {
      params: { page: page.value, limit: limit.value, search: searchQuery.value, sort: sortBy.value, order }
    })
    users.value = data.users || data || []
    total.value = data.total || 0
    totalPages.value = data.total_pages || Math.ceil(total.value / limit.value) || 1
  } catch {}
}

onMounted(loadUsers)
</script>
