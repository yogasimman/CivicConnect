<template>
  <div class="p-6 max-w-7xl mx-auto">
    <div class="flex items-center justify-between mb-8">
      <div>
        <h1 class="page-title">Articles &amp; Announcements</h1>
        <p class="page-subtitle">Manage government articles, news, and public announcements</p>
      </div>
      <router-link to="/articles/new" class="btn-primary flex items-center gap-2">
        <i class="bi bi-plus-lg"></i> New Article
      </router-link>
    </div>

    <!-- Filters -->
    <div class="card p-4 mb-6">
      <div class="flex flex-wrap gap-3 items-center">
        <div class="relative flex-1 max-w-sm">
          <i class="bi bi-search absolute left-3 top-1/2 -translate-y-1/2 text-navy-400"></i>
          <input v-model="search" type="text" placeholder="Search articles..." class="form-input pl-10" />
        </div>
        <select v-model="categoryFilter" class="form-input w-auto">
          <option value="">All Categories</option>
          <option v-for="c in articleCategories" :key="c.id" :value="c.name">{{ c.name }}</option>
        </select>
        <div class="flex items-center gap-1 text-sm text-navy-500">
          <i class="bi bi-collection"></i>
          <span>{{ filteredArticles.length }} article{{ filteredArticles.length !== 1 ? 's' : '' }}</span>
        </div>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="text-center py-16 text-navy-400">
      <i class="bi bi-arrow-clockwise animate-spin text-3xl block mb-2"></i>
      Loading articles...
    </div>

    <!-- Empty state -->
    <div v-else-if="filteredArticles.length === 0" class="card p-16 text-center">
      <i class="bi bi-newspaper text-5xl text-navy-300 mb-3 block"></i>
      <h3 class="text-lg font-semibold text-navy-700">No articles found</h3>
      <p class="text-navy-400 mt-1">Create your first article to get started</p>
      <router-link to="/articles/new" class="btn-primary inline-flex items-center gap-2 mt-4">
        <i class="bi bi-plus-lg"></i> Create Article
      </router-link>
    </div>

    <!-- Articles Grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <div v-for="a in filteredArticles" :key="a.article_id" class="card overflow-hidden hover:shadow-lg transition group">
        <div class="h-44 bg-navy-100 overflow-hidden relative">
          <img v-if="a.thumbnail_url" :src="a.thumbnail_url" :alt="a.title" class="w-full h-full object-cover group-hover:scale-105 transition duration-300" />
          <div v-else class="w-full h-full flex items-center justify-center">
            <i class="bi bi-newspaper text-4xl text-navy-300"></i>
          </div>
          <span class="absolute top-3 left-3 text-xs px-2.5 py-1 rounded-full bg-gold-400 text-white font-semibold shadow">
            {{ a.category || 'General' }}
          </span>
        </div>
        <div class="p-5">
          <div class="flex items-center gap-2 text-xs text-navy-400 mb-2">
            <i class="bi bi-calendar3"></i>
            <span>{{ formatDate(a.created_at) }}</span>
          </div>
          <h3 class="font-serif font-bold text-navy-800 text-lg mb-2 line-clamp-2 leading-tight">{{ a.title }}</h3>
          <p v-if="a.summary" class="text-sm text-navy-500 mb-3 line-clamp-2">{{ a.summary }}</p>
          <div v-if="a.ai_summary" class="text-xs bg-purple-50 text-purple-700 p-2.5 rounded-lg mb-3 line-clamp-2 border border-purple-100">
            <i class="bi bi-stars mr-1"></i> {{ a.ai_summary }}
          </div>
          <div class="flex items-center justify-between pt-3 border-t border-navy-100">
            <span class="text-xs text-navy-400 flex items-center gap-1">
              <i class="bi bi-person-badge"></i>
              {{ a.author_dept_name || a.author_gov_name || 'Admin' }}
            </span>
            <div class="flex gap-3">
              <router-link :to="`/articles/${a.article_id}/edit`" class="text-navy-600 hover:text-gold-500 transition text-sm font-medium flex items-center gap-1">
                <i class="bi bi-pencil-square"></i> Edit
              </router-link>
              <button @click="deleteArticle(a.article_id)" class="text-red-500 hover:text-red-700 transition text-sm font-medium flex items-center gap-1">
                <i class="bi bi-trash3"></i> Delete
              </button>
            </div>
          </div>
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
const loading = ref(true)
const articles = ref([])
const articleCategories = ref([])
const search = ref('')
const categoryFilter = ref('')

const filteredArticles = computed(() => {
  return articles.value.filter(a => {
    if (categoryFilter.value && a.category !== categoryFilter.value) return false
    if (search.value) {
      const q = search.value.toLowerCase()
      return a.title?.toLowerCase().includes(q) || a.summary?.toLowerCase().includes(q)
    }
    return true
  })
})

function formatDate(d) {
  if (!d) return ''
  return new Date(d).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric' })
}

async function deleteArticle(id) {
  if (!confirm('Are you sure you want to delete this article? This action cannot be undone.')) return
  try {
    await api.delete(`/api/v1/content/articles/${id}`)
    await loadArticles()
  } catch { alert('Failed to delete article') }
}

async function loadArticles() {
  loading.value = true
  try {
    const params = {}
    const govId = authStore.user?.government_id
    if (govId) params.government_id = govId
    const { data } = await api.get('/api/v1/content/articles', { params })
    articles.value = Array.isArray(data) ? data : (data?.articles || [])
  } catch { articles.value = [] }
  finally { loading.value = false }
}

async function loadCategories() {
  try {
    const { data } = await api.get('/api/v1/admin/article-categories')
    articleCategories.value = Array.isArray(data) ? data : []
  } catch { articleCategories.value = [] }
}

onMounted(() => {
  loadArticles()
  loadCategories()
})
</script>
