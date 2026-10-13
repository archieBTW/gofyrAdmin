<script lang="ts">
  import { api, getToken } from '../api';
  import type { RouteInfo, WorkbenchExecuteResponse } from '../types';
  import CreateRouteModal from './CreateRouteModal.svelte';

  let routes = $state<RouteInfo[]>([]);
  let isLoadingRoutes = $state(false);
  let routeSearch = $state('');
  let methodFilter = $state('ALL');
  let isCreateRouteOpen = $state(false);
  let mobileView = $state<'list' | 'detail'>('detail');

  // Active Request Builder state
  let method = $state('GET');
  let path = $state('');
  let queryParams = $state<{ key: string; value: string }[]>([{ key: '', value: '' }]);
  let headers = $state<{ key: string; value: string }[]>([
    { key: 'Accept', value: 'application/json' },
  ]);
  let body = $state('{\n  \n}');
  let activeTab = $state<'params' | 'headers' | 'body'>('params');

  // Response state
  let isExecuting = $state(false);
  let response = $state<WorkbenchExecuteResponse | null>(null);
  let executeError = $state<string | null>(null);
  let responseTab = $state<'body' | 'headers'>('body');
  let copiedResponse = $state(false);

  $effect(() => {
    loadRoutes();
  });

  async function loadRoutes() {
    isLoadingRoutes = true;
    try {
      const res = await api.listWorkbenchRoutes();
      routes = res.routes || [];
      if (routes.length > 0 && !path) {
        selectRoute(routes[0]);
      }
    } catch (err: any) {
      console.error('Failed to load routes:', err);
    } finally {
      isLoadingRoutes = false;
    }
  }

  function selectRoute(r: RouteInfo) {
    method = r.method;
    path = r.path;
    mobileView = 'detail';

    // Reset params if new route selected
    if (r.method === 'GET' || r.method === 'DELETE') {
      activeTab = 'params';
    } else {
      activeTab = 'body';
    }
  }

  const filteredRoutes = $derived(
    routes.filter((r) => {
      const matchesSearch =
        r.path.toLowerCase().includes(routeSearch.toLowerCase()) ||
        r.handler.toLowerCase().includes(routeSearch.toLowerCase());
      const matchesMethod =
        methodFilter === 'ALL' || r.method.toUpperCase() === methodFilter;
      return matchesSearch && matchesMethod;
    })
  );

  function getMethodBadge(m: string) {
    switch (m.toUpperCase()) {
      case 'GET':
        return { bg: 'bg-emerald-950/80', text: 'text-emerald-400', border: 'border-emerald-700/60' };
      case 'POST':
        return { bg: 'bg-blue-950/80', text: 'text-blue-400', border: 'border-blue-700/60' };
      case 'PUT':
        return { bg: 'bg-amber-950/80', text: 'text-amber-400', border: 'border-amber-700/60' };
      case 'DELETE':
        return { bg: 'bg-rose-950/80', text: 'text-rose-400', border: 'border-rose-700/60' };
      case 'PATCH':
        return { bg: 'bg-purple-950/80', text: 'text-purple-400', border: 'border-purple-700/60' };
      default:
        return { bg: 'bg-slate-800', text: 'text-slate-300', border: 'border-slate-700' };
    }
  }

  function addParam() {
    queryParams.push({ key: '', value: '' });
  }

  function removeParam(index: number) {
    queryParams.splice(index, 1);
  }

  function addHeader() {
    headers.push({ key: '', value: '' });
  }

  function removeHeader(index: number) {
    headers.splice(index, 1);
  }

  function formatBodyJson() {
    try {
      const parsed = JSON.parse(body);
      body = JSON.stringify(parsed, null, 2);
    } catch {
      // ignore
    }
  }

  async function executeCall() {
    if (!path) return;
    isExecuting = true;
    executeError = null;
    response = null;

    const qp: Record<string, string> = {};
    for (const p of queryParams) {
      if (p.key.trim()) qp[p.key.trim()] = p.value;
    }

    const hd: Record<string, string> = {};
    for (const h of headers) {
      if (h.key.trim()) hd[h.key.trim()] = h.value;
    }

    // Attach admin token if not already specified
    const token = getToken();
    if (token && !hd['Authorization']) {
      hd['Authorization'] = `Bearer ${token}`;
    }

    try {
      const res = await api.executeWorkbenchRoute({
        method,
        path,
        query_params: qp,
        headers: hd,
        body: ['POST', 'PUT', 'PATCH'].includes(method) ? body : undefined,
      });
      response = res;
    } catch (err: any) {
      executeError = err.message || 'Execution error';
    } finally {
      isExecuting = false;
    }
  }

  function copyResponseBody() {
    if (!response) return;
    navigator.clipboard.writeText(response.body);
    copiedResponse = true;
    setTimeout(() => (copiedResponse = false), 2000);
  }

  function formatResponseBody(raw: string, isJson: boolean): string {
    if (!isJson) return raw;
    try {
      return JSON.stringify(JSON.parse(raw), null, 2);
    } catch {
      return raw;
    }
  }
</script>

<div class="flex-1 flex h-full overflow-hidden bg-[#090d16]">
  <!-- Left Column: Routes Navigator (Responsive: full width on mobile if mobileView === 'list') -->
  <div class={`w-full md:w-80 border-r border-[#212338] bg-[#111221]/60 flex flex-col flex-shrink-0 select-none ${mobileView === 'list' ? 'flex' : 'hidden md:flex'}`}>
    <div class="p-4 border-b border-[#212338] space-y-3">
      <div class="flex items-center justify-between">
        <div>
          <h2 class="text-sm font-bold text-white flex items-center gap-2">
            <span>API Routes</span>
            <span class="text-[11px] px-2 py-0.5 rounded-full bg-[#15162a] text-slate-300 font-mono border border-[#212338]">
              {routes.length}
            </span>
          </h2>
          <p class="text-[11px] text-slate-400 mt-0.5">
            Introspected from host Gin application
          </p>
        </div>

        <button
          type="button"
          onclick={() => (isCreateRouteOpen = true)}
          class="flex items-center gap-1 text-[11px] font-semibold px-2 py-0.5 rounded-lg border transition-all hover:brightness-110"
          style="background-color: var(--accent-subtle); color: var(--accent-primary); border-color: var(--accent-border);"
        >
          <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 4v16m8-8H4" />
          </svg>
          <span>Route</span>
        </button>
      </div>

      <!-- Route Search -->
      <div class="relative">
        <input
          type="text"
          placeholder="Filter routes..."
          bind:value={routeSearch}
          class="w-full pl-8 pr-3 py-1.5 rounded-xl bg-[#15162a] border border-[#212338] text-white placeholder-slate-500 text-xs focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500"
        />
        <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-2.5 text-slate-400">
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        </div>
      </div>

      <!-- Method Filter Tabs -->
      <div class="flex items-center gap-1 overflow-x-auto pb-1 text-[10px] font-mono">
        {#each ['ALL', 'GET', 'POST', 'PUT', 'DELETE'] as m}
          <button
            type="button"
            onclick={() => (methodFilter = m)}
            class={`px-2 py-0.5 rounded font-bold transition-colors ${
              methodFilter === m
                ? 'text-white'
                : 'text-slate-400 hover:text-white bg-[#15162a]'
            }`}
            style={methodFilter === m ? 'background-color: var(--accent-primary); color: #0b0c16;' : ''}
          >
            {m}
          </button>
        {/each}
      </div>
    </div>

    <!-- Routes List -->
    <div class="flex-1 overflow-y-auto divide-y divide-[#212338]/50">
      {#each filteredRoutes as r}
        {@const badge = getMethodBadge(r.method)}
        {@const isSelected = path === r.path && method === r.method}
        <button
          type="button"
          onclick={() => selectRoute(r)}
          class={`w-full p-3 text-left transition-all hover:bg-slate-800/60 block ${
            isSelected ? 'bg-indigo-950/30 border-l-2' : ''
          }`}
          style={isSelected ? 'border-color: var(--accent-primary); background-color: var(--accent-subtle);' : ''}
        >
          <div class="flex items-center gap-2 mb-1">
            <span class={`px-1.5 py-0.5 rounded text-[10px] font-mono font-bold border ${badge.bg} ${badge.text} ${badge.border}`}>
              {r.method}
            </span>
            <span class="text-xs font-mono text-slate-200 truncate flex-1 font-medium">
              {r.path}
            </span>
          </div>
          <div class="flex items-center justify-between text-[10px] text-slate-500">
            <span class="truncate">{r.handler}</span>
            {#if r.is_admin}
              <span class="text-amber-500 font-mono text-[9px]">admin</span>
            {/if}
          </div>
        </button>
      {/each}

      {#if filteredRoutes.length === 0}
        <div class="p-6 text-center text-xs text-slate-500 italic">
          No matching routes found
        </div>
      {/if}
    </div>
  </div>

  <!-- Right: Request Builder & Response Inspector (Hidden on mobile if browsing routes list) -->
  <div class={`flex-1 flex flex-col h-full overflow-hidden ${mobileView === 'list' ? 'hidden md:flex' : 'flex'}`}>
    <!-- Top Request URL Bar -->
    <div class="p-3 sm:p-4 border-b border-[#212338] bg-[#111221]/60 flex items-center gap-2 sm:gap-3 select-none">
      <!-- Mobile back to routes button -->
      <button
        type="button"
        onclick={() => (mobileView = 'list')}
        class="md:hidden px-2.5 py-2 rounded-xl bg-[#15162a] border border-[#212338] text-slate-300 text-xs flex items-center gap-1 flex-shrink-0"
        title="View routes list"
      >
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>

      <div class="relative w-24 sm:w-28 flex-shrink-0">
        <select
          bind:value={method}
          class="w-full px-3 py-2 rounded-xl bg-[#15162a] border border-[#212338] text-white font-mono font-bold text-xs focus:ring-1 focus:ring-indigo-500 appearance-none cursor-pointer"
        >
          <option value="GET">GET</option>
          <option value="POST">POST</option>
          <option value="PUT">PUT</option>
          <option value="DELETE">DELETE</option>
          <option value="PATCH">PATCH</option>
        </select>
        <div class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-2 text-slate-400">
          <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </div>

      <input
        type="text"
        bind:value={path}
        placeholder="/api/v1/resource/:id"
        class="flex-1 min-w-0 px-3 sm:px-4 py-2 rounded-xl bg-[#15162a] border border-[#212338] text-white font-mono text-xs focus:ring-1 focus:ring-indigo-500"
      />

      <button
        onclick={executeCall}
        disabled={isExecuting || !path}
        class="px-4 sm:px-5 py-2 rounded-xl font-bold text-xs shadow-lg flex items-center gap-1.5 sm:gap-2 transition-all disabled:opacity-50 flex-shrink-0 hover:opacity-90"
        style="background-color: var(--accent-primary, #a78bfa); color: #0b0c16;"
      >
        {#if isExecuting}
          <svg class="animate-spin w-4 h-4 text-slate-900" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
          </svg>
          <span class="hidden sm:inline">Executing...</span>
        {:else}
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
          <span>Send</span>
        {/if}
      </button>
    </div>

    <!-- Main Workspace Split -->
    <div class="flex-1 grid grid-cols-1 lg:grid-cols-2 divide-y lg:divide-y-0 lg:divide-x divide-[#212338] overflow-hidden">
      <!-- Request Configurator Panel -->
      <div class="flex flex-col h-full overflow-hidden bg-slate-900/20">
        <!-- Tabs -->
        <div class="flex items-center border-b border-slate-800 px-4 text-xs font-semibold select-none bg-slate-900/40">
          <button
            onclick={() => (activeTab = 'params')}
            class={`px-4 py-2.5 border-b-2 transition-colors ${
              activeTab === 'params'
                ? 'border-indigo-500 text-indigo-400'
                : 'border-transparent text-slate-400 hover:text-slate-200'
            }`}
          >
            Params ({queryParams.filter((p) => p.key).length})
          </button>
          <button
            onclick={() => (activeTab = 'headers')}
            class={`px-4 py-2.5 border-b-2 transition-colors ${
              activeTab === 'headers'
                ? 'border-indigo-500 text-indigo-400'
                : 'border-transparent text-slate-400 hover:text-slate-200'
            }`}
          >
            Headers ({headers.filter((h) => h.key).length})
          </button>
          <button
            onclick={() => (activeTab = 'body')}
            class={`px-4 py-2.5 border-b-2 transition-colors ${
              activeTab === 'body'
                ? 'border-indigo-500 text-indigo-400'
                : 'border-transparent text-slate-400 hover:text-slate-200'
            }`}
          >
            Body
          </button>
        </div>

        <!-- Tab Content -->
        <div class="flex-1 overflow-y-auto p-4">
          <!-- Params Tab -->
          {#if activeTab === 'params'}
            <div class="space-y-2">
              {#each queryParams as param, idx}
                <div class="flex items-center gap-2">
                  <input
                    type="text"
                    placeholder="Key"
                    bind:value={param.key}
                    class="flex-1 px-3 py-1.5 rounded-lg bg-slate-800 border border-slate-700 text-white font-mono text-xs"
                  />
                  <input
                    type="text"
                    placeholder="Value"
                    bind:value={param.value}
                    class="flex-1 px-3 py-1.5 rounded-lg bg-slate-800 border border-slate-700 text-white font-mono text-xs"
                  />
                  <button
                    type="button"
                    onclick={() => removeParam(idx)}
                    class="p-1.5 text-slate-500 hover:text-rose-400 transition-colors"
                  >
                    ✕
                  </button>
                </div>
              {/each}
              <button
                type="button"
                onclick={addParam}
                class="text-xs text-indigo-400 hover:text-indigo-300 font-medium mt-2 flex items-center gap-1"
              >
                + Add Parameter
              </button>
            </div>

          <!-- Headers Tab -->
          {:else if activeTab === 'headers'}
            <div class="space-y-2">
              {#each headers as header, idx}
                <div class="flex items-center gap-2">
                  <input
                    type="text"
                    placeholder="Header"
                    bind:value={header.key}
                    class="flex-1 px-3 py-1.5 rounded-lg bg-slate-800 border border-slate-700 text-white font-mono text-xs"
                  />
                  <input
                    type="text"
                    placeholder="Value"
                    bind:value={header.value}
                    class="flex-1 px-3 py-1.5 rounded-lg bg-slate-800 border border-slate-700 text-white font-mono text-xs"
                  />
                  <button
                    type="button"
                    onclick={() => removeHeader(idx)}
                    class="p-1.5 text-slate-500 hover:text-rose-400 transition-colors"
                  >
                    ✕
                  </button>
                </div>
              {/each}
              <button
                type="button"
                onclick={addHeader}
                class="text-xs text-indigo-400 hover:text-indigo-300 font-medium mt-2 flex items-center gap-1"
              >
                + Add Header
              </button>
            </div>

          <!-- Body Tab -->
          {:else if activeTab === 'body'}
            <div class="h-full flex flex-col space-y-2">
              <div class="flex justify-between items-center text-xs text-slate-400">
                <span>JSON Payload</span>
                <button
                  type="button"
                  onclick={formatBodyJson}
                  class="text-indigo-400 hover:underline font-medium"
                >
                  Format JSON
                </button>
              </div>
              <textarea
                bind:value={body}
                rows="14"
                class="w-full flex-1 p-3 rounded-xl bg-slate-800/90 border border-slate-700 text-white font-mono text-xs focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 resize-none"
                placeholder={`{\n  "name": "example"\n}`}
              ></textarea>
            </div>
          {/if}
        </div>
      </div>

      <!-- Live Response Inspector Panel -->
      <div class="flex flex-col h-full overflow-hidden bg-slate-900/40">
        <!-- Response Header Bar -->
        <div class="p-3.5 border-b border-slate-800 flex items-center justify-between text-xs select-none">
          <div class="flex items-center gap-3">
            <span class="font-bold text-slate-300 uppercase tracking-wider text-[11px]">Response</span>
            {#if response}
              <span class={`px-2 py-0.5 rounded-full font-bold font-mono text-[10px] ${
                response.status_code >= 200 && response.status_code < 300
                  ? 'bg-emerald-950 text-emerald-400 border border-emerald-800/60'
                  : 'bg-rose-950 text-rose-400 border border-rose-800/60'
              }`}>
                {response.status_code} {response.status_text}
              </span>
              <span class="text-slate-400 font-mono text-[11px]">
                {response.duration_ms.toFixed(1)} ms
              </span>
            {/if}
          </div>

          {#if response}
            <div class="flex items-center gap-2">
              <button
                type="button"
                onclick={copyResponseBody}
                class="px-2.5 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white font-medium text-[11px] flex items-center gap-1 transition-colors"
              >
                {#if copiedResponse}
                  <span class="text-emerald-400">Copied!</span>
                {:else}
                  <span>Copy</span>
                {/if}
              </button>
            </div>
          {/if}
        </div>

        <!-- Response Body Area -->
        <div class="flex-1 overflow-y-auto p-4 font-mono text-xs">
          {#if executeError}
            <div class="p-4 rounded-xl bg-rose-950/70 border border-rose-800 text-rose-300">
              {executeError}
            </div>
          {:else if response}
            <pre class="text-slate-200 select-all whitespace-pre-wrap">{formatResponseBody(response.body, response.is_json)}</pre>
          {:else}
            <div class="h-full flex flex-col items-center justify-center text-slate-500 text-center">
              <svg class="w-8 h-8 mb-2 text-slate-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
              <span>Click "Send" to execute this request and inspect response</span>
            </div>
          {/if}
        </div>
      </div>
    </div>
  </div>

  <CreateRouteModal
    isOpen={isCreateRouteOpen}
    onClose={() => (isCreateRouteOpen = false)}
    onCreated={loadRoutes}
  />
</div>
