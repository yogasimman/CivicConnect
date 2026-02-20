<template>
  <div class="p-6 max-w-7xl mx-auto">
    <div class="flex items-center justify-between mb-8">
      <div>
        <h1 class="page-title">Community Posts</h1>
        <p class="page-subtitle">View citizen posts and respond with official government replies</p>
      </div>
      <div class="flex items-center gap-2 text-sm text-navy-500">
        <i class="bi bi-chat-square-text"></i>
        <span>{{ posts.length }} post{{ posts.length !== 1 ? 's' : '' }}</span>
      </div>
    </div>

    <!-- Filters -->
    <div class="card p-4 mb-6">
      <div class="flex flex-wrap gap-3 items-center">
        <div class="relative flex-1 max-w-sm">
          <i class="bi bi-search absolute left-3 top-1/2 -translate-y-1/2 text-navy-400"></i>
          <input v-model="search" type="text" placeholder="Search posts..." class="form-input pl-10" />
        </div>
        <div class="flex gap-1">
          <button v-for="f in ['all', 'replied', 'unreplied']" :key="f" @click="filter = f"
            :class="['px-3 py-1.5 rounded-full text-sm font-medium transition', filter === f ? 'bg-navy-800 text-white' : 'bg-navy-100 text-navy-600 hover:bg-navy-200']">
            {{ f === 'all' ? 'All' : f === 'replied' ? 'Replied' : 'Needs Reply' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="text-center py-16 text-navy-400">
      <i class="bi bi-arrow-clockwise animate-spin text-3xl block mb-2"></i>
      Loading posts...
    </div>

    <!-- Empty -->
    <div v-else-if="filteredPosts.length === 0" class="card p-16 text-center">
      <i class="bi bi-chat-square text-5xl text-navy-300 mb-3 block"></i>
      <h3 class="text-lg font-semibold text-navy-700">No posts found</h3>
      <p class="text-navy-400 mt-1">Community posts from citizens will appear here</p>
    </div>

    <!-- Posts -->
    <div v-else class="space-y-4">
      <div v-for="post in filteredPosts" :key="post.post_id" class="card overflow-hidden">
        <!-- Post Header -->
        <div class="p-5 border-b border-navy-100">
          <div class="flex items-start justify-between">
            <div class="flex items-start gap-3">
              <div class="w-10 h-10 bg-navy-200 rounded-full flex items-center justify-center">
                <i class="bi bi-person text-navy-500"></i>
              </div>
              <div>
                <p class="font-medium text-navy-800">Citizen #{{ post.user_id }}</p>
                <p class="text-xs text-navy-400 flex items-center gap-1">
                  <i class="bi bi-clock"></i> {{ formatDate(post.created_at) }}
                  <span v-if="post.location_name" class="ml-2 flex items-center gap-1">
                    <i class="bi bi-geo-alt"></i> {{ post.location_name }}
                  </span>
                </p>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <span v-if="hasOfficialReply(post)" class="badge badge-success flex items-center gap-1">
                <i class="bi bi-shield-check"></i> Replied
              </span>
              <span v-else class="badge badge-warning flex items-center gap-1">
                <i class="bi bi-clock-history"></i> Awaiting Reply
              </span>
              <button @click="togglePost(post.post_id)" class="text-navy-400 hover:text-navy-700 transition p-1">
                <i :class="expandedPosts.includes(post.post_id) ? 'bi bi-chevron-up' : 'bi bi-chevron-down'"></i>
              </button>
            </div>
          </div>
          <!-- Post Content -->
          <div class="mt-3">
            <h3 v-if="post.title" class="font-serif font-bold text-navy-800 text-lg mb-1">{{ post.title }}</h3>
            <p class="text-navy-600 whitespace-pre-line">{{ post.content }}</p>
          </div>
          <!-- Post Stats -->
          <div class="mt-3 flex items-center gap-4 text-xs text-navy-400">
            <span class="flex items-center gap-1"><i class="bi bi-heart"></i> {{ post.like_count || 0 }} likes</span>
            <span class="flex items-center gap-1"><i class="bi bi-chat-dots"></i> {{ (postComments[post.post_id] || []).length }} comments</span>
            <span v-if="post.post_type" class="badge badge-info text-xs">{{ post.post_type }}</span>
          </div>
        </div>

        <!-- Expanded: Comments + Reply -->
        <div v-if="expandedPosts.includes(post.post_id)" class="bg-navy-50">
          <!-- Comments List -->
          <div class="p-5 space-y-3">
            <h4 class="text-sm font-semibold text-navy-700 flex items-center gap-1 mb-2">
              <i class="bi bi-chat-left-text"></i> Comments
            </h4>
            <div v-if="!(postComments[post.post_id] || []).length" class="text-sm text-navy-400 italic">No comments yet</div>
            <div v-for="c in (postComments[post.post_id] || [])" :key="c.comment_id"
              :class="['rounded-lg p-3 text-sm', c.is_official ? 'bg-gold-50 border-2 border-gold-400' : 'bg-white border border-navy-200']">
              <div class="flex items-center justify-between mb-1">
                <div class="flex items-center gap-2">
                  <i :class="c.is_official ? 'bi bi-shield-fill-check text-gold-500' : 'bi bi-person-circle text-navy-400'"></i>
                  <span class="font-medium" :class="c.is_official ? 'text-gold-700' : 'text-navy-700'">
                    {{ c.is_official ? (c.dept_name || 'Government Official') : `Citizen #${c.user_id}` }}
                  </span>
                  <span v-if="c.is_official" class="badge badge-success text-xs">Official</span>
                </div>
                <span class="text-xs text-navy-400">{{ formatDate(c.created_at) }}</span>
              </div>
              <p :class="c.is_official ? 'text-navy-800' : 'text-navy-600'">{{ c.content }}</p>
            </div>
          </div>

          <!-- Reply Form -->
          <div class="p-5 border-t border-navy-200 bg-white">
            <h4 class="text-sm font-semibold text-navy-700 flex items-center gap-2 mb-3">
              <i class="bi bi-shield-fill-check text-gold-500"></i> Official Government Response
            </h4>
            <div class="flex gap-3">
              <textarea v-model="replyText[post.post_id]" rows="3" class="form-input flex-1" placeholder="Write your official response to this citizen post..."></textarea>
              <div class="flex flex-col gap-2">
                <button @click="submitReply(post.post_id)" :disabled="!replyText[post.post_id]?.trim() || replying[post.post_id]"
                  class="btn-primary text-sm px-4 flex items-center gap-1.5 whitespace-nowrap">
                  <i :class="replying[post.post_id] ? 'bi bi-arrow-clockwise animate-spin' : 'bi bi-send'"></i>
                  {{ replying[post.post_id] ? 'Sending...' : 'Send Reply' }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Pagination -->
    <div v-if="totalPages > 1" class="flex justify-center items-center gap-2 mt-6">
      <button @click="page = Math.max(1, page - 1)" :disabled="page <= 1" class="btn-secondary text-sm">
        <i class="bi bi-chevron-left"></i>
      </button>
      <span class="text-sm text-navy-500">Page {{ page }} of {{ totalPages }}</span>
      <button @click="page = Math.min(totalPages, page + 1)" :disabled="page >= totalPages" class="btn-secondary text-sm">
        <i class="bi bi-chevron-right"></i>
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, reactive } from 'vue'
import { useAuthStore } from '../stores/auth'
import api from '../api'

const authStore = useAuthStore()

const loading = ref(true)
const posts = ref([])
const search = ref('')
const filter = ref('all')
const page = ref(1)
const totalPages = ref(1)
const expandedPosts = ref([])
const postComments = reactive({})
const replyText = reactive({})
const replying = reactive({})

const filteredPosts = computed(() => {
  let list = posts.value
  if (search.value) {
    const q = search.value.toLowerCase()
    list = list.filter(p => p.title?.toLowerCase().includes(q) || p.content?.toLowerCase().includes(q))
  }
  if (filter.value === 'replied') {
    list = list.filter(p => hasOfficialReply(p))
  } else if (filter.value === 'unreplied') {
    list = list.filter(p => !hasOfficialReply(p))
  }
  return list
})

function hasOfficialReply(post) {
  const comments = postComments[post.post_id] || []
  return comments.some(c => c.is_official)
}

function togglePost(postId) {
  const idx = expandedPosts.value.indexOf(postId)
  if (idx >= 0) {
    expandedPosts.value.splice(idx, 1)
  } else {
    expandedPosts.value.push(postId)
    if (!postComments[postId]) {
      loadComments(postId)
    }
  }
}

function formatDate(d) {
  if (!d) return ''
  return new Date(d).toLocaleDateString('en-IN', { day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

async function loadPosts() {
  loading.value = true
  try {
    const params = { page: page.value, limit: 20 }
    const govId = authStore.user?.government_id
    if (govId) params.government_id = govId
    const { data } = await api.get('/api/v1/content/posts', { params })
    if (Array.isArray(data)) {
      posts.value = data
      totalPages.value = 1
    } else {
      posts.value = data?.posts || []
      totalPages.value = data?.total_pages || 1
    }
    // Auto-load comments for all visible posts
    for (const p of posts.value) {
      loadComments(p.post_id)
    }
  } catch { posts.value = [] }
  finally { loading.value = false }
}

async function loadComments(postId) {
  try {
    const { data } = await api.get(`/api/v1/content/comments/${postId}`)
    postComments[postId] = Array.isArray(data) ? data : (data?.comments || [])
  } catch { postComments[postId] = [] }
}

async function submitReply(postId) {
  const content = replyText[postId]?.trim()
  if (!content) return
  replying[postId] = true
  const user = authStore.user
  try {
    await api.post('/api/v1/content/comments', {
      user_id: user?.id || user?.admin_id,
      post_id: postId,
      content,
      is_official: true,
      admin_id: user?.admin_id || user?.id,
      dept_id: user?.department_id,
      dept_name: user?.department_name || authStore.departmentName || authStore.governmentName,
    })
    replyText[postId] = ''
    await loadComments(postId)
  } catch {
    alert('Failed to send reply')
  } finally {
    replying[postId] = false
  }
}

watch(page, loadPosts)

onMounted(() => {
  loadPosts()
})
</script>
