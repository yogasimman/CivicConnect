<template>
  <div class="min-h-screen flex items-center justify-center bg-gradient-to-br from-civic-700 via-civic-800 to-civic-900">
    <div class="bg-white rounded-2xl shadow-2xl p-8 w-full max-w-md">
      <!-- Brand -->
      <div class="text-center mb-8">
        <div class="w-14 h-14 bg-civic-600 rounded-xl flex items-center justify-center mx-auto mb-3">
          <svg class="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
          </svg>
        </div>
        <h1 class="text-2xl font-bold text-gray-900">CivicConnect</h1>
        <p class="text-gray-500 mt-1 text-sm">Government Administration Portal</p>
      </div>

      <!-- Mode Toggle -->
      <div class="flex bg-gray-100 rounded-lg p-1 mb-6" v-if="showSetupToggle">
        <button
          @click="mode = 'login'"
          :class="mode === 'login' ? 'bg-white shadow text-gray-900' : 'text-gray-500'"
          class="flex-1 py-2 text-sm font-medium rounded-md transition"
        >Sign In</button>
        <button
          @click="mode = 'setup'"
          :class="mode === 'setup' ? 'bg-white shadow text-gray-900' : 'text-gray-500'"
          class="flex-1 py-2 text-sm font-medium rounded-md transition"
        >Initial Setup</button>
      </div>

      <!-- Login Form -->
      <form v-if="mode === 'login'" @submit.prevent="handleLogin" class="space-y-4">
        <div v-if="error" class="bg-red-50 border border-red-200 text-red-600 text-sm p-3 rounded-lg flex items-start">
          <svg class="w-4 h-4 mr-2 mt-0.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" /></svg>
          {{ error }}
        </div>
        <div v-if="success" class="bg-green-50 border border-green-200 text-green-600 text-sm p-3 rounded-lg">
          {{ success }}
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Email</label>
          <input v-model="email" type="email" required autocomplete="email"
            class="w-full px-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-civic-500 focus:border-civic-500 outline-none transition" placeholder="admin@municipality.gov" />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Password</label>
          <input v-model="password" type="password" required autocomplete="current-password"
            class="w-full px-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-civic-500 focus:border-civic-500 outline-none transition" placeholder="••••••••" />
        </div>
        <button type="submit" :disabled="loading"
          class="w-full bg-civic-600 text-white py-2.5 rounded-lg font-semibold hover:bg-civic-700 transition disabled:opacity-50 disabled:cursor-not-allowed">
          <span v-if="loading" class="flex items-center justify-center">
            <svg class="animate-spin -ml-1 mr-2 h-4 w-4 text-white" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/></svg>
            Signing in...
          </span>
          <span v-else>Sign In</span>
        </button>
      </form>

      <!-- Setup Form (First Time) -->
      <form v-else @submit.prevent="handleSetup" class="space-y-4">
        <div v-if="error" class="bg-red-50 border border-red-200 text-red-600 text-sm p-3 rounded-lg">{{ error }}</div>
        <p class="text-sm text-gray-600 bg-blue-50 border border-blue-200 p-3 rounded-lg">
          Create the initial Super Admin account and municipal government.
        </p>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Municipal Corporation Name</label>
          <input v-model="setupForm.government_name" type="text" required
            class="w-full px-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-civic-500 focus:border-civic-500 outline-none" placeholder="Chennai Corporation" />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Jurisdiction</label>
          <input v-model="setupForm.jurisdiction" type="text"
            class="w-full px-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-civic-500 focus:border-civic-500 outline-none" placeholder="Greater Chennai Area" />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Admin Name</label>
          <input v-model="setupForm.name" type="text" required
            class="w-full px-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-civic-500 focus:border-civic-500 outline-none" placeholder="Full Name" />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Admin Email</label>
          <input v-model="setupForm.email" type="email" required
            class="w-full px-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-civic-500 focus:border-civic-500 outline-none" placeholder="admin@municipality.gov" />
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Password</label>
          <input v-model="setupForm.password" type="password" required minlength="6"
            class="w-full px-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-civic-500 focus:border-civic-500 outline-none" placeholder="Min. 6 characters" />
        </div>
        <button type="submit" :disabled="loading"
          class="w-full bg-emerald-600 text-white py-2.5 rounded-lg font-semibold hover:bg-emerald-700 transition disabled:opacity-50">
          {{ loading ? 'Creating...' : 'Create Super Admin & Government' }}
        </button>
      </form>

      <p class="text-center text-xs text-gray-400 mt-6">CivicConnect v2.0 — Urban Governance Platform</p>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const mode = ref('login')
const showSetupToggle = ref(true)
const email = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const success = ref('')

const setupForm = reactive({
  government_name: '',
  jurisdiction: '',
  name: '',
  email: '',
  password: '',
})

async function handleLogin() {
  error.value = ''
  loading.value = true
  try {
    await authStore.login(email.value, password.value)
    router.push('/dashboard')
  } catch (err) {
    error.value = err.response?.data?.error || 'Login failed. Please check your credentials.'
  } finally {
    loading.value = false
  }
}

async function handleSetup() {
  error.value = ''
  loading.value = true
  try {
    await authStore.seedSuperAdmin(setupForm)
    router.push('/dashboard')
  } catch (err) {
    error.value = err.response?.data?.error || 'Setup failed. A super admin may already exist.'
  } finally {
    loading.value = false
  }
}
</script>
