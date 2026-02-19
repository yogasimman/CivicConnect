<template>
  <div class="max-w-7xl mx-auto px-4 py-8">
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-2xl font-bold text-gray-800">Admin Accounts</h2>
      <button @click="showForm = !showForm"
        class="bg-civic-600 text-white px-4 py-2 rounded-lg text-sm hover:bg-civic-700 transition">
        {{ showForm ? 'Cancel' : '+ Add Admin' }}
      </button>
    </div>

    <!-- Add Form -->
    <div v-if="showForm" class="bg-white rounded-xl shadow-sm p-6 mb-6">
      <form @submit.prevent="createAdmin" class="space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Full Name</label>
            <input v-model="form.full_name" required class="w-full border rounded-lg px-3 py-2" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Email</label>
            <input v-model="form.email" type="email" required class="w-full border rounded-lg px-3 py-2" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Password</label>
            <input v-model="form.password" type="password" required class="w-full border rounded-lg px-3 py-2" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Role</label>
            <select v-model="form.role" class="w-full border rounded-lg px-3 py-2">
              <option v-if="authStore.isSuperAdmin" value="super_admin">Super Admin</option>
              <option v-if="authStore.isSuperAdmin" value="manager">Manager</option>
              <option value="dept_manager">Department Manager</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Government ID</label>
            <input v-model.number="form.government_id" type="number" required class="w-full border rounded-lg px-3 py-2" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Department ID (for DeptAdmin)</label>
            <input v-model.number="form.department_id" type="number" class="w-full border rounded-lg px-3 py-2" />
          </div>
        </div>
        <button type="submit" class="bg-green-600 text-white px-6 py-2 rounded-lg hover:bg-green-700 transition">
          Create Admin
        </button>
      </form>
    </div>

    <!-- Admins Table -->
    <div class="bg-white rounded-xl shadow-sm overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-gray-50">
          <tr>
            <th class="text-left p-3">ID</th>
            <th class="text-left p-3">Name</th>
            <th class="text-left p-3">Email</th>
            <th class="text-left p-3">Role</th>
            <th class="text-left p-3">Gov ID</th>
            <th class="text-left p-3">Dept ID</th>
            <th class="text-left p-3">Created</th>
            <th class="text-left p-3">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in admins" :key="a.ID" class="border-t hover:bg-gray-50">
            <td class="p-3 font-mono">#{{ a.ID }}</td>
            <td class="p-3 font-medium">{{ a.full_name }}</td>
            <td class="p-3 text-gray-600">{{ a.email }}</td>
            <td class="p-3">
              <span class="px-2 py-0.5 rounded-full text-xs"
                :class="{ 'bg-red-100 text-red-700': a.role === 'super_admin', 'bg-purple-100 text-purple-700': a.role === 'manager', 'bg-blue-100 text-blue-700': a.role === 'dept_manager' }">
                {{ a.role === 'super_admin' ? 'Super Admin' : a.role === 'manager' ? 'Manager' : 'Dept Manager' }}
              </span>
            </td>
            <td class="p-3 font-mono">{{ a.government_id }}</td>
            <td class="p-3 font-mono">{{ a.department_id || '—' }}</td>
            <td class="p-3 text-gray-500">{{ new Date(a.CreatedAt).toLocaleDateString() }}</td>
            <td class="p-3">
              <button @click="deleteAdmin(a.ID)" class="text-red-500 hover:underline text-xs">Delete</button>
            </td>
          </tr>
          <tr v-if="!admins.length">
            <td colspan="8" class="text-center py-8 text-gray-400">No admin accounts</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../api'
import { useAuthStore } from '../stores/auth'

const authStore = useAuthStore()

const admins = ref([])
const showForm = ref(false)
const form = ref({
  full_name: '', email: '', password: '',
  role: 'dept_manager', government_id: 0, department_id: 0,
})

async function createAdmin() {
  try {
    await api.post('/api/v1/admin/admins', form.value)
    form.value = { full_name: '', email: '', password: '', role: 'dept_manager', government_id: 0, department_id: 0 }
    showForm.value = false
    await loadAdmins()
  } catch {
    // handle error
  }
}

async function deleteAdmin(id) {
  if (!confirm('Delete this admin account?')) return
  try {
    await api.delete(`/api/v1/admin/admins/${id}`)
    await loadAdmins()
  } catch {
    // handle error
  }
}

async function loadAdmins() {
  try {
    const { data } = await api.get('/api/v1/admin/admins')
    admins.value = data
  } catch {
    admins.value = []
  }
}

onMounted(loadAdmins)
</script>
