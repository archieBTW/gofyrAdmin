<script lang="ts">
  import { api, getToken, removeToken } from './lib/api';
  import type { AdminUser, SchemaResponse, ModelInfo } from './lib/types';
  import { applyTheme, loadThemePreferences } from './lib/theme';
  import AuthView from './lib/components/AuthView.svelte';
  import Navbar from './lib/components/Navbar.svelte';
  import Sidebar from './lib/components/Sidebar.svelte';
  import ModelExplorer from './lib/components/ModelExplorer.svelte';
  import MediaLibrary from './lib/components/MediaLibrary.svelte';
  import Workbench from './lib/components/Workbench.svelte';
  import AuditLogs from './lib/components/AuditLogs.svelte';
  import Lightbox from './lib/components/Lightbox.svelte';
  import ThemeModal from './lib/components/ThemeModal.svelte';
  import CreateModelModal from './lib/components/CreateModelModal.svelte';

  let user = $state<AdminUser | null>(null);
  let isCheckingAuth = $state(true);
  let schema = $state<SchemaResponse | null>(null);

  let activeDatasource = $state('');
  let currentView = $state<'model' | 'media' | 'workbench' | 'audit'>('model');
  let activeModel = $state('');
  let previewImage = $state<string | null>(null);

  let isMobileMenuOpen = $state(false);
  let isThemeModalOpen = $state(false);
  let isCreateModelOpen = $state(false);

  $effect(() => {
    applyTheme(loadThemePreferences());
    checkAuthAndInit();
  });

  async function checkAuthAndInit() {
    isCheckingAuth = true;
    const token = getToken();
    if (!token) {
      isCheckingAuth = false;
      return;
    }

    try {
      const res = await api.getMe();
      user = res.user;
      await loadSchema();
    } catch {
      removeToken();
      user = null;
    } finally {
      isCheckingAuth = false;
    }
  }

  async function loadSchema() {
    try {
      const res = await api.getSchema();
      schema = res;

      if (res.datasources.length > 0 && !activeDatasource) {
        // Prefer primary datasource
        const primary = res.datasources.find((d) => d.is_primary) || res.datasources[0];
        activeDatasource = primary.name;
        if (primary.models.length > 0 && !activeModel) {
          activeModel = primary.models[0].name;
        }
      }
    } catch (err) {
      console.error('Failed to load schema:', err);
    }
  }

  function handleLoginSuccess(authenticatedUser: AdminUser) {
    user = authenticatedUser;
    loadSchema();
  }

  function handleLogout() {
    api.logout();
    user = null;
    schema = null;
  }

  function handleSelectDatasource(dsName: string) {
    activeDatasource = dsName;
    const ds = schema?.datasources.find((d) => d.name === dsName);
    if (ds && ds.models.length > 0) {
      activeModel = ds.models[0].name;
      currentView = 'model';
    }
  }

  function handleSelectView(view: string, modelName?: string) {
    currentView = view as any;
    if (modelName) {
      activeModel = modelName;
    }
  }

  const currentModelInfo = $derived(
    schema?.datasources
      .find((d) => d.name === activeDatasource)
      ?.models.find((m) => m.name === activeModel) || null
  );
</script>

{#if isCheckingAuth}
  <div class="min-h-screen flex items-center justify-center bg-[#0b0c16] text-slate-400">
    <div class="flex flex-col items-center gap-3">
      <div
        class="w-10 h-10 rounded-2xl border-2 border-t-transparent animate-spin flex items-center justify-center"
        style="border-color: var(--accent-primary, #a78bfa); border-top-color: transparent;"
      ></div>
      <span class="text-xs font-mono text-slate-300">Initializing gofyrAdmin...</span>
    </div>
  </div>
{:else if !user}
  <AuthView onSuccess={handleLoginSuccess} />
{:else}
  <div class="min-h-screen flex flex-col bg-[#0b0c16] text-slate-100">
    <!-- Top Navigation Bar -->
    <Navbar
      {user}
      datasources={schema?.datasources || []}
      {activeDatasource}
      defaultDisk={schema?.default_disk || 'local'}
      onSelectDatasource={handleSelectDatasource}
      onLogout={handleLogout}
      onToggleMobileMenu={() => (isMobileMenuOpen = !isMobileMenuOpen)}
      onOpenThemeModal={() => (isThemeModalOpen = true)}
    />

    <!-- Main Shell -->
    <div class="flex-1 flex overflow-hidden relative">
      <!-- Desktop Left Sidebar -->
      <div class="hidden md:flex flex-shrink-0 h-full">
        <Sidebar
          datasources={schema?.datasources || []}
          {activeDatasource}
          {currentView}
          {activeModel}
          onSelectView={handleSelectView}
          onOpenCreateModel={() => (isCreateModelOpen = true)}
        />
      </div>

      <!-- Mobile Left Sidebar Drawer -->
      {#if isMobileMenuOpen}
        <div class="fixed inset-0 z-40 md:hidden flex">
          <!-- Backdrop -->
          <div
            class="fixed inset-0 bg-black/70 backdrop-blur-sm transition-opacity"
            onclick={() => (isMobileMenuOpen = false)}
            onkeydown={(e) => e.key === 'Escape' && (isMobileMenuOpen = false)}
            role="button"
            tabindex="0"
            aria-label="Close menu"
          ></div>
          <!-- Drawer Sheet -->
          <div class="relative w-64 max-w-[80vw] h-full shadow-2xl z-50 flex flex-col">
            <Sidebar
              datasources={schema?.datasources || []}
              {activeDatasource}
              {currentView}
              {activeModel}
              onSelectView={handleSelectView}
              onOpenCreateModel={() => (isCreateModelOpen = true)}
              onCloseMobileMenu={() => (isMobileMenuOpen = false)}
            />
          </div>
        </div>
      {/if}

      <!-- Main Center Content Area -->
      <main class="flex-1 flex flex-col overflow-hidden min-w-0">
        {#if currentView === 'model'}
          <ModelExplorer
            datasource={activeDatasource}
            model={currentModelInfo}
            onPreviewImage={(url) => (previewImage = url)}
          />
        {:else if currentView === 'media'}
          <MediaLibrary
            disks={schema?.disks || ['local']}
            defaultDisk={schema?.default_disk || 'local'}
            onPreviewImage={(url) => (previewImage = url)}
          />
        {:else if currentView === 'workbench'}
          <Workbench />
        {:else if currentView === 'audit'}
          <AuditLogs />
        {/if}
      </main>
    </div>

    <!-- Image Lightbox Modal -->
    {#if previewImage}
      <Lightbox src={previewImage} onClose={() => (previewImage = null)} />
    {/if}

    <!-- Appearance & Theme Customizer Modal -->
    <ThemeModal
      isOpen={isThemeModalOpen}
      onClose={() => (isThemeModalOpen = false)}
    />

    <!-- Dynamic Model Creation Modal -->
    <CreateModelModal
      isOpen={isCreateModelOpen}
      datasources={schema?.datasources || []}
      {activeDatasource}
      onClose={() => (isCreateModelOpen = false)}
      onCreated={loadSchema}
    />
  </div>
{/if}
