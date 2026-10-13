<script lang="ts">
  import { api } from '../api';
  import type { DatasourceSchema, FieldInfo } from '../types';

  let {
    isOpen = false,
    datasources = [] as DatasourceSchema[],
    activeDatasource = '',
    onClose = () => {},
    onCreated = () => {},
  } = $props<{
    isOpen: boolean;
    datasources: DatasourceSchema[];
    activeDatasource: string;
    onClose: () => void;
    onCreated: () => void;
  }>();

  let selectedDS = $state('');
  let modelName = $state('');
  let tableName = $state('');
  let fields = $state<
    {
      name: string;
      db_name: string;
      gofyr_type: string;
      required: boolean;
      searchable: boolean;
      readonly: boolean;
    }[]
  >([
    {
      name: 'Title',
      db_name: 'title',
      gofyr_type: 'text',
      required: true,
      searchable: true,
      readonly: false,
    },
    {
      name: 'Description',
      db_name: 'description',
      gofyr_type: 'textarea',
      required: false,
      searchable: false,
      readonly: false,
    },
  ]);

  let isSubmitting = $state(false);
  let errorMessage = $state<string | null>(null);

  $effect(() => {
    if (isOpen) {
      selectedDS = activeDatasource || (datasources[0]?.name ?? 'primary');
      errorMessage = null;
    }
  });

  function handleNameChange(val: string) {
    modelName = val;
    if (!tableName || tableName === slugify(modelName.slice(0, -1))) {
      tableName = slugify(val) + (val.endsWith('s') ? '' : 's');
    }
  }

  function slugify(text: string): string {
    return text
      .toLowerCase()
      .trim()
      .replace(/[^\w\s-]/g, '')
      .replace(/[\s_-]+/g, '_')
      .replace(/^-+|-+$/g, '');
  }

  function addField() {
    fields.push({
      name: '',
      db_name: '',
      gofyr_type: 'text',
      required: false,
      searchable: false,
      readonly: false,
    });
  }

  function removeField(index: number) {
    fields.splice(index, 1);
  }

  async function handleSubmit(e: Event) {
    e.preventDefault();
    if (!modelName.trim() || !tableName.trim() || !selectedDS) return;

    isSubmitting = true;
    errorMessage = null;

    const payloadFields: FieldInfo[] = fields
      .filter((f) => f.name.trim() !== '')
      .map((f) => ({
        name: f.name.trim(),
        db_name: f.db_name.trim() || slugify(f.name),
        type: mapTypeToGo(f.gofyr_type),
        gofyr_type: f.gofyr_type,
        label: f.name.trim(),
        required: f.required,
        searchable: f.searchable,
        readonly: f.readonly,
        hidden: false,
        is_primary_key: false,
      }));

    try {
      await api.createCustomModel({
        datasource: selectedDS,
        name: modelName.trim(),
        table_name: tableName.trim(),
        fields: payloadFields,
      });
      onCreated();
      onClose();
    } catch (err: any) {
      errorMessage = err.message || 'Failed to create model';
    } finally {
      isSubmitting = false;
    }
  }

  function mapTypeToGo(gofyrType: string): string {
    switch (gofyrType) {
      case 'number':
        return 'float64';
      case 'boolean':
        return 'bool';
      case 'datetime':
        return 'time.Time';
      default:
        return 'string';
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
      class="bg-[#111221] border border-[#212338] rounded-2xl p-6 max-w-2xl w-full shadow-2xl my-8 relative"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.stopPropagation()}
      role="document"
      tabindex="0"
    >
      <div class="flex items-center justify-between pb-4 border-b border-[#212338]">
        <div>
          <h2 class="text-base font-bold text-white flex items-center gap-2">
            <span>Create New Model</span>
            <span class="text-[11px] px-2 py-0.5 rounded bg-violet-950/80 text-violet-300 font-mono border border-violet-800/60">
              Dynamic Schema
            </span>
          </h2>
          <p class="text-xs text-slate-400 mt-0.5">
            Define a new model table and fields to register it in real-time
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
        <!-- Datasource & Model / Table Names -->
        <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div>
            <label for="model-ds" class="block text-xs font-semibold uppercase text-slate-400 mb-1">
              Datasource
            </label>
            <select
              id="model-ds"
              bind:value={selectedDS}
              class="w-full px-3 py-2 rounded-xl bg-[#16182e] border border-[#262842] text-white text-xs font-medium focus:ring-1 focus:ring-violet-500"
            >
              {#each datasources as ds}
                <option value={ds.name}>{ds.name} ({ds.dialect})</option>
              {/each}
            </select>
          </div>

          <div>
            <label for="model-name" class="block text-xs font-semibold uppercase text-slate-400 mb-1">
              Model Name
            </label>
            <input
              id="model-name"
              type="text"
              required
              placeholder="e.g. Coupon"
              value={modelName}
              oninput={(e) => handleNameChange((e.target as HTMLInputElement).value)}
              class="w-full px-3 py-2 rounded-xl bg-[#16182e] border border-[#262842] text-white text-xs placeholder-slate-500 focus:ring-1 focus:ring-violet-500"
            />
          </div>

          <div>
            <label for="table-name" class="block text-xs font-semibold uppercase text-slate-400 mb-1">
              Table Name
            </label>
            <input
              id="table-name"
              type="text"
              required
              placeholder="e.g. coupons"
              bind:value={tableName}
              class="w-full px-3 py-2 rounded-xl bg-[#16182e] border border-[#262842] text-white text-xs font-mono placeholder-slate-500 focus:ring-1 focus:ring-violet-500"
            />
          </div>
        </div>

        <!-- Field Definitions -->
        <div class="space-y-2 pt-2">
          <div class="flex items-center justify-between">
            <span class="text-xs font-semibold uppercase tracking-wider text-slate-300">
              Columns & Field Types
            </span>
            <button
              type="button"
              onclick={addField}
              class="text-xs text-violet-400 hover:text-violet-300 font-semibold"
            >
              + Add Column
            </button>
          </div>

          <div class="space-y-2 max-h-60 overflow-y-auto pr-1">
            {#each fields as field, idx}
              <div class="p-3 rounded-xl bg-[#16182e] border border-[#262842] flex flex-col sm:flex-row items-center gap-2">
                <input
                  type="text"
                  placeholder="Field Name (e.g. Code)"
                  bind:value={field.name}
                  oninput={() => {
                    if (!field.db_name) field.db_name = slugify(field.name);
                  }}
                  class="w-full sm:w-1/3 px-2.5 py-1.5 rounded-lg bg-slate-900 border border-slate-700/80 text-white text-xs"
                />

                <input
                  type="text"
                  placeholder="Column (e.g. code)"
                  bind:value={field.db_name}
                  class="w-full sm:w-1/4 px-2.5 py-1.5 rounded-lg bg-slate-900 border border-slate-700/80 text-white font-mono text-xs"
                />

                <select
                  bind:value={field.gofyr_type}
                  class="w-full sm:w-1/4 px-2.5 py-1.5 rounded-lg bg-slate-900 border border-slate-700/80 text-white text-xs"
                >
                  <option value="text">Text (string)</option>
                  <option value="textarea">Textarea (long text)</option>
                  <option value="number">Number (float/int)</option>
                  <option value="boolean">Boolean (true/false)</option>
                  <option value="datetime">Datetime</option>
                  <option value="image">Image Photo</option>
                  <option value="gallery">Gallery (Multi-image)</option>
                  <option value="file">File Attachment</option>
                </select>

                <div class="flex items-center gap-2 text-[11px] text-slate-400">
                  <label class="flex items-center gap-1 cursor-pointer">
                    <input type="checkbox" bind:checked={field.searchable} class="rounded bg-slate-800 border-slate-700 text-violet-500" />
                    <span>Search</span>
                  </label>
                  <label class="flex items-center gap-1 cursor-pointer">
                    <input type="checkbox" bind:checked={field.required} class="rounded bg-slate-800 border-slate-700 text-violet-500" />
                    <span>Req</span>
                  </label>
                </div>

                <button
                  type="button"
                  onclick={() => removeField(idx)}
                  class="p-1 text-slate-500 hover:text-rose-400"
                  title="Remove column"
                >
                  ✕
                </button>
              </div>
            {/each}
          </div>
        </div>

        <!-- Submit & Actions -->
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
            disabled={isSubmitting || !modelName || !tableName}
            class="px-5 py-2 rounded-xl bg-violet-600 hover:bg-violet-500 text-white font-semibold text-xs shadow-lg shadow-violet-600/30 flex items-center gap-1.5 disabled:opacity-50"
          >
            {#if isSubmitting}
              <span>Creating Table...</span>
            {:else}
              <span>Create Model</span>
            {/if}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
