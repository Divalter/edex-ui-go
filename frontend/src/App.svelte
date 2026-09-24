<script lang="ts">
  import BootScreen from './lib/components/BootScreen.svelte';
  import HudPanel from './lib/components/HudPanel.svelte';
  import TabBar, { type Tab } from './lib/components/TabBar.svelte';
  import Terminal from './lib/components/Terminal.svelte';
  import { themeStore } from './lib/stores/theme';
  import { configStore } from './lib/stores/config';

  let isBooting = $state(true);
  let showKeyboard = $state(false);
  
  let tabs = $state<Tab[]>([
    { id: 'tab1', title: 'Terminal 1' }
  ]);
  let activeTabId = $state('tab1');
  let nextTabId = 2;

  $effect(() => {
    configStore.load().then(() => {
      if (configStore.config.noIntro) {
        isBooting = false;
      }
      themeStore.load(configStore.config.theme).then(() => {
        themeStore.apply();
      });
    });

    const handleKeyDown = (e: KeyboardEvent) => {
      // Toggle virtual keyboard: Ctrl+Shift+K
      if (e.ctrlKey && e.shiftKey && (e.key === 'K' || e.key === 'k')) {
        e.preventDefault();
        showKeyboard = !showKeyboard;
      }
      // Tab switching shortcuts: Ctrl+1..5
      if (e.ctrlKey && !e.shiftKey && e.key >= '1' && e.key <= '5') {
        const tabIndex = parseInt(e.key) - 1;
        if (tabIndex < tabs.length) {
          e.preventDefault();
          activeTabId = tabs[tabIndex].id;
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  });

  function handleBootComplete() {
    isBooting = false;
  }

  function handleSelectTab(id: string) {
    activeTabId = id;
  }

  function handleCloseTab(id: string) {
    tabs = tabs.filter(t => t.id !== id);
    if (tabs.length === 0) {
      handleNewTab();
    } else if (activeTabId === id) {
      activeTabId = tabs[tabs.length - 1].id;
    }
  }

  function handleNewTab() {
    if (tabs.length >= 5) return;
    const newId = `tab${nextTabId++}`;
    tabs = [...tabs, { id: newId, title: `Terminal ${nextTabId - 1}` }];
    activeTabId = newId;
  }

  function toggleKeyboard() {
    showKeyboard = !showKeyboard;
  }
</script>

{#if isBooting}
  <BootScreen oncomplete={handleBootComplete} />
{:else}
  <div class="layout" class:keyboard-open={showKeyboard}>
    <div class="column left">
      <HudPanel title="System Stats">
        <div class="placeholder-content text-glow">
          [HOST] EDEX-GO-STATION<br/>
          [OS] LINUX / AMD64<br/>
          [UPTIME] 00:42:19<br/>
          [CPU LOAD] 3.2%<br/>
          [RAM] 1.8GB / 16.0GB
        </div>
      </HudPanel>
      <div style="height: 10px;"></div>
      <HudPanel title="Network Info">
        <div class="placeholder-content text-glow">
          [IFACE] ETH0 (ONLINE)<br/>
          [IP] 192.168.1.100<br/>
          [PING] 14 ms (1.1.1.1)<br/>
          TX: 124 Kbps<br/>
          RX: 504 Kbps
        </div>
      </HudPanel>
    </div>

    <div class="center-main">
      <HudPanel title="Main Terminal">
        <div class="terminal-header-controls">
          <TabBar 
            {tabs} 
            {activeTabId}
            onselect={handleSelectTab}
            onclose={handleCloseTab}
            onnew={handleNewTab}
          />
          <button 
            class="kb-toggle-btn text-glow" 
            onclick={toggleKeyboard}
            title="Toggle Virtual Keyboard (Ctrl+Shift+K)"
          >
            KEYBOARD: {showKeyboard ? '[ON]' : '[OFF]'}
          </button>
        </div>
        
        <div class="terminal-wrapper">
          {#each tabs as tab (tab.id)}
            <div class="terminal-tab-content" style="display: {tab.id === activeTabId ? 'block' : 'none'}">
              {#if tab.id === activeTabId}
                <Terminal sessionId={tab.id} />
              {/if}
            </div>
          {/each}
        </div>
      </HudPanel>
    </div>

    <div class="column right">
      <HudPanel title="Top Processes">
        <div class="placeholder-content text-glow">
          PID   CMD        %CPU  %MEM<br/>
          ---------------------------<br/>
          1     init       0.1   0.2<br/>
          842   wails      0.4   0.5<br/>
          1204  xterm      0.3   0.4<br/>
          3412  bash       0.0   0.1
        </div>
      </HudPanel>
    </div>

    {#if showKeyboard}
      <div class="keyboard-area">
        <HudPanel title="Virtual Keyboard [Ctrl+Shift+K]">
          <div class="placeholder-content text-glow" style="text-align: center; padding: 20px;">
            <p>[TOUCH KEYBOARD MODULE DOCKED]</p>
            <p style="font-size: 0.8rem; opacity: 0.7;">Click keys or toggle off with [Ctrl+Shift+K]</p>
          </div>
        </HudPanel>
      </div>
    {/if}
  </div>
  
  <div class="scanline"></div>
{/if}

<style>
  .layout {
    display: grid;
    grid-template-columns: 260px 1fr 260px;
    grid-template-rows: 1fr;
    gap: 12px;
    padding: 12px;
    width: 100vw;
    height: 100vh;
    box-sizing: border-box;
    transition: grid-template-rows 0.2s ease-in-out;
  }

  .layout.keyboard-open {
    grid-template-rows: 1fr 180px;
  }

  .column {
    display: flex;
    flex-direction: column;
    height: 100%;
    grid-row: 1 / 2;
  }

  .left {
    grid-column: 1 / 2;
  }

  .right {
    grid-column: 3 / 4;
  }

  .center-main {
    grid-column: 2 / 3;
    grid-row: 1 / 2;
    display: flex;
    flex-direction: column;
  }

  .terminal-header-controls {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;
    gap: 10px;
  }

  .kb-toggle-btn {
    background: transparent;
    border: 1px solid var(--color-border);
    color: var(--color-text);
    font-family: var(--font-main);
    font-size: 0.8rem;
    padding: 4px 10px;
    cursor: pointer;
    transition: all 0.2s;
    letter-spacing: 1px;
    white-space: nowrap;
  }

  .kb-toggle-btn:hover {
    background: var(--color-primary);
    color: var(--color-bg);
  }

  .keyboard-area {
    grid-column: 1 / 4;
    grid-row: 2 / 3;
    animation: fadeIn 0.2s ease-in-out;
  }

  .placeholder-content {
    font-family: var(--font-main);
    color: var(--color-text);
    opacity: 0.85;
    line-height: 1.6;
    font-size: 0.85rem;
  }

  .terminal-wrapper {
    flex: 1;
    position: relative;
    overflow: hidden;
  }
  
  .terminal-tab-content {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
  }

  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(10px); }
    to { opacity: 1; transform: translateY(0); }
  }
</style>
