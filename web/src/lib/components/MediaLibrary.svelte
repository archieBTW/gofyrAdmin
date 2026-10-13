<script lang="ts">
  import { api } from '../api';
  import type { MediaFile } from '../types';
  import Uploader from './Uploader.svelte';

  let {
    disks = ['local'],
    defaultDisk = 'local',
    onPreviewImage = (url: string) => {},
  } = $props<{
    disks: string[];
    defaultDisk: string;
    onPreviewImage: (url: string) => void;
  }>();

  let files = $state<MediaFile[]>([]);
  let total = $state(0);
  let totalPages = $state(1);
  let page = $state(1);
  let limit = $state(24);
  let search = $state('');
  let selectedDisk = $state('all');
  let selectedType = $state('all');
  let isLoading = $state(false);
  let showUploadModal = $state(false);
  let copiedId = $state<number | null>(null);

  $effect(() => {
    loadFiles();
  });

  async function loadFiles() {
    isLoading = true;
    try {
      const res = await api.listMediaFiles({
        page,
        limit,
        search,
        disk: selectedDisk === 'all' ? undefined : selectedDisk,
        type: selectedType === 'all' ? undefined : selectedType,
      });
      files = res.data || [];
      total = res.total;
      totalPages = res.total_pages || 1;
    } catch (err: any) {
      console.error('Failed to load media files:', err);
    } finally {
      isLoading = false;
    }
  }

  function handleSearch(e: Event) {
    search = (e.target as HTMLInputElement).value;
    page = 1;
    loadFiles();
  }

  function formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  function copyUrl(file: MediaFile) {
    navigator.clipboard.writeText(file.url);
    copiedId = file.id;
    setTimeout(() => (copiedId = null), 2000);
  }

  async function deleteFile(file: MediaFile) {
    if (!confirm(`Are you sure you want to delete '${file.filename}'?`)) return;
    try {
      await api.deleteMediaFile(file.id);
      loadFiles();
    } catch (err: any) {
      alert(`Delete failed: ${err.message}`);
    }
  }
</script>

<div class="flex-1 flex flex-col h-full overflow-hidden bg-[#090d16]">
  <!-- Header Bar -->
  <div class="px-6 py-4 border-b border-slate-800 bg-slate-900/40 flex flex-col md:flex-row md:items-center justify-between gap-4 select-none">
    <div>
      <h1 class="text-xl font-bold text-white tracking-tight flex items-center gap-2">
        <span>Media & Storage</span>
        <span class="text-xs px-2.5 py-0.5 rounded-full bg-slate-800 text-slate-300 font-mono border border-slate-700">
          {total} files
        </span>
      </h1>
      <p class="text-xs text-slate-400 mt-0.5">
        Manage photos, gallery assets, and attachments across Local disk and S3
      </p>
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-3">
      <!-- Search -->
      <div class="relative w-64">
        <input
          type="text"
          placeholder="Search files..."
          value={search}
          oninput={handleSearch}
          class="w-full pl-9 pr-4 py-1.5 rounded-xl bg-slate-800/80 border border-slate-700/80 text-white placeholder-slate-500 text-xs focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500"
        />
        <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-slate-400">
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        </div>
      </div>

      <!-- Upload Modal Toggle -->
      <button
        onclick={() => (showUploadModal = true)}
        class="px-3.5 py-1.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-medium text-xs shadow-lg shadow-indigo-600/30 flex items-center gap-1.5 transition-all"
      >
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
        </svg>
        <span>Upload Media</span>
      </button>
    </div>
  </div>

  <!-- Filters Row -->
  <div class="px-6 py-2.5 border-b border-slate-800/60 bg-slate-900/20 flex items-center justify-between text-xs select-none">
    <div class="flex items-center gap-4">
      <!-- Disk Filter -->
      <div class="flex items-center gap-1.5">
        <span class="text-slate-500 font-semibold uppercase text-[10px]">Disk:</span>
        <button
          onclick={() => { selectedDisk = 'all'; page = 1; loadFiles(); }}
          class={`px-2 py-0.5 rounded-lg transition-colors ${selectedDisk === 'all' ? 'bg-indigo-600 text-white font-semibold' : 'text-slate-400 hover:text-white'}`}
        >
          All
        </button>
        {#each disks as disk}
          <button
            onclick={() => { selectedDisk = disk; page = 1; loadFiles(); }}
            class={`px-2 py-0.5 rounded-lg uppercase font-mono text-[11px] transition-colors ${selectedDisk === disk ? 'bg-indigo-600 text-white font-semibold' : 'text-slate-400 hover:text-white'}`}
          >
            {disk}
          </button>
        {/each}
      </div>

      <!-- Type Filter -->
      <div class="flex items-center gap-1.5 pl-4 border-l border-slate-800">
        <span class="text-slate-500 font-semibold uppercase text-[10px]">Type:</span>
        <button
          onclick={() => { selectedType = 'all'; page = 1; loadFiles(); }}
          class={`px-2 py-0.5 rounded-lg transition-colors ${selectedType === 'all' ? 'bg-indigo-600 text-white font-semibold' : 'text-slate-400 hover:text-white'}`}
        >
          All
        </button>
        <button
          onclick={() => { selectedType = 'image'; page = 1; loadFiles(); }}
          class={`px-2 py-0.5 rounded-lg transition-colors ${selectedType === 'image' ? 'bg-indigo-600 text-white font-semibold' : 'text-slate-400 hover:text-white'}`}
        >
          Photos
        </button>
        <button
          onclick={() => { selectedType = 'doc'; page = 1; loadFiles(); }}
          class={`px-2 py-0.5 rounded-lg transition-colors ${selectedType === 'doc' ? 'bg-indigo-600 text-white font-semibold' : 'text-slate-400 hover:text-white'}`}
        >
          Files
        </button>
      </div>
    </div>

    <button
      onclick={loadFiles}
      class="text-slate-400 hover:text-white flex items-center gap-1"
      title="Refresh"
    >
      <svg class={`w-3.5 h-3.5 ${isLoading ? 'animate-spin' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
      </svg>
      <span>Refresh</span>
    </button>
  </div>

  <!-- Media Grid -->
  <div class="flex-1 overflow-y-auto p-6">
    {#if files.length > 0}
      <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4">
        {#each files as file}
          {@const isImage = file.mime_type.startsWith('image/')}
          <div class="group relative bg-slate-900/80 border border-slate-800 rounded-xl overflow-hidden hover:border-slate-700 hover:shadow-xl transition-all flex flex-col">
            <!-- Thumbnail Box -->
            <div class="aspect-square bg-slate-950 flex items-center justify-center overflow-hidden relative">
              {#if isImage}
                <img
                  src={file.url}
                  alt={file.original_name}
                  class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                />
                <!-- Preview Overlay Button -->
                <button
                  type="button"
                  onclick={() => onPreviewImage(file.url)}
                  class="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 flex items-center justify-center text-white transition-opacity backdrop-blur-[2px]"
                  title="View full image"
                >
                  <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                  </svg>
                </button>
              {:else}
                <div class="text-indigo-400">
                  <svg class="w-12 h-12" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                  </svg>
                </div>
              {/if}

              <!-- Disk Badge -->
              <span class="absolute top-2 left-2 px-1.5 py-0.5 rounded bg-slate-900/90 text-slate-300 border border-slate-700/80 text-[10px] font-mono uppercase font-bold">
                {file.disk}
              </span>
            </div>

            <!-- Metadata Info -->
            <div class="p-2.5 flex-1 flex flex-col justify-between">
              <div>
                <p class="text-xs font-semibold text-white truncate" title={file.original_name}>
                  {file.original_name}
                </p>
                <div class="flex items-center justify-between text-[10px] text-slate-400 mt-1">
                  <span>{formatBytes(file.size)}</span>
                  <span>/{file.folder}</span>
                </div>
              </div>

              <!-- Action Bar -->
              <div class="mt-2 pt-2 border-t border-slate-800/80 flex items-center justify-between text-xs">
                <button
                  type="button"
                  onclick={() => copyUrl(file)}
                  class="text-indigo-400 hover:text-indigo-300 font-medium flex items-center gap-1"
                  title="Copy URL"
                >
                  {#if copiedId === file.id}
                    <span class="text-emerald-400 font-semibold">Copied!</span>
                  {:else}
                    <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                    </svg>
                    <span>Copy</span>
                  {/if}
                </button>

                <button
                  type="button"
                  onclick={() => deleteFile(file)}
                  class="text-slate-500 hover:text-rose-400 transition-colors"
                  title="Delete file"
                >
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              </div>
            </div>
          </div>
        {/each}
      </div>
    {:else if !isLoading}
      <div class="h-64 flex flex-col items-center justify-center text-slate-500 text-xs">
        <svg class="w-10 h-10 mb-2 text-slate-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
        </svg>
        <span>No media files found. Upload a photo or file to get started.</span>
      </div>
    {/if}
  </div>

  <!-- Pagination Footer -->
  <div class="px-6 py-3 border-t border-slate-800 bg-slate-900/60 flex items-center justify-between text-xs select-none">
    <div class="text-slate-400">
      Page <span class="font-bold text-white">{page}</span> of <span class="font-bold text-white">{totalPages}</span>
    </div>

    <div class="flex items-center gap-2">
      <button
        onclick={() => { page = Math.max(1, page - 1); loadFiles(); }}
        disabled={page <= 1 || isLoading}
        class="px-3 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 disabled:opacity-40"
      >
        Previous
      </button>

      <button
        onclick={() => { page = Math.min(totalPages, page + 1); loadFiles(); }}
        disabled={page >= totalPages || isLoading}
        class="px-3 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 disabled:opacity-40"
      >
        Next
      </button>
    </div>
  </div>

  <!-- Direct Upload Modal -->
  {#if showUploadModal}
    <div
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4"
      onclick={() => (showUploadModal = false)}
      onkeydown={(e) => e.key === 'Escape' && (showUploadModal = false)}
      role="dialog"
      aria-modal="true"
      tabindex="-1"
    >
      <div
        class="bg-slate-900 border border-slate-800 rounded-2xl p-6 max-w-md w-full shadow-2xl space-y-4"
        onclick={(e) => e.stopPropagation()}
        onkeydown={(e) => e.stopPropagation()}
        role="document"
        tabindex="0"
      >
        <div class="flex items-center justify-between">
          <h2 class="text-base font-bold text-white">Upload New Media</h2>
          <button
            type="button"
            onclick={() => (showUploadModal = false)}
            class="text-slate-400 hover:text-white"
          >
            ✕
          </button>
        </div>

        <Uploader
          disk={defaultDisk}
          folder="media"
          type="image"
          onChange={() => {
            showUploadModal = false;
            loadFiles();
          }}
        />
      </div>
    </div>
  {/if}
</div>
