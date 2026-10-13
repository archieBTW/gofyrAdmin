<script lang="ts">
  import { api } from '../api';
  import type { AuditLog } from '../types';

  let logs = $state<AuditLog[]>([]);
  let total = $state(0);
  let totalPages = $state(1);
  let page = $state(1);
  let limit = $state(25);
  let search = $state('');
  let isLoading = $state(false);

  $effect(() => {
    loadLogs();
  });

  async function loadLogs() {
    isLoading = true;
    try {
      const res = await api.listAuditLogs({
        page,
        limit,
        search,
      });
      logs = res.data || [];
      total = res.total;
      totalPages = res.total_pages || 1;
    } catch (err: any) {
      console.error('Failed to load audit logs:', err);
    } finally {
      isLoading = false;
    }
  }

  function handleSearch(e: Event) {
    search = (e.target as HTMLInputElement).value;
    page = 1;
    loadLogs();
  }

  function getActionBadge(action: string) {
    if (action.includes('login') || action.includes('mfa')) {
      return { bg: 'bg-emerald-950/80', text: 'text-emerald-400', border: 'border-emerald-800/60' };
    }
    if (action.includes('delete')) {
      return { bg: 'bg-rose-950/80', text: 'text-rose-400', border: 'border-rose-800/60' };
    }
    if (action.includes('create')) {
      return { bg: 'bg-blue-950/80', text: 'text-blue-400', border: 'border-blue-800/60' };
    }
    if (action.includes('update')) {
      return { bg: 'bg-amber-950/80', text: 'text-amber-400', border: 'border-amber-800/60' };
    }
    return { bg: 'bg-slate-800', text: 'text-slate-300', border: 'border-slate-700' };
  }
</script>

<div class="flex-1 flex flex-col h-full overflow-hidden bg-[#090d16]">
  <!-- Header -->
  <div class="px-6 py-4 border-b border-slate-800 bg-slate-900/40 flex flex-col md:flex-row md:items-center justify-between gap-4 select-none">
    <div>
      <h1 class="text-xl font-bold text-white tracking-tight flex items-center gap-2">
        <span>Audit Logs</span>
        <span class="text-xs px-2.5 py-0.5 rounded-full bg-slate-800 text-slate-300 font-mono border border-slate-700">
          {total} events
        </span>
      </h1>
      <p class="text-xs text-slate-400 mt-0.5">
        Security activity, MFA logins, and datasource mutation records
      </p>
    </div>

    <!-- Search -->
    <div class="relative w-64">
      <input
        type="text"
        placeholder="Search audit trail..."
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
  </div>

  <!-- Table -->
  <div class="flex-1 overflow-auto">
    <table class="w-full text-left border-collapse">
      <thead class="sticky top-0 bg-slate-900/95 backdrop-blur border-b border-slate-800 text-[11px] font-bold text-slate-400 uppercase tracking-wider select-none z-10">
        <tr>
          <th class="px-6 py-3">Timestamp</th>
          <th class="px-6 py-3">User</th>
          <th class="px-6 py-3">Action</th>
          <th class="px-6 py-3">Target</th>
          <th class="px-6 py-3">IP Address</th>
          <th class="px-6 py-3">Details</th>
        </tr>
      </thead>

      <tbody class="divide-y divide-slate-800/60 text-xs">
        {#each logs as log}
          {@const badge = getActionBadge(log.action)}
          <tr class="hover:bg-slate-800/30 transition-colors">
            <td class="px-6 py-3 text-slate-400 font-mono text-[11px] whitespace-nowrap">
              {log.created_at ? log.created_at.substring(0, 19).replace('T', ' ') : '-'}
            </td>
            <td class="px-6 py-3 text-white font-medium">
              {log.user_email || 'System'}
            </td>
            <td class="px-6 py-3">
              <span class={`inline-block px-2 py-0.5 rounded-full font-mono text-[10px] font-bold border ${badge.bg} ${badge.text} ${badge.border}`}>
                {log.action}
              </span>
            </td>
            <td class="px-6 py-3 text-slate-300 font-mono text-[11px]">
              {#if log.model}
                {log.datasource ? `${log.datasource} / ` : ''}{log.model} {log.record_id ? `#${log.record_id}` : ''}
              {:else}
                -
              {/if}
            </td>
            <td class="px-6 py-3 text-slate-400 font-mono text-[11px]">
              {log.ip_address || '-'}
            </td>
            <td class="px-6 py-3 text-slate-300 max-w-sm truncate" title={log.details}>
              {log.details || '-'}
            </td>
          </tr>
        {/each}

        {#if logs.length === 0 && !isLoading}
          <tr>
            <td colspan="6" class="py-12 text-center text-slate-500 text-xs">
              No audit records recorded yet.
            </td>
          </tr>
        {/if}
      </tbody>
    </table>
  </div>

  <!-- Pagination -->
  <div class="px-6 py-3 border-t border-slate-800 bg-slate-900/60 flex items-center justify-between text-xs select-none">
    <div class="text-slate-400">
      Page <span class="font-bold text-white">{page}</span> of <span class="font-bold text-white">{totalPages}</span>
    </div>

    <div class="flex items-center gap-2">
      <button
        onclick={() => { page = Math.max(1, page - 1); loadLogs(); }}
        disabled={page <= 1 || isLoading}
        class="px-3 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 disabled:opacity-40"
      >
        Previous
      </button>

      <button
        onclick={() => { page = Math.min(totalPages, page + 1); loadLogs(); }}
        disabled={page >= totalPages || isLoading}
        class="px-3 py-1 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 disabled:opacity-40"
      >
        Next
      </button>
    </div>
  </div>
</div>
