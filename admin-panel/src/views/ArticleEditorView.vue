<template>
  <div class="flex h-screen bg-navy-50">
    <!-- Main Editor Area -->
    <div class="flex-1 flex flex-col overflow-hidden">
      <!-- Top Bar -->
      <div class="bg-white border-b border-navy-200 px-6 py-3 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <router-link to="/articles" class="text-navy-400 hover:text-navy-700 transition">
            <i class="bi bi-arrow-left text-xl"></i>
          </router-link>
          <h1 class="font-serif text-lg font-bold text-navy-800">{{ isEditing ? 'Edit Article' : 'New Article' }}</h1>
          <span v-if="saving" class="text-xs text-navy-400 flex items-center gap-1">
            <i class="bi bi-arrow-clockwise animate-spin"></i> Saving...
          </span>
          <span v-else-if="lastSaved" class="text-xs text-green-600 flex items-center gap-1">
            <i class="bi bi-check-circle"></i> Saved
          </span>
        </div>
        <div class="flex items-center gap-3">
          <button @click="previewMode = !previewMode" class="btn-secondary text-sm flex items-center gap-1.5">
            <i :class="previewMode ? 'bi bi-pencil' : 'bi bi-eye'"></i>
            {{ previewMode ? 'Editor' : 'Preview' }}
          </button>
          <button @click="saveArticle" :disabled="saving" class="btn-primary text-sm flex items-center gap-1.5">
            <i class="bi bi-cloud-upload"></i>
            {{ isEditing ? 'Update' : 'Publish' }}
          </button>
        </div>
      </div>

      <!-- Toolbar -->
      <div v-if="editor && !previewMode" class="bg-white border-b border-navy-200 px-6 py-2 flex items-center gap-1 flex-wrap">
        <button @click="editor.chain().focus().toggleBold().run()" :class="['toolbar-btn', { active: editor.isActive('bold') }]" title="Bold">
          <i class="bi bi-type-bold"></i>
        </button>
        <button @click="editor.chain().focus().toggleItalic().run()" :class="['toolbar-btn', { active: editor.isActive('italic') }]" title="Italic">
          <i class="bi bi-type-italic"></i>
        </button>
        <button @click="editor.chain().focus().toggleUnderline().run()" :class="['toolbar-btn', { active: editor.isActive('underline') }]" title="Underline">
          <i class="bi bi-type-underline"></i>
        </button>
        <button @click="editor.chain().focus().toggleStrike().run()" :class="['toolbar-btn', { active: editor.isActive('strike') }]" title="Strikethrough">
          <i class="bi bi-type-strikethrough"></i>
        </button>
        <div class="w-px h-6 bg-navy-200 mx-1"></div>
        <button @click="editor.chain().focus().toggleHeading({ level: 1 }).run()" :class="['toolbar-btn', { active: editor.isActive('heading', { level: 1 }) }]" title="Heading 1">
          <i class="bi bi-type-h1"></i>
        </button>
        <button @click="editor.chain().focus().toggleHeading({ level: 2 }).run()" :class="['toolbar-btn', { active: editor.isActive('heading', { level: 2 }) }]" title="Heading 2">
          <i class="bi bi-type-h2"></i>
        </button>
        <button @click="editor.chain().focus().toggleHeading({ level: 3 }).run()" :class="['toolbar-btn', { active: editor.isActive('heading', { level: 3 }) }]" title="Heading 3">
          <i class="bi bi-type-h3"></i>
        </button>
        <div class="w-px h-6 bg-navy-200 mx-1"></div>
        <button @click="editor.chain().focus().toggleBulletList().run()" :class="['toolbar-btn', { active: editor.isActive('bulletList') }]" title="Bullet List">
          <i class="bi bi-list-ul"></i>
        </button>
        <button @click="editor.chain().focus().toggleOrderedList().run()" :class="['toolbar-btn', { active: editor.isActive('orderedList') }]" title="Ordered List">
          <i class="bi bi-list-ol"></i>
        </button>
        <button @click="editor.chain().focus().toggleBlockquote().run()" :class="['toolbar-btn', { active: editor.isActive('blockquote') }]" title="Quote">
          <i class="bi bi-quote"></i>
        </button>
        <button @click="editor.chain().focus().setHorizontalRule().run()" class="toolbar-btn" title="Horizontal Rule">
          <i class="bi bi-hr"></i>
        </button>
        <div class="w-px h-6 bg-navy-200 mx-1"></div>
        <button @click="editor.chain().focus().setTextAlign('left').run()" :class="['toolbar-btn', { active: editor.isActive({ textAlign: 'left' }) }]" title="Align Left">
          <i class="bi bi-text-left"></i>
        </button>
        <button @click="editor.chain().focus().setTextAlign('center').run()" :class="['toolbar-btn', { active: editor.isActive({ textAlign: 'center' }) }]" title="Align Center">
          <i class="bi bi-text-center"></i>
        </button>
        <button @click="editor.chain().focus().setTextAlign('right').run()" :class="['toolbar-btn', { active: editor.isActive({ textAlign: 'right' }) }]" title="Align Right">
          <i class="bi bi-text-right"></i>
        </button>
        <div class="w-px h-6 bg-navy-200 mx-1"></div>
        <button @click="insertLink" class="toolbar-btn" title="Insert Link">
          <i class="bi bi-link-45deg"></i>
        </button>
        <button @click="showImageUpload = true" class="toolbar-btn" title="Insert Image">
          <i class="bi bi-image"></i>
        </button>
        <div class="w-px h-6 bg-navy-200 mx-1"></div>
        <button @click="editor.chain().focus().undo().run()" class="toolbar-btn" title="Undo">
          <i class="bi bi-arrow-counterclockwise"></i>
        </button>
        <button @click="editor.chain().focus().redo().run()" class="toolbar-btn" title="Redo">
          <i class="bi bi-arrow-clockwise"></i>
        </button>
      </div>

      <!-- Editor Content -->
      <div class="flex-1 overflow-y-auto">
        <div class="max-w-4xl mx-auto py-8 px-6">
          <!-- Title Input -->
          <input v-model="form.title" type="text" placeholder="Article Title" class="w-full text-3xl font-serif font-bold text-navy-900 placeholder-navy-300 border-0 outline-none mb-4 bg-transparent" />

          <!-- TipTap or Preview -->
          <div v-if="previewMode" class="prose prose-lg max-w-none">
            <div v-html="editor?.getHTML()"></div>
          </div>
          <editor-content v-else :editor="editor" class="tiptap-editor" />
        </div>
      </div>
    </div>

    <!-- Right Sidebar -->
    <div class="w-80 bg-white border-l border-navy-200 flex flex-col overflow-y-auto">
      <div class="p-5 border-b border-navy-200">
        <h3 class="font-serif font-bold text-navy-800 mb-4 flex items-center gap-2">
          <i class="bi bi-gear"></i> Article Settings
        </h3>

        <!-- Category -->
        <div class="mb-4">
          <label class="form-label">Category</label>
          <select v-model="form.category" class="form-input">
            <option value="">Select category</option>
            <option v-for="c in articleCategories" :key="c.id" :value="c.name">{{ c.name }}</option>
          </select>
        </div>

        <!-- Summary -->
        <div class="mb-4">
          <label class="form-label">Summary</label>
          <textarea v-model="form.summary" rows="3" class="form-input text-sm" placeholder="Brief description of the article"></textarea>
        </div>

        <!-- Thumbnail -->
        <div class="mb-4">
          <label class="form-label">Featured Image</label>
          <FileUpload v-model="form.thumbnail_url" accept="image/*" />
        </div>
      </div>

      <!-- AI Summary -->
      <div class="p-5 border-b border-navy-200">
        <h3 class="font-serif font-bold text-navy-800 mb-3 flex items-center gap-2">
          <i class="bi bi-stars text-purple-600"></i> AI Summary
        </h3>
        <p class="text-xs text-navy-400 mb-3">Auto-generate a summary using AI based on article content</p>
        <button @click="generateSummary" :disabled="aiLoading" class="w-full bg-purple-600 text-white rounded-lg px-4 py-2 text-sm font-medium hover:bg-purple-700 disabled:opacity-50 transition flex items-center justify-center gap-2">
          <i :class="aiLoading ? 'bi bi-arrow-clockwise animate-spin' : 'bi bi-magic'"></i>
          {{ aiLoading ? 'Generating...' : 'Generate Summary' }}
        </button>
        <textarea v-model="form.ai_summary" rows="4" class="form-input mt-3 text-sm bg-purple-50 border-purple-200 focus:border-purple-400" placeholder="AI summary will appear here..."></textarea>
      </div>

      <!-- Author Info -->
      <div class="p-5">
        <h3 class="font-serif font-bold text-navy-800 mb-3 flex items-center gap-2">
          <i class="bi bi-person-badge"></i> Author
        </h3>
        <div class="bg-navy-50 rounded-lg p-3 text-sm">
          <p class="font-medium text-navy-700">{{ authStore.user?.name || 'Administrator' }}</p>
          <p class="text-navy-400 text-xs mt-0.5">{{ authStore.governmentName || 'Government' }}</p>
          <p v-if="authStore.departmentName" class="text-navy-400 text-xs">{{ authStore.departmentName }}</p>
        </div>
      </div>
    </div>

    <!-- Image Upload Modal -->
    <div v-if="showImageUpload" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showImageUpload = false">
      <div class="bg-white rounded-xl shadow-xl max-w-md w-full p-6">
        <div class="flex items-center justify-between mb-4">
          <h3 class="font-serif font-bold text-navy-800">Insert Image</h3>
          <button @click="showImageUpload = false" class="text-navy-400 hover:text-navy-600">&times;</button>
        </div>
        <FileUpload v-model="inlineImageUrl" accept="image/*" />
        <div class="flex justify-end gap-3 mt-4">
          <button @click="showImageUpload = false" class="btn-secondary text-sm">Cancel</button>
          <button @click="insertImage" :disabled="!inlineImageUrl" class="btn-primary text-sm">Insert</button>
        </div>
      </div>
    </div>

    <!-- Error -->
    <div v-if="formError" class="fixed bottom-6 left-1/2 -translate-x-1/2 bg-red-600 text-white px-6 py-3 rounded-lg shadow-lg z-50 flex items-center gap-2">
      <i class="bi bi-exclamation-triangle"></i>
      {{ formError }}
      <button @click="formError = ''" class="ml-3 text-white/80 hover:text-white">&times;</button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Underline from '@tiptap/extension-underline'
import TiptapLink from '@tiptap/extension-link'
import TiptapImage from '@tiptap/extension-image'
import Placeholder from '@tiptap/extension-placeholder'
import TextAlign from '@tiptap/extension-text-align'
import { useAuthStore } from '../stores/auth'
import api from '../api'
import FileUpload from '../components/FileUpload.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const isEditing = ref(false)
const saving = ref(false)
const lastSaved = ref(false)
const previewMode = ref(false)
const aiLoading = ref(false)
const formError = ref('')
const articleCategories = ref([])
const showImageUpload = ref(false)
const inlineImageUrl = ref('')

const form = ref({
  title: '',
  summary: '',
  category: '',
  content: '',
  thumbnail_url: '',
  ai_summary: '',
})

const editor = useEditor({
  extensions: [
    StarterKit,
    Underline,
    TiptapLink.configure({ openOnClick: false }),
    TiptapImage.configure({ inline: false, allowBase64: false }),
    Placeholder.configure({ placeholder: 'Start writing your article... Use the toolbar above for formatting.' }),
    TextAlign.configure({ types: ['heading', 'paragraph'] }),
  ],
  content: '',
  editorProps: {
    attributes: {
      class: 'prose prose-lg max-w-none focus:outline-none min-h-[400px]',
    },
  },
})

function insertLink() {
  const previousUrl = editor.value?.getAttributes('link').href
  const url = window.prompt('Enter URL:', previousUrl)
  if (url === null) return
  if (url === '') {
    editor.value?.chain().focus().extendMarkRange('link').unsetLink().run()
    return
  }
  editor.value?.chain().focus().extendMarkRange('link').setLink({ href: url }).run()
}

function insertImage() {
  if (inlineImageUrl.value) {
    editor.value?.chain().focus().setImage({ src: inlineImageUrl.value }).run()
    inlineImageUrl.value = ''
    showImageUpload.value = false
  }
}

async function generateSummary() {
  const html = editor.value?.getHTML()
  if (!html) return
  aiLoading.value = true
  try {
    const { data } = await api.post('/api/v1/content/ai-summary', { text: html })
    form.value.ai_summary = data.summary || data.text || ''
  } catch {
    form.value.ai_summary = 'Failed to generate summary.'
  } finally {
    aiLoading.value = false
  }
}

async function saveArticle() {
  formError.value = ''
  if (!form.value.title.trim()) {
    formError.value = 'Title is required'
    return
  }
  saving.value = true
  const user = authStore.user
  const payload = {
    title: form.value.title,
    summary: form.value.summary,
    category: form.value.category,
    content: editor.value?.getHTML() || '',
    thumbnail_url: form.value.thumbnail_url,
    ai_summary: form.value.ai_summary,
    author_id: user?.id,
    government_id: user?.government_id,
    author_type: user?.role || 'admin',
    author_admin_id: user?.admin_id || user?.id,
    author_dept_id: user?.department_id,
    author_dept_name: user?.department_name || authStore.departmentName,
    author_gov_name: user?.government_name || authStore.governmentName,
    author_logo_url: authStore.governmentLogo,
  }
  try {
    if (isEditing.value) {
      await api.put(`/api/v1/content/articles/${route.params.id}`, payload)
    } else {
      await api.post('/api/v1/content/articles', payload)
    }
    lastSaved.value = true
    setTimeout(() => { lastSaved.value = false }, 3000)
    if (!isEditing.value) {
      router.push('/articles')
    }
  } catch (e) {
    formError.value = e.response?.data?.error || 'Failed to save article'
  } finally {
    saving.value = false
  }
}

async function loadArticle(id) {
  try {
    const { data } = await api.get(`/api/v1/content/articles/${id}`)
    const article = data
    form.value = {
      title: article.title || '',
      summary: article.summary || '',
      category: article.category || '',
      content: article.content || '',
      thumbnail_url: article.thumbnail_url || '',
      ai_summary: article.ai_summary || '',
    }
    editor.value?.commands.setContent(article.content || '')
  } catch {
    formError.value = 'Failed to load article'
  }
}

async function loadCategories() {
  try {
    const { data } = await api.get('/api/v1/admin/article-categories')
    articleCategories.value = Array.isArray(data) ? data : []
  } catch { articleCategories.value = [] }
}

onMounted(() => {
  loadCategories()
  if (route.params.id) {
    isEditing.value = true
    loadArticle(route.params.id)
  }
})

onBeforeUnmount(() => {
  editor.value?.destroy()
})
</script>

<style>
.toolbar-btn {
  @apply w-8 h-8 flex items-center justify-center rounded text-navy-600 hover:bg-navy-100 hover:text-navy-800 transition text-sm;
}
.toolbar-btn.active {
  @apply bg-navy-800 text-white;
}
.tiptap-editor .ProseMirror {
  min-height: 400px;
  outline: none;
}
.tiptap-editor .ProseMirror p.is-editor-empty:first-child::before {
  content: attr(data-placeholder);
  float: left;
  color: #94a3b8;
  pointer-events: none;
  height: 0;
}
.tiptap-editor .ProseMirror h1 { @apply text-3xl font-serif font-bold text-navy-900 mt-8 mb-4; }
.tiptap-editor .ProseMirror h2 { @apply text-2xl font-serif font-bold text-navy-800 mt-6 mb-3; }
.tiptap-editor .ProseMirror h3 { @apply text-xl font-serif font-semibold text-navy-700 mt-5 mb-2; }
.tiptap-editor .ProseMirror p { @apply text-base text-navy-700 leading-relaxed mb-4; }
.tiptap-editor .ProseMirror ul { @apply list-disc ml-6 mb-4; }
.tiptap-editor .ProseMirror ol { @apply list-decimal ml-6 mb-4; }
.tiptap-editor .ProseMirror li { @apply mb-1; }
.tiptap-editor .ProseMirror blockquote { @apply border-l-4 border-gold-400 pl-4 italic text-navy-600 my-4; }
.tiptap-editor .ProseMirror hr { @apply border-navy-200 my-8; }
.tiptap-editor .ProseMirror a { @apply text-blue-600 underline; }
.tiptap-editor .ProseMirror img { @apply rounded-lg max-w-full my-4 shadow; }
</style>
