<template>
  <div class="min-h-screen flex items-center justify-center bg-navy-800 relative overflow-hidden">
    <div class="absolute inset-0 bg-gradient-to-br from-navy-900 via-navy-800 to-navy-700"></div>
    <div class="absolute inset-0 opacity-5" style="background-image: url('data:image/svg+xml,<svg xmlns=%22http://www.w3.org/2000/svg%22 width=%2260%22 height=%2260%22><path d=%22M30 0L60 30L30 60L0 30Z%22 fill=%22none%22 stroke=%22white%22 stroke-width=%220.5%22/></svg>');"></div>

    <div class="relative z-10 w-full max-w-md px-4">
      <!-- Header -->
      <div class="text-center mb-8">
        <div class="w-20 h-20 bg-gold-500/20 rounded-2xl flex items-center justify-center mx-auto mb-4 border border-gold-400/30">
          <i class="bi bi-bank2 text-gold-400 text-4xl"></i>
        </div>
        <h1 class="text-3xl font-serif font-bold text-white tracking-tight">CivicConnect</h1>
        <p class="text-navy-300 mt-1 text-sm">Government Administration Portal</p>
      </div>

      <!-- Tab Switcher -->
      <div class="flex bg-navy-700/50 rounded-lg p-1 mb-6">
        <button @click="mode = 'login'" :class="mode === 'login' ? 'bg-navy-600 text-white shadow' : 'text-navy-300 hover:text-white'" class="flex-1 py-2 text-sm font-medium rounded-md transition">Sign In</button>
        <button @click="mode = 'setup'" :class="mode === 'setup' ? 'bg-navy-600 text-white shadow' : 'text-navy-300 hover:text-white'" class="flex-1 py-2 text-sm font-medium rounded-md transition">Initial Setup</button>
      </div>

      <!-- Login Form -->
      <div v-if="mode === 'login'" class="card p-6">
        <h2 class="text-lg font-serif font-bold text-navy-800 mb-4">Administrator Sign In</h2>
        <form @submit.prevent="login" class="space-y-4">
          <div>
            <label class="form-label">Email Address</label>
            <div class="relative">
              <i class="bi bi-envelope absolute left-3 top-2.5 text-navy-300"></i>
              <input v-model="loginForm.email" type="email" required class="form-input pl-9" placeholder="admin@government.gov" />
            </div>
          </div>
          <div>
            <label class="form-label">Password</label>
            <div class="relative">
              <i class="bi bi-lock absolute left-3 top-2.5 text-navy-300"></i>
              <input v-model="loginForm.password" type="password" required class="form-input pl-9" placeholder="Enter password" />
            </div>
          </div>
          <p v-if="error" class="text-sm text-red-600 bg-red-50 p-2 rounded">{{ error }}</p>
          <button type="submit" :disabled="loading" class="btn-primary w-full justify-center py-2.5">
            <i v-if="loading" class="bi bi-arrow-repeat animate-spin"></i>
            {{ loading ? 'Signing in...' : 'Sign In' }}
          </button>
        </form>
      </div>

      <!-- Setup Form -->
      <div v-else class="card p-6">
        <h2 class="text-lg font-serif font-bold text-navy-800 mb-1">Initial System Setup</h2>
        <p class="text-sm text-navy-400 mb-4">Create the first Super Admin and municipality</p>
        <form @submit.prevent="setup" class="space-y-3">
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="form-label">Municipality Name</label>
              <input v-model="setupForm.government_name" required class="form-input" placeholder="Chennai Municipal Corp." />
            </div>
            <div>
              <label class="form-label">Jurisdiction</label>
              <input v-model="setupForm.jurisdiction" required class="form-input" placeholder="Chennai" />
            </div>
          </div>
          <div>
            <label class="form-label">Admin Full Name</label>
            <input v-model="setupForm.name" required class="form-input" placeholder="S. Radhakrishnan" />
          </div>
          <div>
            <label class="form-label">Email Address</label>
            <input v-model="setupForm.email" type="email" required class="form-input" placeholder="admin@government.gov" />
          </div>
          <div>
            <label class="form-label">Password</label>
            <input v-model="setupForm.password" type="password" required class="form-input" placeholder="Min. 8 characters" />
          </div>
          <p v-if="error" class="text-sm text-red-600 bg-red-50 p-2 rounded">{{ error }}</p>
          <p v-if="setupSuccess" class="text-sm text-green-700 bg-green-50 p-2 rounded">{{ setupSuccess }}</p>
          <button type="submit" :disabled="loading" class="btn-primary w-full justify-center py-2.5">
            <i v-if="loading" class="bi bi-arrow-repeat animate-spin"></i>
            {{ loading ? 'Setting up...' : 'Initialize System' }}
          </button>
        </form>
      </div>

      <p class="text-center text-navy-400 text-xs mt-6">&copy; 2026 CivicConnect Government Portal</p>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useRouter } from 'vue-router'

const authStore = useAuthStore()
const router = useRouter()
const mode = ref('login')
const loading = ref(false)
const error = ref('')
const setupSuccess = ref('')

const loginForm = ref({ email: '', password: '' })
const setupForm = ref({ government_name: '', jurisdiction: '', name: '', email: '', password: '' })

async function login() {
  error.value = ''
  loading.value = true
  try {
    await authStore.login(loginForm.value.email, loginForm.value.password)
    router.push('/dashboard')
  } catch (e) {
    error.value = e.response?.data?.error || 'Invalid credentials'
  } finally {
    loading.value = false
  }
}

async function setup() {
  error.value = ''
  setupSuccess.value = ''
  loading.value = true
  try {
    await authStore.seed(setupForm.value)
    setupSuccess.value = 'System initialized! Switch to Sign In to continue.'
    mode.value = 'login'
    loginForm.value.email = setupForm.value.email
  } catch (e) {
    error.value = e.response?.data?.error || 'Setup failed'
  } finally {
    loading.value = false
  }
}
</script>
