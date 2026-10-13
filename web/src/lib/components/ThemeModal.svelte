<script lang="ts">
  import {
    ACCENT_PALETTES,
    type AccentTheme,
    type Density,
    type ThemePreferences,
    loadThemePreferences,
    saveThemePreferences,
  } from '../theme';

  let {
    isOpen = false,
    onClose = () => {},
  } = $props<{
    isOpen: boolean;
    onClose: () => void;
  }>();

  let prefs = $state<ThemePreferences>({
    accent: 'lavender',
    density: 'comfortable',
  });

  $effect(() => {
    if (isOpen) {
      prefs = loadThemePreferences();
    }
  });

  function selectAccent(accent: AccentTheme) {
    prefs.accent = accent;
    saveThemePreferences(prefs);
  }

  function selectDensity(density: Density) {
    prefs.density = density;
    saveThemePreferences(prefs);
  }

  const accents: AccentTheme[] = ['lavender', 'violet', 'indigo', 'rose', 'emerald', 'amber', 'cyan'];
</script>

{#if isOpen}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 backdrop-blur-sm p-4 overflow-y-auto"
    onclick={onClose}
    onkeydown={(e) => e.key === 'Escape' && onClose()}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
  >
    <div
      class="bg-[#111221] border border-[#212338] rounded-2xl p-6 max-w-md w-full shadow-2xl relative"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.stopPropagation()}
      role="document"
      tabindex="0"
    >
      <div class="flex items-center justify-between pb-4 border-b border-[#212338]">
        <div>
          <h2 class="text-base font-bold text-white">Appearance & Theme</h2>
          <p class="text-xs text-slate-400 mt-0.5">Customize your personal admin dashboard styling</p>
        </div>
        <button
          onclick={onClose}
          class="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
        >
          <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Accent Color Selection -->
      <div class="py-5 border-b border-[#212338] space-y-3">
        <label class="block text-xs font-semibold uppercase tracking-wider text-slate-300">
          Accent Color
        </label>
        <div class="grid grid-cols-2 gap-2.5">
          {#each accents as acc}
            {@const palette = ACCENT_PALETTES[acc]}
            {@const isSelected = prefs.accent === acc}
            <button
              type="button"
              onclick={() => selectAccent(acc)}
              class={`flex items-center gap-3 p-2.5 rounded-xl border text-left transition-all ${
                isSelected
                  ? 'border-white/40 bg-white/5 ring-1 ring-white/30'
                  : 'border-[#212338] hover:border-slate-700 hover:bg-slate-800/40'
              }`}
            >
              <span
                class="w-5 h-5 rounded-full flex-shrink-0 shadow-sm border border-white/20"
                style={`background-color: ${palette.primary};`}
              ></span>
              <div class="flex-1 min-w-0">
                <span class="text-xs font-medium text-white block truncate">
                  {palette.label}
                </span>
                {#if acc === 'lavender'}
                  <span class="text-[10px] text-slate-400">Default</span>
                {/if}
              </div>
              {#if isSelected}
                <svg class="w-4 h-4 text-white flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                </svg>
              {/if}
            </button>
          {/each}
        </div>
      </div>

      <!-- Density Selection -->
      <div class="py-5 border-b border-[#212338] space-y-3">
        <label class="block text-xs font-semibold uppercase tracking-wider text-slate-300">
          Display Density
        </label>
        <div class="grid grid-cols-2 gap-2.5">
          <button
            type="button"
            onclick={() => selectDensity('comfortable')}
            class={`p-3 rounded-xl border text-left transition-all ${
              prefs.density === 'comfortable'
                ? 'border-white/40 bg-white/5 ring-1 ring-white/30'
                : 'border-[#212338] hover:border-slate-700 hover:bg-slate-800/40'
            }`}
          >
            <div class="text-xs font-semibold text-white">Comfortable</div>
            <div class="text-[11px] text-slate-400 mt-0.5">Spacious touch targets & standard padding</div>
          </button>
          <button
            type="button"
            onclick={() => selectDensity('compact')}
            class={`p-3 rounded-xl border text-left transition-all ${
              prefs.density === 'compact'
                ? 'border-white/40 bg-white/5 ring-1 ring-white/30'
                : 'border-[#212338] hover:border-slate-700 hover:bg-slate-800/40'
            }`}
          >
            <div class="text-xs font-semibold text-white">Compact</div>
            <div class="text-[11px] text-slate-400 mt-0.5">Higher density for large datasets</div>
          </button>
        </div>
      </div>

      <div class="pt-4 flex justify-end">
        <button
          onclick={onClose}
          class="px-4 py-2 rounded-xl text-xs font-semibold text-white transition-opacity hover:opacity-90 shadow-md"
          style="background-color: var(--accent-primary); color: #0b0c16;"
        >
          Done
        </button>
      </div>
    </div>
  </div>
{/if}
