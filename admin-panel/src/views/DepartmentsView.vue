<template>
  <div class="max-w-7xl mx-auto px-4 py-8">
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-2xl font-bold text-gray-800">Departments</h2>
      <button @click="showForm = !showForm"
        class="bg-civic-600 text-white px-4 py-2 rounded-lg text-sm hover:bg-civic-700 transition">
        {{ showForm ? 'Cancel' : '+ Add Department' }}
      </button>
    </div>

    <!-- Add/Edit Form -->
    <div v-if="showForm" class="bg-white rounded-xl shadow-sm p-6 mb-6">
      <form @submit.prevent="saveDept" class="space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Department Name</label>
            <input v-model="form.name" required class="w-full border rounded-lg px-3 py-2" placeholder="Public Works" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Government ID</label>
            <input v-model.number="form.government_id" type="number" required class="w-full border rounded-lg px-3 py-2" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Email</label>
            <input v-model="form.email" type="email" class="w-full border rounded-lg px-3 py-2" placeholder="dept@gov.in" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Phone</label>
            <input v-model="form.phone" class="w-full border rounded-lg px-3 py-2" placeholder="+91-XXXXXXXXXX" />
          </div>
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Services (comma-separated)</label>
          <input v-model="servicesText" class="w-full border rounded-lg px-3 py-2" placeholder="Road repair, Water supply, Drainage" />
        </div>
        <div class="flex gap-2">
          <button type="submit" class="bg-green-600 text-white px-6 py-2 rounded-lg hover:bg-green-700 transition">
            {{ editing ? 'Update' : 'Create' }}
          </button>
          <button type="button" @click="resetForm" class="bg-gray-200 px-4 py-2 rounded-lg">Cancel</button>
        </div>
      </form>
    </div>

    <!-- Departments Grid -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div v-for="d in departments" :key="d.ID"
        class="bg-white rounded-xl shadow-sm p-5 border-l-4 border-civic-500 hover:shadow-md transition">
        <div class="flex justify-between items-start">
          <h3 class="font-semibold text-gray-800">{{ d.name }}</h3>
          <div class="flex gap-2">
            <button @click="editDept(d)" class="text-civic-600 hover:underline text-xs">Edit</button>
            <button @click="deleteDept(d.ID)" class="text-red-500 hover:underline text-xs">Delete</button>
          </div>
        </div>
        <p v-if="d.email" class="text-sm text-gray-500 mt-1">{{ d.email }}</p>
        <p v-if="d.phone" class="text-sm text-gray-500">{{ d.phone }}</p>
        <div v-if="d.services?.length" class="flex flex-wrap gap-1 mt-2">
          <span v-for="s in d.services" :key="s" class="text-xs bg-blue-100 text-blue-700 px-2 py-0.5 rounded-full">{{ s }}</span>
        </div>
        <p class="text-xs text-gray-400 mt-2">Gov ID: {{ d.government_id }} · Created {{ new Date(d.CreatedAt).toLocaleDateString() }}</p>
      </div>
      <div v-if="!departments.length" class="col-span-full text-center py-8 text-gray-400">
        No departments yet. Create one above.
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../api'

const departments = ref([])
const showForm = ref(false)
const editing = ref(null)
const servicesText = ref('')
const form = ref({ name: '', government_id: 0, email: '', phone: '' })

function resetForm() {
  form.value = { name: '', government_id: 0, email: '', phone: '' }
  servicesText.value = ''
  editing.value = null
  showForm.value = false
}

function editDept(d) {
  editing.value = d.ID
  form.value = { name: d.name, government_id: d.government_id, email: d.email || '', phone: d.phone || '' }
  servicesText.value = (d.services || []).join(', ')
  showForm.value = true
}

async function saveDept() {
  const payload = { ...form.value, services: servicesText.value.split(',').map(s => s.trim()).filter(Boolean) }
  try {
    if (editing.value) {
      await api.put(`/api/v1/admin/departments/${editing.value}`, payload)
    } else {
      await api.post('/api/v1/admin/departments', payload)
    }
    resetForm()
    await loadDepts()
  } catch {
    // handle error
  }
}

async function deleteDept(id) {
  if (!confirm('Delete this department?')) return
  try {
    await api.delete(`/api/v1/admin/departments/${id}`)
    await loadDepts()
  } catch {
    // handle error
  }
}

async function loadDepts() {
  try {
    const { data } = await api.get('/api/v1/admin/departments')
    departments.value = data
  } catch {
    departments.value = []
  }
}

onMounted(loadDepts)
</script>
