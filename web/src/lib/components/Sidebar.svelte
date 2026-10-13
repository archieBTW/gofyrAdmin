<script lang="ts">
  import type { DatasourceSchema, ModelInfo } from '../types';

  let {
    datasources = [] as DatasourceSchema[],
    activeDatasource = '',
    currentView = 'model', // 'model' | 'media' | 'workbench' | 'audit'
    activeModel = '',
    onSelectView = (view: string, modelName?: string) => {},
    onOpenCreateModel = () => {},
    onCloseMobileMenu = () => {},
  } = $props<{
    datasources: DatasourceSchema[];
    activeDatasource: string;
    currentView: string;
    activeModel: string;
    onSelectView: (view: string, modelName?: string) => void;
    onOpenCreateModel: () => void;
    onCloseMobileMenu?: () => void;
  }>();

  const currentDS = $derived(datasources.find((d) => d.name === activeDatasource));
  const models = $derived(currentDS?.models || []);

  function handleNavigate(view: string, modelName?: string) {
    onSelectView(view, modelName);
    onCloseMobileMenu();
  }
</script>

<aside class="w-64 bg-[#111221] border-r border-[#212338] flex flex-col flex-shrink-0 select-none overflow-y-auto h-full">
  <div class="p-4 space-y-6 flex-1">
    <!-- Models Section -->
    <div>
      <div class="px-2 mb-2 flex items-center justify-between text-[11px] font-bold text-slate-400 uppercase tracking-wider">
        <div class="flex items-center gap-1.5">
          <span>GORM Models</span>
          <span class="text-xs px-1.5 py-0.2 rounded bg-[#15162a] text-slate-400 font-mono border border-[#212338]">
            {models.length}
          </span>
        </div>

        <!-- + New Model Button -->
        <button
          type="button"
          onclick={() => {
            onOpenCreateModel();
            onCloseMobileMenu();
          }}
          title="Create New Model"
          class="flex items-center gap-1 text-[11px] font-semibold px-2 py-0.5 rounded-lg border transition-colors hover:brightness-110"
          style="background-color: var(--accent-subtle); color: var(--accent-primary); border-color: var(--accent-border);"
        >
          <svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 4v16m8-8H4" />
          </svg>
          <span>Model</span>
        </button>
      </div>

      <nav class="space-y-1">
        {#each models as model}
          {@const isActive = currentView === 'model' && activeModel === model.name}
          <button
            onclick={() => handleNavigate('model', model.name)}
            class={`w-full flex items-center justify-between px-3 py-2 rounded-xl text-xs font-medium transition-all ${
              isActive
                ? 'font-semibold'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
            }`}
            style={isActive ? 'background-color: var(--accent-subtle); color: var(--accent-primary); border: 1px solid var(--accent-border);' : ''}
          >
            <div class="flex items-center gap-2.5 truncate">
              <svg class={`w-4 h-4 flex-shrink-0 ${isActive ? '' : 'text-slate-500'}`} style={isActive ? 'color: var(--accent-primary);' : ''} fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h18M3 14h18m-9-4v8m-7 0h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
              <span class="truncate">{model.name}</span>
            </div>
            <span class="text-[10px] text-slate-500 font-mono">
              {model.table_name}
            </span>
          </button>
        {/each}

        {#if models.length === 0}
          <div class="px-3 py-4 text-xs text-slate-500 italic text-center">
            No models found. Click "+ Model" above to create one.
          </div>
        {/if}
      </nav>
    </div>

    <!-- Platform Modules Section -->
    <div>
      <div class="px-2 mb-2 text-[11px] font-bold text-slate-400 uppercase tracking-wider">
        <span>Workbench & Tools</span>
      </div>

      <nav class="space-y-1">
        <!-- Media Library -->
        <button
          onclick={() => handleNavigate('media')}
          class={`w-full flex items-center gap-2.5 px-3 py-2 rounded-xl text-xs font-medium transition-all ${
            currentView === 'media'
              ? 'font-semibold'
              : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
          }`}
          style={currentView === 'media' ? 'background-color: var(--accent-subtle); color: var(--accent-primary); border: 1px solid var(--accent-border);' : ''}
        >
          <svg class={`w-4 h-4 flex-shrink-0 ${currentView === 'media' ? '' : 'text-slate-500'}`} style={currentView === 'media' ? 'color: var(--accent-primary);' : ''} fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <span>Media & Storage</span>
        </button>

        <!-- API Workbench -->
        <button
          onclick={() => handleNavigate('workbench')}
          class={`w-full flex items-center gap-2.5 px-3 py-2 rounded-xl text-xs font-medium transition-all ${
            currentView === 'workbench'
              ? 'font-semibold'
              : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
          }`}
          style={currentView === 'workbench' ? 'background-color: var(--accent-subtle); color: var(--accent-primary); border: 1px solid var(--accent-border);' : ''}
        >
          <svg class={`w-4 h-4 flex-shrink-0 ${currentView === 'workbench' ? '' : 'text-slate-500'}`} style={currentView === 'workbench' ? 'color: var(--accent-primary);' : ''} fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
          </svg>
          <span>API Workbench</span>
        </button>

        <!-- Audit Logs -->
        <button
          onclick={() => handleNavigate('audit')}
          class={`w-full flex items-center gap-2.5 px-3 py-2 rounded-xl text-xs font-medium transition-all ${
            currentView === 'audit'
              ? 'font-semibold'
              : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/60'
          }`}
          style={currentView === 'audit' ? 'background-color: var(--accent-subtle); color: var(--accent-primary); border: 1px solid var(--accent-border);' : ''}
        >
          <svg class={`w-4 h-4 flex-shrink-0 ${currentView === 'audit' ? '' : 'text-slate-500'}`} style={currentView === 'audit' ? 'color: var(--accent-primary);' : ''} fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <span>Audit Logs</span>
        </button>
      </nav>
    </div>
  </div>
</aside>
