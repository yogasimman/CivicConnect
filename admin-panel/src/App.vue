<template>
  <div class="min-h-screen bg-navy-50" v-if="!authStore.isAuthenticated">
    <router-view />
  </div>
  <div class="min-h-screen flex bg-navy-50" v-else>
    <!-- Sidebar -->
    <aside class="w-64 bg-navy-800 text-white flex flex-col fixed inset-y-0 z-30">
      <!-- Brand -->
      <div class="h-16 flex items-center px-5 border-b border-navy-700/50">
        <div class="w-9 h-9 bg-gold-500/20 rounded-lg flex items-center justify-center mr-3">
          <i class="bi bi-bank2 text-gold-400 text-lg"></i>
        </div>
        <div>
          <h1 class="text-base font-bold tracking-tight font-serif">CivicConnect</h1>
          <p class="text-navy-300 text-xs">{{ authStore.governmentName || 'Government Portal' }}</p>
        </div>
      </div>

      <!-- Navigation -->
      <nav class="flex-1 py-4 px-3 space-y-0.5 overflow-y-auto">
        <router-link to="/dashboard" class="nav-link" active-class="nav-link-active">
          <i class="bi bi-speedometer2 nav-icon"></i>
          Dashboard
        </router-link>

        <!-- SuperAdmin: Municipalities -->
        <template v-if="authStore.canManageMunicipalities">
          <p class="nav-section-header">Administration</p>
          <router-link to="/municipalities" class="nav-link" active-class="nav-link-active">
            <i class="bi bi-building nav-icon"></i>
            Municipalities
          </router-link>
        </template>

        <!-- Admin management -->
        <template v-if="authStore.canManageAdmins">
          <router-link to="/admins" class="nav-link" active-class="nav-link-active">
            <i class="bi bi-shield-check nav-icon"></i>
            {{ authStore.isSuperAdmin ? 'Managers' : 'Dept. Managers' }}
          </router-link>
        </template>

        <!-- Manager / DeptManager sections -->
        <template v-if="authStore.canManageDepartments">
          <p class="nav-section-header">Municipality</p>
          <router-link to="/departments" class="nav-link" active-class="nav-link-active">
            <i class="bi bi-diagram-3 nav-icon"></i>
            Departments
          </router-link>
        </template>

        <template v-if="authStore.canManageComplaints">
          <p v-if="!authStore.canManageDepartments" class="nav-section-header">Department</p>
          <p v-else class="nav-section-header">Services</p>
          <router-link to="/complaints" class="nav-link" active-class="nav-link-active">
            <i class="bi bi-exclamation-triangle nav-icon"></i>
            Complaints
          </router-link>
          <router-link to="/articles" class="nav-link" active-class="nav-link-active">
            <i class="bi bi-newspaper nav-icon"></i>
            Articles
          </router-link>
          <router-link to="/community-posts" class="nav-link" active-class="nav-link-active">
            <i class="bi bi-chat-square-text nav-icon"></i>
            Community Posts
          </router-link>
          <router-link to="/users" class="nav-link" active-class="nav-link-active">
            <i class="bi bi-people nav-icon"></i>
            Citizens
          </router-link>
        </template>

        <router-link to="/settings" class="nav-link" active-class="nav-link-active">
          <i class="bi bi-gear nav-icon"></i>
          Settings
        </router-link>
      </nav>

      <!-- User Info -->
      <div class="p-4 border-t border-navy-700/50">
        <div class="flex items-center">
          <div class="w-9 h-9 bg-navy-600 rounded-full flex items-center justify-center text-sm font-bold">
            {{ authStore.user?.name?.[0]?.toUpperCase() || '?' }}
          </div>
          <div class="ml-3 flex-1 min-w-0">
            <p class="text-sm font-medium truncate">{{ authStore.user?.name }}</p>
            <p class="text-xs text-navy-300">{{ roleLabel }}</p>
          </div>
          <button @click="logout" class="text-navy-400 hover:text-white transition" title="Logout">
            <i class="bi bi-box-arrow-right text-lg"></i>
          </button>
        </div>
      </div>
    </aside>

    <!-- Main Content -->
    <main class="flex-1 ml-64 min-h-screen">
      <router-view />
    </main>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useAuthStore } from './stores/auth'
import { useRouter } from 'vue-router'

const authStore = useAuthStore()
const router = useRouter()

const roleLabel = computed(() => {
  const labels = { super_admin: 'Super Admin', manager: 'Manager', dept_manager: 'Dept Manager' }
  return labels[authStore.role] || authStore.role
})

function logout() {
  authStore.logout()
  router.push('/login')
}
</script>

<style>
.nav-link {
  @apply flex items-center px-3 py-2.5 text-sm font-medium text-navy-200 rounded-md hover:bg-navy-700 hover:text-white transition-colors;
}
.nav-link-active {
  @apply bg-navy-900/50 text-white border-l-[3px] border-gold-400;
}
.nav-icon {
  @apply text-base mr-3 flex-shrink-0 w-5 text-center;
}
.nav-section-header {
  @apply text-[10px] font-bold text-gold-400/80 uppercase tracking-widest px-3 pt-5 pb-1;
}
</style>
