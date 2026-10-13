<script lang="ts">
  import { api } from '../api';

  let {
    isOpen = false,
    onClose = () => {},
    onCreated = () => {},
  } = $props<{
    isOpen: boolean;
    onClose: () => void;
    onCreated: () => void;
  }>();

  let method = $state('GET');
  let path = $state('/api/v1/');
  let statusCode = $state(200);
  let description = $state('');
  let responseBody = $state('{\n  "message": "Custom mock response",\n  "status": "success"\n}');
  let isSubmitting = $state(false);
  let errorMessage = $state<string | null>(null);

  async function handleSubmit(e: Event) {
    e.preventDefault();
    if (!path.trim()) return;

    isSubmitting = true;
    errorMessage = null;

    try {
      await api.createCustomRoute({
        method,
        path: path.trim(),
        status_code: Number(statusCode) || 200,
        response_body: responseBody,
        description: description.trim(),
      });
      onCreated();
      onClose();
    } catch (err: any) {
      errorMessage = err.message || 'Failed to create route';
    } finally {
      isSubmitting = false;
    }
  }
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
      class="bg-[#111221] border border-[#212338] rounded-2xl p-6 max-w-xl w-full shadow-2xl my-8 relative"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.stopPropagation()}
      role="document"
      tabindex="0"
    >
      <div class="flex items-center justify-between pb-4 border-b border-[#212338]">
        <div>
          <h2 class="text-base font-bold text-white flex items-center gap-2">
            <span>Create Custom Route</span>
            <span class="text-[11px] px-2 py-0.5 rounded bg-violet-950/80 text-violet-300 font-mono border border-violet-800/60">
              API Mock
            </span>
          </h2>
          <p class="text-xs text-slate-400 mt-0.5">
            Register and mount a dynamic mock endpoint onto the host Gin engine
          </p>
        </div>

        <button
          type="button"
          onclick={onClose}
          class="p-1 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800"
        >
          ✕
        </button>
      </div>

      {#if errorMessage}
        <div class="mt-4 p-3 rounded-xl bg-rose-950/70 border border-rose-800 text-rose-300 text-xs">
          {errorMessage}
        </div>
      {/if}

      <form onsubmit={handleSubmit} class="mt-4 space-y-4">
        <div class="grid grid-cols-3 gap-3">
          <div>
            <label for="route-method" class="block text-xs font-semibold uppercase text-slate-400 mb-1">
              Method
            </label>
            <select
              id="route-method"
              bind:value={method}
              class="w-full px-3 py-2 rounded-xl bg-[#16182e] border border-[#262842] text-white font-mono font-bold text-xs focus:ring-1 focus:ring-violet-500"
            >
              <option value="GET">GET</option>
              <option value="POST">POST</option>
              <option value="PUT">PUT</option>
              <option value="DELETE">DELETE</option>
              <option value="PATCH">PATCH</option>
            </select>
          </div>

          <div class="col-span-2">
            <label for="route-path" class="block text-xs font-semibold uppercase text-slate-400 mb-1">
              Path
            </label>
            <input
              id="route-path"
              type="text"
              required
              placeholder="/api/v1/resource"
              bind:value={path}
              class="w-full px-3 py-2 rounded-xl bg-[#16182e] border border-[#262842] text-white font-mono text-xs focus:ring-1 focus:ring-violet-500"
            />
          </div>
        </div>

        <div class="grid grid-cols-3 gap-3">
          <div>
            <label for="route-status" class="block text-xs font-semibold uppercase text-slate-400 mb-1">
              Status Code
            </label>
            <input
              id="route-status"
              type="number"
              bind:value={statusCode}
              class="w-full px-3 py-2 rounded-xl bg-[#16182e] border border-[#262842] text-white font-mono text-xs focus:ring-1 focus:ring-violet-500"
            />
          </div>

          <div class="col-span-2">
            <label for="route-desc" class="block text-xs font-semibold uppercase text-slate-400 mb-1">
              Description
            </label>
            <input
              id="route-desc"
              type="text"
              placeholder="e.g. Return promo campaign discounts"
              bind:value={description}
              class="w-full px-3 py-2 rounded-xl bg-[#16182e] border border-[#262842] text-white text-xs focus:ring-1 focus:ring-violet-500"
            />
          </div>
        </div>

        <div>
          <label for="route-response-body" class="block text-xs font-semibold uppercase text-slate-400 mb-1">
            Mock Response JSON Body
          </label>
          <textarea
            id="route-response-body"
            rows="6"
            bind:value={responseBody}
            class="w-full p-3 rounded-xl bg-[#16182e] border border-[#262842] text-white font-mono text-xs focus:ring-1 focus:ring-violet-500 resize-none"
          ></textarea>
        </div>

        <div class="pt-4 border-t border-[#212338] flex items-center justify-end gap-3">
          <button
            type="button"
            onclick={onClose}
            class="px-4 py-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs font-semibold"
          >
            Cancel
          </button>

          <button
            type="submit"
            disabled={isSubmitting || !path}
            class="px-5 py-2 rounded-xl bg-violet-600 hover:bg-violet-500 text-white font-semibold text-xs shadow-lg shadow-violet-600/30 flex items-center gap-1.5 disabled:opacity-50"
          >
            {#if isSubmitting}
              <span>Mounting Route...</span>
            {:else}
              <span>Save & Mount Route</span>
            {/if}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
