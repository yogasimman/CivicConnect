import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/LoginView.vue'),
    meta: { guest: true },
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: () => import('../views/DashboardView.vue'),
    meta: { requiresAuth: true },
  },
  {
    path: '/municipalities',
    name: 'Municipalities',
    component: () => import('../views/MunicipalitiesView.vue'),
    meta: { requiresAuth: true, roles: ['super_admin'] },
  },
  {
    path: '/complaints',
    name: 'Complaints',
    component: () => import('../views/ComplaintsView.vue'),
    meta: { requiresAuth: true, roles: ['manager', 'dept_manager'] },
  },
  {
    path: '/articles',
    name: 'Articles',
    component: () => import('../views/ArticlesView.vue'),
    meta: { requiresAuth: true, roles: ['manager', 'dept_manager'] },
  },
  {
    path: '/articles/new',
    name: 'NewArticle',
    component: () => import('../views/ArticleEditorView.vue'),
    meta: { requiresAuth: true, roles: ['manager', 'dept_manager'] },
  },
  {
    path: '/articles/:id/edit',
    name: 'EditArticle',
    component: () => import('../views/ArticleEditorView.vue'),
    meta: { requiresAuth: true, roles: ['manager', 'dept_manager'] },
  },
  {
    path: '/community-posts',
    name: 'CommunityPosts',
    component: () => import('../views/CommunityPostsView.vue'),
    meta: { requiresAuth: true, roles: ['manager', 'dept_manager'] },
  },
  {
    path: '/departments',
    name: 'Departments',
    component: () => import('../views/DepartmentsView.vue'),
    meta: { requiresAuth: true, roles: ['manager', 'dept_manager'] },
  },
  {
    path: '/admins',
    name: 'Admins',
    component: () => import('../views/AdminsView.vue'),
    meta: { requiresAuth: true, roles: ['super_admin', 'manager'] },
  },
  {
    path: '/users',
    name: 'Users',
    component: () => import('../views/UsersView.vue'),
    meta: { requiresAuth: true, roles: ['manager', 'dept_manager'] },
  },
  {
    path: '/settings',
    name: 'Settings',
    component: () => import('../views/SettingsView.vue'),
    meta: { requiresAuth: true },
  },
  { path: '/', redirect: '/dashboard' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to, _from, next) => {
  const auth = useAuthStore()
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return next('/login')
  }
  if (to.meta.guest && auth.isAuthenticated) {
    return next('/dashboard')
  }
  if (to.meta.roles && !to.meta.roles.includes(auth.user?.role)) {
    return next('/dashboard')
  }
  next()
})

export default router
