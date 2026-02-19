import { defineStore } from 'pinia'
import api from '../api'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('civic_token') || null,
    user: JSON.parse(localStorage.getItem('civic_user') || 'null'),
  }),

  getters: {
    isAuthenticated: (state) => !!state.token,
    isSuperAdmin: (state) => state.user?.role === 'super_admin',
    isManager: (state) => state.user?.role === 'manager',
    isDeptManager: (state) => state.user?.role === 'dept_manager',
    role: (state) => state.user?.role || '',
    canManageComplaints: (state) => ['manager', 'dept_manager'].includes(state.user?.role),
    canManageArticles: (state) => ['manager', 'dept_manager'].includes(state.user?.role),
    canManageDepartments: (state) => ['super_admin', 'manager'].includes(state.user?.role),
    canManageAdmins: (state) => ['super_admin', 'manager'].includes(state.user?.role),
  },

  actions: {
    async login(email, password) {
      const { data } = await api.post('/api/v1/admin/login', { email, password })
      this.token = data.token
      this.user = data.admin
      localStorage.setItem('civic_token', data.token)
      localStorage.setItem('civic_user', JSON.stringify(data.admin))
      api.defaults.headers.common['Authorization'] = `Bearer ${data.token}`
    },

    async seedSuperAdmin(payload) {
      const { data } = await api.post('/api/v1/admin/seed', payload)
      this.token = data.token
      this.user = data.admin
      localStorage.setItem('civic_token', data.token)
      localStorage.setItem('civic_user', JSON.stringify(data.admin))
      api.defaults.headers.common['Authorization'] = `Bearer ${data.token}`
      return data
    },

    async fetchMe() {
      const { data } = await api.get('/api/v1/admin/me')
      this.user = data
      localStorage.setItem('civic_user', JSON.stringify(data))
    },

    logout() {
      this.token = null
      this.user = null
      localStorage.removeItem('civic_token')
      localStorage.removeItem('civic_user')
      delete api.defaults.headers.common['Authorization']
    },
  },
})
