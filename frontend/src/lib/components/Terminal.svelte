<script lang="ts">
  import { onMount } from 'svelte';
  import { Terminal } from '@xterm/xterm';
  import { FitAddon } from '@xterm/addon-fit';
  import { WebglAddon } from '@xterm/addon-webgl';
  import '@xterm/xterm/css/xterm.css';
  import { themeStore } from '../stores/theme';
  import { TerminalWebSocket } from '../utils/websocket';
  
  import { 
    GetTerminalWSInfo, 
    CreateTerminalSession, 
    ResizeTerminal, 
    CloseTerminalSession 
  } from '../../../wailsjs/go/main/App.js';

  let { sessionId } = $props<{ sessionId: string }>();

  let terminalElement = $state<HTMLElement>();
  let xterm = $state<Terminal>();
  let fitAddon = $state<FitAddon>();
  let wsClient = $state<TerminalWebSocket>();

  $effect(() => {
    if (!terminalElement) return;

    xterm = new Terminal({
      cursorBlink: true,
      fontFamily: 'Share Tech Mono, monospace',
      fontSize: 14,
      theme: {
        background: themeStore.theme.background,
        foreground: themeStore.theme.text,
        cursor: themeStore.theme.primary,
        selectionBackground: 'rgba(0, 255, 230, 0.3)'
      }
    });

    fitAddon = new FitAddon();
    xterm.loadAddon(fitAddon);

    xterm.open(terminalElement);
    
    try {
      const webgl = new WebglAddon();
      webgl.onContextLoss(e => {
        webgl.dispose();
      });
      xterm.loadAddon(webgl);
    } catch (e) {
      console.warn("WebGL addon failed to load, falling back to canvas/dom renderer", e);
    }

    fitAddon.fit();

    let resizeObserver = new ResizeObserver(() => {
      if (fitAddon && xterm) {
        fitAddon.fit();
        ResizeTerminal(sessionId, xterm.cols, xterm.rows).catch(() => {});
      }
    });
    
    resizeObserver.observe(terminalElement);

    const initConnection = async () => {
      try {
        let port = 8080;
        let token = 'dev-token';
        
        try {
          const info = await GetTerminalWSInfo();
          if (info && info.port) {
            port = info.port;
            token = info.token;
          }
          await CreateTerminalSession(sessionId);
        } catch (e) {
          console.warn("Wails runtime not yet available or failed to create session", e);
        }

        wsClient = new TerminalWebSocket(port, sessionId, token);
        wsClient.connect(
          (data) => {
            if (xterm) xterm.write(data);
          },
          () => {
            if (xterm) xterm.write('\r\n[Terminal Disconnected]\r\n');
          },
          (err) => {
            if (xterm) xterm.write(`\r\n[Terminal Error: ${err}]\r\n`);
          }
        );

        xterm!.onData(data => {
          if (wsClient) wsClient.send(data);
        });

      } catch (err) {
        console.error("Failed to init terminal connection", err);
        if (xterm) xterm.write('\r\n[Failed to connect to backend]\r\n');
      }
    };

    initConnection();

    return () => {
      resizeObserver.disconnect();
      if (wsClient) wsClient.close();
      if (xterm) xterm.dispose();
      CloseTerminalSession(sessionId).catch(() => {});
    };
  });
</script>

<div class="terminal-container" bind:this={terminalElement}></div>

<style>
  .terminal-container {
    width: 100%;
    height: 100%;
    padding: 10px;
    background: rgba(0, 0, 0, 0.4);
    border: 1px solid var(--color-highlight);
  }
  
  :global(.xterm-viewport) {
    /* Hide scrollbar for more sci-fi look */
    overflow-y: hidden !important;
  }
</style>
