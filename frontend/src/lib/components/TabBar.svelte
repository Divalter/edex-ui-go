<script lang="ts">
  export interface Tab {
    id: string;
    title: string;
  }
  
  let { tabs, activeTabId, onselect, onclose, onnew } = $props<{
    tabs: Tab[], 
    activeTabId: string,
    onselect: (id: string) => void,
    onclose: (id: string) => void,
    onnew: () => void
  }>();

  $effect(() => {
    const handleKeydown = (e: KeyboardEvent) => {
      if (e.ctrlKey) {
        const num = parseInt(e.key);
        if (num >= 1 && num <= 5 && num <= tabs.length) {
          e.preventDefault();
          onselect(tabs[num - 1].id);
        } else if (e.key === 'Tab') {
          e.preventDefault();
          const currentIndex = tabs.findIndex((t: Tab) => t.id === activeTabId);
          if (currentIndex !== -1) {
            const nextIndex = e.shiftKey 
              ? (currentIndex - 1 + tabs.length) % tabs.length
              : (currentIndex + 1) % tabs.length;
            onselect(tabs[nextIndex].id);
          }
        }
      }
    };
    
    window.addEventListener('keydown', handleKeydown);
    return () => window.removeEventListener('keydown', handleKeydown);
  });
</script>

<div class="tab-bar">
  {#each tabs as tab}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div 
      class="tab {activeTabId === tab.id ? 'active box-glow text-glow' : ''}"
      onclick={() => onselect(tab.id)}
    >
      <span class="tab-title">{tab.title}</span>
      <span class="tab-close" onclick={(e) => { e.stopPropagation(); onclose(tab.id); }}>&times;</span>
    </div>
  {/each}
  
  {#if tabs.length < 5}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="tab new-tab" onclick={onnew}>+</div>
  {/if}
</div>

<style>
  .tab-bar {
    display: flex;
    gap: 5px;
    height: 40px;
    padding: 0 10px;
    border-bottom: 1px solid var(--color-border);
    margin-bottom: 10px;
    background: rgba(0,0,0,0.3);
  }

  .tab {
    display: flex;
    align-items: center;
    padding: 0 15px;
    border: 1px solid var(--color-border);
    border-bottom: none;
    cursor: pointer;
    font-size: 14px;
    background: rgba(0, 255, 230, 0.05);
    color: var(--color-text);
    transition: all 0.2s;
    min-width: 120px;
    justify-content: space-between;
  }

  .tab:hover {
    background: rgba(0, 255, 230, 0.1);
  }

  .tab.active {
    background: rgba(0, 255, 230, 0.15);
    border-top-width: 2px;
  }

  .tab-title {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .tab-close {
    font-size: 18px;
    margin-left: 10px;
    opacity: 0.7;
  }

  .tab-close:hover {
    opacity: 1;
    color: var(--color-error);
    text-shadow: 0 0 5px var(--color-error);
  }

  .new-tab {
    min-width: 40px;
    justify-content: center;
    font-size: 20px;
    font-weight: bold;
  }
</style>
