<script lang="ts">
  import { api } from '../api';
  import type { ModelInfo } from '../types';
  import Uploader from './Uploader.svelte';

  let {
    isOpen = false,
    model = null as ModelInfo | null,
    datasource = '',
    initialData = null as Record<string, any> | null,
    onClose = () => {},
    onSaved = () => {},
  } = $props<{
    isOpen: boolean;
    model: ModelInfo | null;
    datasource: string;
    initialData: Record<string, any> | null;
    onClose: () => void;
    onSaved: () => void;
  }>();

  let formData = $state<Record<string, any>>({});
  let isSaving = $state(false);
  let errorMessage = $state<string | null>(null);

  // Update form data when initialData or model changes
  $effect(() => {
    if (isOpen && model) {
      errorMessage = null;
      if (initialData) {
        // Deep copy
        formData = JSON.parse(JSON.stringify(initialData));
      } else {
        const init: Record<string, any> = {};
        for (const field of model.fields) {
          if (field.is_primary_key) continue;
          if (field.gofyr_type === 'boolean') init[field.db_name] = false;
          else if (field.gofyr_type === 'gallery') init[field.db_name] = [];
          else if (field.gofyr_type === 'number') init[field.db_name] = 0;
          else init[field.db_name] = '';
        }
        formData = init;
      }
    }
  });

  const isEditing = $derived(!!initialData && !!model && !!initialData[model.primary_key]);

  async function handleSubmit(e: Event) {
    e.preventDefault();
    if (!model || !datasource) return;

    isSaving = true;
    errorMessage = null;

    try {
      if (isEditing) {
        const id = initialData![model.primary_key];
        await api.updateRecord(datasource, model.name, id, formData);
      } else {
        await api.createRecord(datasource, model.name, formData);
      }
      onSaved();
      onClose();
    } catch (err: any) {
      errorMessage = err.message || 'Failed to save record';
    } finally {
      isSaving = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && isOpen) {
      onClose();
    }
  }

  function addGalleryItem(fieldName: string, url: string) {
    if (!url) return;
    const current = Array.isArray(formData[fieldName]) ? [...formData[fieldName]] : [];
    current.push(url);
    formData[fieldName] = current;
  }

  function removeGalleryItem(fieldName: string, index: number) {
    const current = Array.isArray(formData[fieldName]) ? [...formData[fieldName]] : [];
    current.splice(index, 1);
    formData[fieldName] = current;
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if isOpen && model}
  <div class="fixed inset-0 z-40 overflow-hidden">
    <!-- Backdrop -->
    <div
      class="absolute inset-0 bg-black/60 backdrop-blur-sm transition-opacity"
      onclick={onClose}
      aria-hidden="true"
    ></div>

    <div class="fixed inset-y-0 right-0 max-w-full flex pl-10">
      <div class="w-screen max-w-xl bg-slate-900 border-l border-slate-800 shadow-2xl flex flex-col">
        <!-- Header -->
        <div class="px-6 py-5 border-b border-slate-800 flex items-center justify-between bg-slate-900/90 backdrop-blur">
          <div>
            <h2 class="text-lg font-bold text-white flex items-center gap-2">
              {#if isEditing}
                <span>Edit {model.name}</span>
                <span class="text-xs px-2 py-0.5 rounded bg-slate-800 text-indigo-400 font-mono border border-slate-700">
                  #{initialData?.[model.primary_key]}
                </span>
              {:else}
                <span>New {model.name}</span>
              {/if}
            </h2>
            <p class="text-xs text-slate-400 mt-0.5">
              Datasource: <span class="text-slate-300 font-medium">{datasource}</span> • Table: <span class="font-mono text-slate-300">{model.table_name}</span>
            </p>
          </div>

          <button
            onclick={onClose}
            class="p-2 rounded-lg text-slate-400 hover:text-white hover:bg-slate-800 transition-colors"
            title="Close Drawer"
          >
            <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- Form Content -->
        <form onsubmit={handleSubmit} class="flex-1 overflow-y-auto px-6 py-6 space-y-6">
          {#if errorMessage}
            <div class="p-4 rounded-xl bg-rose-950/60 border border-rose-800 text-rose-300 text-sm flex items-start gap-3">
              <svg class="w-5 h-5 flex-shrink-0 text-rose-400 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span>{errorMessage}</span>
            </div>
          {/if}

          {#each model.fields as field}
            {#if !field.hidden}
              <div class="space-y-1.5">
                <div class="flex items-center justify-between">
                  <label for={field.db_name} class="text-xs font-semibold text-slate-300 uppercase tracking-wider flex items-center gap-1.5">
                    <span>{field.label}</span>
                    {#if field.required}
                      <span class="text-rose-400">*</span>
                    {/if}
                  </label>

                  <span class="text-[10px] text-slate-500 font-mono">
                    {field.db_name} ({field.type})
                  </span>
                </div>

                {#if field.is_primary_key}
                  <input
                    id={field.db_name}
                    type="text"
                    disabled
                    value={formData[field.db_name] || '(auto-generated)'}
                    class="w-full px-3.5 py-2.5 rounded-xl bg-slate-800/50 border border-slate-700/60 text-slate-400 font-mono text-sm cursor-not-allowed"
                  />
                {:else if field.gofyr_type === 'image' || field.gofyr_type === 'file'}
                  <Uploader
                    value={formData[field.db_name] || ''}
                    disk={field.disk || 'local'}
                    folder={field.folder || 'uploads'}
                    type={field.gofyr_type}
                    disabled={field.readonly}
                    onChange={(url) => {
                      formData[field.db_name] = url;
                    }}
                  />
                {:else if field.gofyr_type === 'gallery'}
                  <div class="space-y-3">
                    <!-- Existing Gallery Grid -->
                    {#if Array.isArray(formData[field.db_name]) && formData[field.db_name].length > 0}
                      <div class="grid grid-cols-3 gap-2.5">
                        {#each formData[field.db_name] as photo, idx}
                          <div class="relative group rounded-lg overflow-hidden border border-slate-700 bg-slate-800 aspect-square">
                            <img src={photo} alt="" class="w-full h-full object-cover" />
                            <button
                              type="button"
                              onclick={() => removeGalleryItem(field.db_name, idx)}
                              class="absolute top-1 right-1 p-1 rounded-md bg-rose-900/90 hover:bg-rose-800 text-white opacity-0 group-hover:opacity-100 transition-opacity"
                              title="Delete photo"
                            >
                              <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                              </svg>
                            </button>
                          </div>
                        {/each}
                      </div>
                    {/if}

                    <!-- Uploader for new gallery item -->
                    <Uploader
                      value=""
                      disk={field.disk || 'local'}
                      folder={field.folder || 'products'}
                      type="image"
                      disabled={field.readonly}
                      onChange={(url) => addGalleryItem(field.db_name, url)}
                    />
                  </div>
                {:else if field.gofyr_type === 'boolean'}
                  <div class="flex items-center gap-3 pt-1">
                    <input
                      id={field.db_name}
                      type="checkbox"
                      bind:checked={formData[field.db_name]}
                      disabled={field.readonly}
                      class="w-5 h-5 rounded border-slate-700 bg-slate-800 text-indigo-600 focus:ring-indigo-500 focus:ring-offset-slate-900"
                    />
                    <label for={field.db_name} class="text-sm text-slate-300 select-none cursor-pointer">
                      {formData[field.db_name] ? 'True / Enabled' : 'False / Disabled'}
                    </label>
                  </div>
                {:else if field.gofyr_type === 'textarea'}
                  <textarea
                    id={field.db_name}
                    bind:value={formData[field.db_name]}
                    disabled={field.readonly}
                    rows="3"
                    class="w-full px-3.5 py-2.5 rounded-xl bg-slate-800 border border-slate-700/80 text-white placeholder-slate-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 text-sm transition-all"
                    placeholder={`Enter ${field.label}...`}
                  ></textarea>
                {:else if field.gofyr_type === 'number'}
                  <input
                    id={field.db_name}
                    type="number"
                    step="any"
                    bind:value={formData[field.db_name]}
                    disabled={field.readonly}
                    class="w-full px-3.5 py-2.5 rounded-xl bg-slate-800 border border-slate-700/80 text-white placeholder-slate-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 text-sm transition-all"
                  />
                {:else if field.gofyr_type === 'datetime'}
                  <input
                    id={field.db_name}
                    type="datetime-local"
                    bind:value={formData[field.db_name]}
                    disabled={field.readonly}
                    class="w-full px-3.5 py-2.5 rounded-xl bg-slate-800 border border-slate-700/80 text-white placeholder-slate-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 text-sm transition-all"
                  />
                {:else}
                  <input
                    id={field.db_name}
                    type={field.gofyr_type === 'password' ? 'password' : 'text'}
                    bind:value={formData[field.db_name]}
                    disabled={field.readonly}
                    class="w-full px-3.5 py-2.5 rounded-xl bg-slate-800 border border-slate-700/80 text-white placeholder-slate-500 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 text-sm transition-all"
                    placeholder={`Enter ${field.label}...`}
                  />
                {/if}
              </div>
            {/if}
          {/each}

          <div class="pt-6 border-t border-slate-800 flex items-center justify-end gap-3 sticky bottom-0 bg-slate-900 pb-2">
            <button
              type="button"
              onclick={onClose}
              disabled={isSaving}
              class="px-4 py-2.5 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 hover:text-white font-medium text-sm transition-colors"
            >
              Cancel
            </button>

            <button
              type="submit"
              disabled={isSaving}
              class="px-5 py-2.5 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-medium text-sm shadow-lg shadow-indigo-600/30 flex items-center gap-2 transition-all disabled:opacity-50"
            >
              {#if isSaving}
                <svg class="animate-spin w-4 h-4 text-white" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                </svg>
                <span>Saving...</span>
              {:else}
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                </svg>
                <span>{isEditing ? 'Save Changes' : 'Create Record'}</span>
              {/if}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
{/if}
