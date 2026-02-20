<template>
  <div class="file-upload-wrapper">
    <div
      class="border-2 border-dashed rounded-lg p-4 text-center transition-colors cursor-pointer"
      :class="dragOver ? 'border-navy-500 bg-navy-50' : 'border-navy-200 hover:border-navy-300'"
      @dragover.prevent="dragOver = true"
      @dragleave.prevent="dragOver = false"
      @drop.prevent="handleDrop"
      @click="$refs.fileInput.click()"
    >
      <input ref="fileInput" type="file" class="hidden" :accept="accept" @change="handleSelect" />

      <!-- Preview -->
      <div v-if="previewUrl" class="mb-3">
        <img :src="previewUrl" alt="Preview" class="max-h-32 mx-auto rounded-md object-cover" />
      </div>

      <!-- Upload icon + text -->
      <div v-if="!uploading">
        <i class="bi bi-cloud-arrow-up text-2xl text-navy-400"></i>
        <p class="text-sm text-navy-500 mt-1">{{ previewUrl ? 'Click or drag to replace' : 'Click or drag file to upload' }}</p>
        <p class="text-xs text-navy-300 mt-1">JPG, PNG, GIF, WebP, SVG — Max 10MB</p>
      </div>

      <!-- Progress -->
      <div v-else class="py-2">
        <div class="w-full bg-navy-100 rounded-full h-2 mb-2">
          <div class="bg-navy-600 h-2 rounded-full transition-all" style="width: 100%; animation: pulse 1s infinite"></div>
        </div>
        <p class="text-xs text-navy-400">Uploading...</p>
      </div>
    </div>
    <p v-if="error" class="text-xs text-red-600 mt-1">{{ error }}</p>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import api from '../api'

const props = defineProps({
  modelValue: { type: String, default: '' },
  accept: { type: String, default: 'image/*' },
})
const emit = defineEmits(['update:modelValue'])

const dragOver = ref(false)
const uploading = ref(false)
const error = ref('')
const previewUrl = ref(props.modelValue || '')

watch(() => props.modelValue, (v) => { previewUrl.value = v || '' })

function handleDrop(e) {
  dragOver.value = false
  const file = e.dataTransfer.files[0]
  if (file) uploadFile(file)
}

function handleSelect(e) {
  const file = e.target.files[0]
  if (file) uploadFile(file)
}

async function uploadFile(file) {
  if (file.size > 10 * 1024 * 1024) {
    error.value = 'File too large (max 10MB)'
    return
  }
  error.value = ''
  uploading.value = true

  const formData = new FormData()
  formData.append('file', file)

  try {
    const { data } = await api.post('/api/v1/content/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
    previewUrl.value = data.url
    emit('update:modelValue', data.url)
  } catch (e) {
    error.value = e.response?.data?.error || 'Upload failed'
  } finally {
    uploading.value = false
  }
}
</script>
