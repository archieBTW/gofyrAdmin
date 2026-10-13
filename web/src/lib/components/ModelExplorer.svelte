<script lang="ts">
  import { api } from '../api';
  import type { ModelInfo } from '../types';
  import RecordDrawer from './RecordDrawer.svelte';

  let {
    datasource = '',
    model = null as ModelInfo | null,
    onPreviewImage = (url: string) => {},
  } = $props<{
    datasource: string;
    model: ModelInfo | null;
    onPreviewImage: (url: string) => void;
  }>();

  let records = $state<Record<string, any>[]>([]);
  let total = $state(0);
  let totalPages = $state(1);
  let page = $state(1);
  let limit = $state(20);
  let search = $state('');
  let sortColumn = $state('');
  let sortDirection = $state<'asc' | 'desc'>('desc');
  let isLoading = $state(false);
  let errorMessage = $state<string | null>(null);

  // Bulk selection
  let selectedIds = $state<Set<any>>(new Set());

  // Drawer state
  let isDrawerOpen = $state(false);
  let selectedRecord = $state<Record<string, any> | null>(null);

  // Search debounce timer
  let searchTimeout: any;

  // Reactively reload records when datasource or model changes
  $effect(() => {
    if (datasource && model) {
      page = 1;
      selectedIds = new Set();
      loadRecords();
    }
  });

  async function loadRecords() {
    if (!datasource || !model) return;
    isLoading = true;
    errorMessage = null;

    const sortParam = sortColumn ? `${sortColumn} ${sortDirection}` : '';

    try {
      const res = await api.listRecords(datasource, model.name, {
        page,
        limit,
        search,
        sort: sortParam,
      });
      records = res.data || [];
      total = res.total;
      totalPages = res.total_pages || 1;
    } catch (err: any) {
      errorMessage = err.message || 'Failed to fetch records';
    } finally {
      isLoading = false;
    }
  }

  function handleSearchInput(e: Event) {
    const val = (e.target as HTMLInputElement).value;
    search = val;
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
      page = 1;
      loadRecords();
    }, 300);
  }

  function toggleSort(col: string) {
    if (sortColumn === col) {
      sortDirection = sortDirection === 'asc' ? 'desc' : 'asc';
    } else {
      sortColumn = col;
      sortDirection = 'asc';
    }
    loadRecords();
  }

  function toggleSelectAll() {
    if (!model) return;
    const pk = model.primary_key;
    if (selectedIds.size === records.length && records.length > 0) {
      selectedIds = new Set();
    } else {
      const all = new Set();
      for (const r of records) {
        all.add(r[pk]);
      }
      selectedIds = all;
    }
  }

  function toggleSelectRow(id: any) {
    const next = new Set(selectedIds);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    selectedIds = next;
  }

  function openCreate() {
    selectedRecord = null;
    isDrawerOpen = true;
  }

  function openEdit(record: Record<string, any>) {
    selectedRecord = record;
    isDrawerOpen = true;
  }

  async function deleteSingle(id: any) {
    if (!model || !datasource) return;
    if (!confirm(`Are you sure you want to delete this record (#${id})?`)) return;

    try {
      await api.deleteRecord(datasource, model.name, id);
      selectedIds.delete(id);
      loadRecords();
    } catch (err: any) {
      alert(`Delete failed: ${err.message}`);
    }
  }

  async function deleteBulk() {
    if (!model || !datasource || selectedIds.size === 0) return;
    if (!confirm(`Are you sure you want to delete ${selectedIds.size} selected records?`)) return;

    try {
      await api.bulkDeleteRecords(datasource, model.name, Array.from(selectedIds));
      selectedIds = new Set();
      loadRecords();
    } catch (err: any) {
      alert(`Bulk delete failed: ${err.message}`);
    }
  }

  // Format cell value
  function formatCell(val: any, fieldType: string): string {
    if (val === null || val === undefined) return '-';
    if (typeof val === 'object') return JSON.stringify(val);
    return String(val);
  }
</script>

<div class="flex-1 flex flex-col h-full overflow-hidden bg-[#090d16]">
  {#if model}
    <!-- Top Action Bar -->
    <div class="px-6 py-4 border-b border-slate-800 bg-slate-900/40 flex flex-col md:flex-row md:items-center justify-between gap-4 select-none">
      <div>
        <div class="flex items-center gap-3">
          <h1 class="text-xl font-bold text-white tracking-tight">{model.name}</h1>
          <span class="text-xs px-2.5 py-0.5 rounded-full bg-slate-800 text-slate-300 font-mono border border-slate-700">
            {model.table_name}
          </span>
          <span class="text-xs text-slate-400">
            {total} {total === 1 ? 'record' : 'records'}
          </span>
        </div>
      </div>

      <div class="flex items-center gap-3">
        <!-- Search Input -->
        <div class="relative w-64">
          <input
            type="text"
            placeholder={`Search ${model.name}...`}
            value={search}
            oninput={handleSearchInput}
            class="w-full pl-9 pr-4 py-1.5 rounded-xl bg-slate-800/80 border border-slate-700/80 text-white placeholder-slate-500 text-xs focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 transition-all"
          />
          <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-slate-400">
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
          </div>
        </div>

        <!-- Bulk Delete Button (shows when items are selected) -->
        {#if selectedIds.size > 0}
          <button
            onclick={deleteBulk}
            class="px-3 py-1.5 rounded-xl bg-rose-600/90 hover:bg-rose-500 text-white text-xs font-semibold flex items-center gap-1.5 shadow-lg shadow-rose-600/20 transition-all"
          >
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
            <span>Delete Selected ({selectedIds.size})</span>
          </button>
        {/if}

        <!-- Refresh Button -->
        <button
          onclick={loadRecords}
          disabled={isLoading}
          class="p-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white transition-colors"
          title="Refresh table"
        >
          <svg class={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
        </button>

        <!-- New Record Button -->
        <button
          onclick={openCreate}
          class="px-3.5 py-1.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-medium text-xs shadow-lg shadow-indigo-600/30 flex items-center gap-1.5 transition-all"
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          <span>New {model.name}</span>
        </button>
      </div>
    </div>

    <!-- Error Banner -->
    {#if errorMessage}
      <div class="m-6 p-4 rounded-xl bg-rose-950/60 border border-rose-800 text-rose-300 text-xs">
        {errorMessage}
      </div>
    {/if}

    <!-- Table Container -->
    <div class="flex-1 overflow-auto">
      <table class="w-full text-left border-collapse">
        <thead class="sticky top-0 bg-slate-900/95 backdrop-blur border-b border-slate-800 text-[11px] font-bold text-slate-400 uppercase tracking-wider select-none z-10">
          <tr>
            <!-- Select All Checkbox -->
            <th class="w-10 px-4 py-3">
              <input
                type="checkbox"
                checked={selectedIds.size === records.length && records.length > 0}
                onchange={toggleSelectAll}
                class="w-4 h-4 rounded border-slate-700 bg-slate-800 text-indigo-600 focus:ring-0"
              />
            </th>

            <!-- Columns -->
            {#each model.fields as field}
              {#if !field.hidden}
                <th
                  onclick={() => toggleSort(field.db_name)}
                  class="px-4 py-3 cursor-pointer hover:text-white transition-colors group"
                >
                  <div class="flex items-center gap-1.5">
                    <span>{field.label}</span>
                    <span class="text-slate-600 group-hover:text-slate-400 text-xs">
                      {#if sortColumn === field.db_name}
                        {sortDirection === 'asc' ? '↑' : '↓'}
                      {:else}
                        ↕
                      {/if}
                    </span>
                  </div>
                </th>
              {/if}
            {/each}

            <!-- Actions Header -->
            <th class="w-24 px-4 py-3 text-right">Actions</th>
          </tr>
        </thead>

        <tbody class="divide-y divide-slate-800/60 text-xs">
          {#each records as record}
            {@const pkVal = record[model.primary_key]}
            {@const isSelected = selectedIds.has(pkVal)}
            <tr class={`transition-colors hover:bg-slate-800/40 ${isSelected ? 'bg-indigo-950/20' : ''}`}>
              <!-- Row Checkbox -->
              <td class="px-4 py-3">
                <input
                  type="checkbox"
                  checked={isSelected}
                  onchange={() => toggleSelectRow(pkVal)}
                  class="w-4 h-4 rounded border-slate-700 bg-slate-800 text-indigo-600 focus:ring-0"
                />
              </td>

              <!-- Field Values -->
              {#each model.fields as field}
                {#if !field.hidden}
                  {@const val = record[field.db_name]}
                  <td class="px-4 py-3 text-slate-300">
                    {#if field.is_primary_key}
                      <span class="font-mono font-semibold text-indigo-400">#{val}</span>
                    {:else if field.gofyr_type === 'image' && val}
                      <!-- Clickable image preview thumbnail -->
                      <button
                        type="button"
                        onclick={() => onPreviewImage(String(val))}
                        class="relative group/thumb w-9 h-9 rounded-lg overflow-hidden border border-slate-700 bg-slate-800 hover:ring-2 hover:ring-indigo-500 transition-all block"
                        title="Click to view photo"
                      >
                        <img src={String(val)} alt="" class="w-full h-full object-cover" />
                      </button>
                    {:else if field.gofyr_type === 'gallery' && Array.isArray(val)}
                      <div class="flex items-center gap-1.5">
                        <span class="text-[10px] px-2 py-0.5 rounded-full bg-slate-800 text-slate-300 border border-slate-700 font-mono">
                          {val.length} photos
                        </span>
                        {#if val.length > 0}
                          <button
                            type="button"
                            onclick={() => onPreviewImage(String(val[0]))}
                            class="w-6 h-6 rounded overflow-hidden border border-slate-700 bg-slate-800"
                          >
                            <img src={String(val[0])} alt="" class="w-full h-full object-cover" />
                          </button>
                        {/if}
                      </div>
                    {:else if field.gofyr_type === 'file' && val}
                      <a
                        href={String(val)}
                        target="_blank"
                        rel="noopener noreferrer"
                        class="inline-flex items-center gap-1 text-indigo-400 hover:underline max-w-[150px] truncate"
                      >
                        <svg class="w-3.5 h-3.5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                        </svg>
                        <span class="truncate">{String(val).split('/').pop()}</span>
                      </a>
                    {:else if field.gofyr_type === 'boolean'}
                      <span class={`inline-block px-2 py-0.5 rounded-full text-[10px] font-semibold border ${
                        val
                          ? 'bg-emerald-950/80 text-emerald-400 border-emerald-800/60'
                          : 'bg-slate-800 text-slate-400 border-slate-700'
                      }`}>
                        {val ? 'Active' : 'Inactive'}
                      </span>
                    {:else if field.gofyr_type === 'datetime' && val}
                      <span class="text-slate-400 text-xs font-mono" title={String(val)}>
                        {String(val).substring(0, 19).replace('T', ' ')}
                      </span>
                    {:else}
                      <span class="max-w-xs truncate block" title={String(val || '')}>
                        {formatCell(val, field.type)}
                      </span>
                    {/if}
                  </td>
                {/if}
              {/each}

              <!-- Row Actions -->
              <td class="px-4 py-3 text-right">
                <div class="flex items-center justify-end gap-1.5">
                  <button
                    onclick={() => openEdit(record)}
                    class="p-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white transition-colors"
                    title="Edit Record"
                  >
                    <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                    </svg>
                  </button>

                  <button
                    onclick={() => deleteSingle(pkVal)}
                    class="p-1 rounded-lg bg-slate-800 hover:bg-rose-900/80 text-slate-400 hover:text-rose-200 transition-colors"
                    title="Delete Record"
                  >
                    <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                </div>
              </td>
            </tr>
          {/each}

          {#if records.length === 0 && !isLoading}
            <tr>
              <td colspan={model.fields.length + 2} class="py-12 text-center text-slate-500 text-xs">
                No records found. Click "New {model.name}" to add your first entry.
              </td>
            </tr>
          {/if}
        </tbody>
      </table>
    </div>

    <!-- Pagination Footer -->
    <div class="px-6 py-3 border-t border-slate-800 bg-slate-900/60 flex items-center justify-between text-xs select-none">
      <div class="text-slate-400">
        Page <span class="font-bold text-white">{page}</span> of <span class="font-bold text-white">{totalPages}</span>
        <span class="text-slate-600 ml-2">({total} total records)</span>
      </div>

      <div class="flex items-center gap-2">
        <select
          value={limit}
          onchange={(e) => {
            limit = Number((e.target as HTMLSelectElement).value);
            page = 1;
            loadRecords();
          }}
          class="px-2 py-1 rounded-lg bg-slate-800 border border-slate-700 text-slate-300 text-xs focus:ring-0"
        >
          <option value={10}>10 / page</option>
          <option value={20}>20 / page</option>
          <option value={50}>50 / page</option>
          <option value={100}>100 / page</option>
        </select>

        <button
          onclick={() => { page = Math.max(1, page - 1); loadRecords(); }}
          disabled={page <= 1 || isLoading}
          class="px-3 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 disabled:opacity-40 transition-colors"
        >
          Previous
        </button>

        <button
          onclick={() => { page = Math.min(totalPages, page + 1); loadRecords(); }}
          disabled={page >= totalPages || isLoading}
          class="px-3 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 disabled:opacity-40 transition-colors"
        >
          Next
        </button>
      </div>
    </div>
  {/if}

  <!-- Slide-over Drawer -->
  <RecordDrawer
    isOpen={isDrawerOpen}
    {model}
    {datasource}
    initialData={selectedRecord}
    onClose={() => (isDrawerOpen = false)}
    onSaved={loadRecords}
  />
</div>
