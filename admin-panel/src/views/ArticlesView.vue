<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">Articles</h1>
        <p class="text-gray-500 text-sm mt-1">Manage government articles and announcements</p>
      </div>
      <button @click="openEditor()" class="bg-civic-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-civic-700 transition flex items-center gap-2">
        <span>+</span> New Article
      </button>
    </div>

    <!-- Filters -->
    <div class="flex gap-3 mb-6">
      <input v-model="search" type="text" placeholder="Search articles..." class="border border-gray-300 rounded-lg px-3 py-2 text-sm flex-1 max-w-sm focus:ring-civic-500 focus:border-civic-500" />
      <select v-model="categoryFilter" class="border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-civic-500 focus:border-civic-500">
        <option value="">All Categories</option>
        <option v-for="c in categories" :key="c" :value="c">{{ c }}</option>
      </select>
    </div>

    <!-- Articles Grid -->
    <div v-if="loading" class="text-center py-12 text-gray-400">Loading articles...</div>
    <div v-else-if="filteredArticles.length === 0" class="text-center py-12 text-gray-400">No articles found</div>
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div v-for="a in filteredArticles" :key="a.id" class="bg-white rounded-xl border border-gray-200 overflow-hidden hover:shadow-md transition group">
        <div class="p-5">
          <div class="flex items-center gap-2 mb-2">
            <span class="text-xs px-2 py-0.5 rounded-full bg-blue-100 text-blue-700">{{ a.category || 'General' }}</span>
            <span class="text-xs text-gray-400">{{ new Date(a.created_at).toLocaleDateString() }}</span>
          </div>
          <h3 class="font-semibold text-gray-900 mb-1 line-clamp-2">{{ a.title }}</h3>
          <p v-if="a.subtitle" class="text-sm text-gray-500 mb-3 line-clamp-2">{{ a.subtitle }}</p>
          <div class="flex items-center justify-between mt-3 pt-3 border-t">
            <span class="text-xs text-gray-400">By {{ a.author_name || 'Admin' }}</span>
            <div class="flex gap-2">
              <button @click="openEditor(a)" class="text-civic-600 hover:text-civic-700 text-sm font-medium">Edit</button>
              <button @click="deleteArticle(a.id)" class="text-red-500 hover:text-red-700 text-sm font-medium">Delete</button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Editor Modal -->
    <div v-if="showEditor" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" @click.self="showEditor = false">
      <div class="bg-white rounded-2xl shadow-xl max-w-3xl w-full max-h-[90vh] overflow-y-auto">
        <div class="p-6 border-b flex items-center justify-between">
          <h2 class="text-xl font-bold text-gray-900">{{ editingId ? 'Edit Article' : 'New Article' }}</h2>
          <button @click="showEditor = false" class="text-gray-400 hover:text-gray-600">✕</button>
        </div>
        <form @submit.prevent="saveArticle" class="p-6 space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Title</label>
            <input v-model="form.title" required class="w-full border border-gray-300 rounded-lg px-3 py-2 focus:ring-civic-500 focus:border-civic-500" placeholder="Article title" />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Subtitle</label>
            <input v-model="form.subtitle" class="w-full border border-gray-300 rounded-lg px-3 py-2 focus:ring-civic-500 focus:border-civic-500" placeholder="Brief subtitle" />
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">Category</label>
              <input v-model="form.category" class="w-full border border-gray-300 rounded-lg px-3 py-2 focus:ring-civic-500 focus:border-civic-500" placeholder="e.g. Infrastructure, Health" />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 mb-1">Government ID</label>
              <input v-model.number="form.government_id" type="number" required class="w-full border border-gray-300 rounded-lg px-3 py-2 focus:ring-civic-500 focus:border-civic-500" />
            </div>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Content</label>
            <textarea v-model="form.content" rows="12" required class="w-full border border-gray-300 rounded-lg px-3 py-2 focus:ring-civic-500 focus:border-civic-500 font-mono text-sm" placeholder="Article content (supports plain text)"></textarea>
          </div>
          <div class="flex justify-end gap-3 pt-2">
            <button type="button" @click="showEditor = false" class="px-4 py-2 text-sm text-gray-600 hover:text-gray-800">Cancel</button>
            <button type="submit" class="bg-civic-600 text-white px-6 py-2 rounded-lg text-sm font-medium hover:bg-civic-700 transition">
              {{ editingId ? 'Update' : 'Publish' }}
            </button>
          </div>
        </form>
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
const search = ref('')
const categoryFilter = ref('')
const showEditor = ref(false)
const editingId = ref(null)
const form = ref({ title: '', subtitle: '', category: '', content: '', government_id: 0 })

const categories = computed(() => {
  const cats = new Set(articles.value.map(a => a.category).filter(Boolean))
  return [...cats]
})

const filteredArticles = computed(() => {
  return articles.value.filter(a => {
    if (categoryFilter.value && a.category !== categoryFilter.value) return false
    if (search.value) {
      const q = search.value.toLowerCase()
      return a.title?.toLowerCase().includes(q) || a.subtitle?.toLowerCase().includes(q)
    }
    return true
  })
})

function openEditor(article = null) {
  if (article) {
    editingId.value = article.id
    form.value = { title: article.title, subtitle: article.subtitle || '', category: article.category || '', content: article.content || '', government_id: article.government_id || authStore.user?.government_id || 0 }
  } else {
    editingId.value = null
    form.value = { title: '', subtitle: '', category: '', content: '', government_id: authStore.user?.government_id || 0 }
  }
  showEditor.value = true
}

async function saveArticle() {
  const payload = { ...form.value, author_id: authStore.user?.id }
  try {
    if (editingId.value) {
      await api.put(`/api/v1/content/articles/${editingId.value}`, payload)
    } else {
      await api.post('/api/v1/content/articles', payload)
    }
    showEditor.value = false
    await loadArticles()
  } catch (e) {
    alert('Failed to save article: ' + (e.response?.data?.error || e.message))
  }
}

async function deleteArticle(id) {
  if (!confirm('Delete this article?')) return
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

onMounted(loadArticles)
</script>
