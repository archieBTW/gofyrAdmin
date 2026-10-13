<script lang="ts">
  let { src = '', alt = '', onClose = () => {} } = $props<{
    src: string;
    alt?: string;
    onClose: () => void;
  }>();

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onClose();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if src}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm p-4 transition-all"
    onclick={onClose}
    onkeydown={handleKeydown}
    role="dialog"
    aria-modal="true"
    tabindex="-1"
  >
    <div
      class="relative max-w-4xl max-h-[90vh] bg-slate-900 border border-slate-700/60 rounded-xl overflow-hidden shadow-2xl p-2"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.stopPropagation()}
      role="document"
      tabindex="0"
    >
      <button
        type="button"
        onclick={onClose}
        class="absolute top-4 right-4 z-10 p-2 rounded-full bg-slate-800/80 hover:bg-slate-700 text-slate-300 hover:text-white transition-colors cursor-pointer"
        title="Close preview"
      >
        <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>

      <img
        {src}
        {alt}
        class="max-h-[80vh] w-auto max-w-full rounded-lg object-contain mx-auto"
      />

      <div class="mt-2 px-2 py-1 flex items-center justify-between text-xs text-slate-400">
        <span class="truncate max-w-md">{alt || src}</span>
        <a
          href={src}
          target="_blank"
          rel="noopener noreferrer"
          class="text-indigo-400 hover:text-indigo-300 underline font-medium ml-4"
        >
          Open original
        </a>
      </div>
    </div>
  </div>
{/if}
