<template>
  <div class="max-w-7xl mx-auto px-4 py-8">
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-2xl font-bold text-gray-800">Registered Citizens</h2>
      <span class="text-sm text-gray-500">{{ users.length }} users</span>
    </div>

    <div class="bg-white rounded-xl shadow-sm overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-gray-50">
          <tr>
            <th class="text-left p-3">ID</th>
            <th class="text-left p-3">Name</th>
            <th class="text-left p-3">Aadhar No</th>
            <th class="text-left p-3">Email</th>
            <th class="text-left p-3">Phone</th>
            <th class="text-left p-3">Location</th>
            <th class="text-left p-3">Joined</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.ID" class="border-t hover:bg-gray-50">
            <td class="p-3 font-mono">#{{ u.ID }}</td>
            <td class="p-3 font-medium">{{ u.full_name }}</td>
            <td class="p-3 font-mono text-gray-600">{{ maskAadhar(u.aadhar_no) }}</td>
            <td class="p-3 text-gray-600">{{ u.email || '—' }}</td>
            <td class="p-3 text-gray-600">{{ u.phone || '—' }}</td>
            <td class="p-3 text-gray-500 text-xs">{{ u.location || '—' }}</td>
            <td class="p-3 text-gray-500">{{ new Date(u.CreatedAt).toLocaleDateString() }}</td>
          </tr>
          <tr v-if="!users.length">
            <td colspan="7" class="text-center py-8 text-gray-400">No users found</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../api'

const users = ref([])

function maskAadhar(aadhar) {
  if (!aadhar || aadhar.length < 4) return '****'
  return '****-****-' + aadhar.slice(-4)
}

onMounted(async () => {
  try {
    const { data } = await api.get('/api/v1/admin/users')
    users.value = data
  } catch {
    // empty
  }
})
</script>
