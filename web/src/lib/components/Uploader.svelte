<script lang="ts">
  import { api } from '../api';

  let {
    value = '',
    disk = 'local',
    folder = 'uploads',
    type = 'image', // 'image' | 'file'
    disabled = false,
    onChange = (url: string) => {},
  } = $props<{
    value?: string;
    disk?: string;
    folder?: string;
    type?: string;
    disabled?: boolean;
    onChange?: (url: string) => void;
  }>();

  let isUploading = $state(false);
  let uploadProgress = $state(0);
  let uploadError = $state<string | null>(null);
  let isDragging = $state(false);
  let fileInput = $state<HTMLInputElement | null>(null);
  let copied = $state(false);

  async function handleFile(file: File) {
    if (disabled || isUploading) return;
    uploadError = null;
    isUploading = true;
    uploadProgress = 0;

    try {
      const resp = await api.uploadMedia(file, disk, folder, (progress) => {
        uploadProgress = progress;
      });
      onChange(resp.data.url);
    } catch (err: any) {
      uploadError = err.message || 'Upload failed';
    } finally {
      isUploading = false;
    }
  }

  function onDrop(e: DragEvent) {
    e.preventDefault();
    isDragging = false;
    if (e.dataTransfer && e.dataTransfer.files.length > 0) {
      handleFile(e.dataTransfer.files[0]);
    }
  }

  function onDragOver(e: DragEvent) {
    e.preventDefault();
    isDragging = true;
  }

  function onDragLeave() {
    isDragging = false;
  }

  function triggerPicker() {
    if (fileInput && !disabled && !isUploading) {
      fileInput.click();
    }
  }

  function handleInputChange(e: Event) {
    const target = e.target as HTMLInputElement;
    if (target.files && target.files.length > 0) {
      handleFile(target.files[0]);
    }
  }

  function clearFile() {
    onChange('');
    if (fileInput) fileInput.value = '';
  }

  function copyUrl() {
    if (!value) return;
    navigator.clipboard.writeText(value);
    copied = true;
    setTimeout(() => {
      copied = false;
    }, 2000);
  }
</script>

<div class="w-full space-y-2">
  <input
    bind:this={fileInput}
    type="file"
    class="hidden"
    accept={type === 'image' ? 'image/*' : '*/*'}
    onchange={handleInputChange}
  />

  {#if value}
    <!-- File Preview State -->
    <div class="relative flex items-center gap-3 p-3 bg-slate-900/90 border border-slate-700/80 rounded-xl overflow-hidden group">
      {#if type === 'image'}
        <div class="relative w-16 h-16 rounded-lg overflow-hidden bg-slate-800 flex-shrink-0 border border-slate-700">
          <img src={value} alt="Preview" class="w-full h-full object-cover" />
        </div>
      {:else}
        <div class="w-12 h-12 rounded-lg bg-indigo-950/80 border border-indigo-700/50 flex items-center justify-center flex-shrink-0 text-indigo-400">
          <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
        </div>
      {/if}

      <div class="flex-1 min-w-0 pr-2">
        <div class="flex items-center gap-2">
          <span class="text-xs font-semibold uppercase px-2 py-0.5 rounded bg-slate-800 text-slate-300 border border-slate-700">
            {disk}
          </span>
          {#if folder}
            <span class="text-xs text-slate-400 truncate">/{folder}</span>
          {/if}
        </div>
        <p class="text-xs text-slate-300 font-mono truncate mt-1 select-all" title={value}>
          {value}
        </p>
      </div>

      <div class="flex items-center gap-1.5 flex-shrink-0">
        <button
          type="button"
          onclick={copyUrl}
          class="p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white transition-colors"
          title="Copy URL"
        >
          {#if copied}
            <svg class="w-4 h-4 text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
          {:else}
            <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
          {/if}
        </button>

        <button
          type="button"
          onclick={triggerPicker}
          class="p-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white transition-colors"
          title="Replace File"
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
          </svg>
        </button>

        <button
          type="button"
          onclick={clearFile}
          class="p-1.5 rounded-lg bg-slate-800 hover:bg-rose-900/80 text-slate-400 hover:text-rose-200 transition-colors"
          title="Remove File"
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
          </svg>
        </button>
      </div>
    </div>
  {:else}
    <!-- Dropzone Empty State -->
    <div
      role="button"
      tabindex="0"
      onclick={triggerPicker}
      onkeydown={(e) => e.key === 'Enter' && triggerPicker()}
      ondrop={onDrop}
      ondragover={onDragOver}
      ondragleave={onDragLeave}
      class={`border-2 border-dashed rounded-xl p-6 text-center cursor-pointer transition-all ${
        isDragging
          ? 'border-indigo-500 bg-indigo-950/20'
          : 'border-slate-700/80 hover:border-slate-600 bg-slate-900/40 hover:bg-slate-900/70'
      } ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
    >
      <div class="flex flex-col items-center justify-center gap-2">
        <div class="w-10 h-10 rounded-full bg-slate-800 border border-slate-700 flex items-center justify-center text-slate-400 group-hover:text-indigo-400 transition-colors">
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
          </svg>
        </div>
        <div class="text-sm font-medium text-slate-300">
          Click to upload or drag & drop {type === 'image' ? 'photo' : 'file'}
        </div>
        <div class="flex items-center gap-2 text-xs text-slate-500">
          <span>Target disk:</span>
          <span class="px-1.5 py-0.5 rounded bg-slate-800 text-slate-300 font-mono text-[10px] uppercase border border-slate-700">
            {disk}
          </span>
          {#if folder}
            <span>• folder: /{folder}</span>
          {/if}
        </div>
      </div>
    </div>
  {/if}

  <!-- Progress Bar -->
  {#if isUploading}
    <div class="space-y-1">
      <div class="flex justify-between text-xs text-slate-400 font-mono">
        <span>Uploading...</span>
        <span>{uploadProgress}%</span>
      </div>
      <div class="w-full bg-slate-800 rounded-full h-1.5 overflow-hidden">
        <div
          class="bg-indigo-500 h-1.5 rounded-full transition-all duration-200"
          style={`width: ${uploadProgress}%`}
        ></div>
      </div>
    </div>
  {/if}

  {#if uploadError}
    <p class="text-xs text-rose-400 font-medium">
      {uploadError}
    </p>
  {/if}
</div>
