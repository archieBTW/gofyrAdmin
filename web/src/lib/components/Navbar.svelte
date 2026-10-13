<script lang="ts">
  import type { AdminUser, DatasourceSchema } from '../types';

  let {
    user = null as AdminUser | null,
    datasources = [] as DatasourceSchema[],
    activeDatasource = '',
    defaultDisk = 'local',
    onSelectDatasource = (ds: string) => {},
    onLogout = () => {},
    onToggleMobileMenu = () => {},
    onOpenThemeModal = () => {},
  } = $props<{
    user: AdminUser | null;
    datasources: DatasourceSchema[];
    activeDatasource: string;
    defaultDisk: string;
    onSelectDatasource: (ds: string) => void;
    onLogout: () => void;
    onToggleMobileMenu: () => void;
    onOpenThemeModal: () => void;
  }>();

  let isUserMenuOpen = $state(false);

  function getDialectBadge(dialect: string) {
    const d = dialect.toLowerCase();
    if (d.includes('postgres') || d.includes('pg')) {
      return { label: 'PostgreSQL', bg: 'bg-blue-950/80', text: 'text-blue-400', border: 'border-blue-700/60' };
    }
    if (d.includes('mysql') || d.includes('mariadb')) {
      return { label: 'MySQL', bg: 'bg-amber-950/80', text: 'text-amber-400', border: 'border-amber-700/60' };
    }
    if (d.includes('sqlite')) {
      return { label: 'SQLite', bg: 'bg-cyan-950/80', text: 'text-cyan-400', border: 'border-cyan-700/60' };
    }
    return { label: dialect, bg: 'bg-slate-800', text: 'text-slate-300', border: 'border-slate-700' };
  }

  const currentDS = $derived(datasources.find((d) => d.name === activeDatasource));
</script>

<header class="h-16 bg-[#111221]/90 backdrop-blur-md border-b border-[#212338] flex items-center justify-between px-4 sm:px-6 sticky top-0 z-30 select-none">
  <!-- Left: Hamburger (mobile) + Brand + Datasource Switcher -->
  <div class="flex items-center gap-3 sm:gap-6">
    <!-- Mobile Hamburger Toggle -->
    <button
      type="button"
      onclick={onToggleMobileMenu}
      class="p-2 -ml-2 rounded-xl text-slate-400 hover:text-white hover:bg-slate-800/80 md:hidden transition-colors"
      aria-label="Toggle navigation menu"
    >
      <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
      </svg>
    </button>

    <!-- Typographic Logo (Icon removed) -->
    <div class="flex items-center">
      <span class="text-base sm:text-lg font-black tracking-tight text-white font-mono">
        gofyr<span style="color: var(--accent-primary, #a78bfa);">Admin</span>
      </span>
    </div>

    <!-- Datasource Switcher -->
    {#if datasources.length > 0}
      <div class="hidden sm:flex items-center gap-2 pl-4 border-l border-[#212338]">
        <span class="text-[11px] font-semibold uppercase text-slate-500">Datasource:</span>
        <div class="relative">
          <select
            value={activeDatasource}
            onchange={(e) => onSelectDatasource((e.target as HTMLSelectElement).value)}
            class="pl-3 pr-8 py-1.5 rounded-lg bg-[#15162a] border border-[#212338] text-white font-medium text-xs focus:ring-1 focus:ring-indigo-500 appearance-none cursor-pointer"
          >
            {#each datasources as ds}
              <option value={ds.name}>
                {ds.name} {ds.is_primary ? '(Primary)' : ''}
              </option>
            {/each}
          </select>
          <div class="pointer-events-none absolute inset-y-0 right-0 flex items-center px-2 text-slate-400">
            <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
            </svg>
          </div>
        </div>

        {#if currentDS}
          {@const badge = getDialectBadge(currentDS.dialect)}
          <span class={`hidden lg:inline-block text-[11px] font-semibold px-2 py-0.5 rounded-full border ${badge.bg} ${badge.text} ${badge.border}`}>
            {badge.label}
          </span>
        {/if}
      </div>
    {/if}
  </div>

  <!-- Right: Theme Customizer + Storage Indicator + User Menu -->
  <div class="flex items-center gap-2 sm:gap-4">
    <!-- Theme Settings Button -->
    <button
      type="button"
      onclick={onOpenThemeModal}
      title="Customize Theme & Appearance"
      class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-xl border border-[#212338] bg-[#15162a] hover:bg-slate-800 text-xs text-slate-300 hover:text-white transition-colors"
    >
      <span
        class="w-3 h-3 rounded-full flex-shrink-0"
        style="background-color: var(--accent-primary, #a78bfa); box-shadow: 0 0 8px var(--accent-glow);"
      ></span>
      <span class="hidden md:inline text-[11px] font-medium">Theme</span>
    </button>

    <!-- Storage Badge -->
    <div class="hidden sm:flex items-center gap-2 px-3 py-1 rounded-full bg-[#15162a] border border-[#212338] text-xs">
      <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
      <span class="text-slate-400 font-medium">Storage:</span>
      <span class="text-slate-200 font-mono uppercase text-[11px] font-bold">{defaultDisk}</span>
    </div>

    <!-- User Profile Dropdown -->
    {#if user}
      <div class="relative">
        <button
          onclick={() => (isUserMenuOpen = !isUserMenuOpen)}
          class="flex items-center gap-2 p-1.5 rounded-xl hover:bg-slate-800 text-left transition-colors"
        >
          <div
            class="w-8 h-8 rounded-lg font-bold text-xs flex items-center justify-center uppercase border border-[#212338]"
            style="background-color: var(--accent-subtle, rgba(167, 139, 250, 0.15)); color: var(--accent-primary, #a78bfa);"
          >
            {user.name ? user.name[0] : user.email[0]}
          </div>
          <div class="hidden md:block">
            <div class="text-xs font-semibold text-white leading-tight">{user.name || user.email}</div>
            <div class="text-[10px] text-slate-400 font-mono">{user.role}</div>
          </div>
          <svg class="w-3.5 h-3.5 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {#if isUserMenuOpen}
          <div
            class="absolute right-0 mt-2 w-56 rounded-xl bg-[#111221] border border-[#212338] shadow-2xl py-2 z-50 animate-in fade-in zoom-in-95 duration-100"
          >
            <div class="px-4 py-2 border-b border-[#212338]">
              <p class="text-xs font-semibold text-white truncate">{user.name || 'Admin User'}</p>
              <p class="text-[11px] text-slate-400 truncate">{user.email}</p>
              <span
                class="inline-block mt-1 text-[10px] uppercase font-bold px-2 py-0.5 rounded border border-[#212338]"
                style="background-color: var(--accent-subtle); color: var(--accent-primary);"
              >
                {user.role}
              </span>
            </div>

            <!-- Mobile: Datasource Selector inside user menu if on small screen -->
            {#if datasources.length > 0}
              <div class="sm:hidden px-4 py-2 border-b border-[#212338]">
                <label class="block text-[10px] font-bold uppercase text-slate-500 mb-1">Datasource</label>
                <select
                  value={activeDatasource}
                  onchange={(e) => onSelectDatasource((e.target as HTMLSelectElement).value)}
                  class="w-full px-2 py-1 rounded bg-[#15162a] border border-[#212338] text-white text-xs"
                >
                  {#each datasources as ds}
                    <option value={ds.name}>{ds.name}</option>
                  {/each}
                </select>
              </div>
            {/if}

            <button
              onclick={() => {
                isUserMenuOpen = false;
                onOpenThemeModal();
              }}
              class="w-full px-4 py-2 text-left text-xs text-slate-300 hover:text-white hover:bg-slate-800 flex items-center gap-2 transition-colors"
            >
              <svg class="w-4 h-4 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21a4 4 0 01-4-4V5a2 2 0 012-2h4a2 2 0 012 2v12a4 4 0 01-4 4zm0 0h12a2 2 0 002-2v-4a2 2 0 00-2-2h-2.343M11 7.343l1.657-1.657a2 2 0 012.828 0l2.829 2.829a2 2 0 010 2.828l-8.486 8.485M7 17h.01" />
              </svg>
              <span>Appearance & Theme</span>
            </button>

            <button
              onclick={() => {
                isUserMenuOpen = false;
                onLogout();
              }}
              class="w-full px-4 py-2 text-left text-xs text-rose-400 hover:text-rose-300 hover:bg-rose-950/30 flex items-center gap-2 transition-colors mt-1"
            >
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
              </svg>
              <span>Sign Out</span>
            </button>
          </div>
        {/if}
      </div>
    {/if}
  </div>
</header>
