<script lang="ts">
  let { oncomplete } = $props();
  
  let lines = $state<string[]>([]);
  let showTitle = $state(false);
  
  const bootSequence = [
    "BIOS Date 09/23/26 17:41:15 Ver 08.00.10",
    "CPU: Quantum Process Unit @ 12.4 GHz",
    "Speed: 12.4 GHz",
    "Initializing Neural Interface...",
    "Memory Test:  32768M OK",
    "Detecting Primary Master ... Quantum SSD",
    "Detecting Primary Slave  ... None",
    "Booting system...",
    "[ OK ] Started Kernel Logging Service",
    "[ OK ] Started Cybernetic Link",
    "Mounting Virtual File Systems...",
    "Accessing Mainframe..."
  ];

  $effect(() => {
    let lineIndex = 0;
    
    const interval = setInterval(() => {
      if (lineIndex < bootSequence.length) {
        lines.push(bootSequence[lineIndex]);
        lines = lines;
        lineIndex++;
      } else {
        clearInterval(interval);
        setTimeout(() => {
          showTitle = true;
          setTimeout(() => {
            if (oncomplete) oncomplete();
          }, 2000);
        }, 500);
      }
    }, 150);
    
    return () => clearInterval(interval);
  });
  
  function skip() {
    if (oncomplete) oncomplete();
  }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="boot-screen" onclick={skip}>
  <div class="terminal-text">
    {#each lines as line}
      <div class="line text-glow">{line}</div>
    {/each}
  </div>
  
  {#if showTitle}
    <div class="title-container text-glow">
      <h1>eDEX-UI-GO</h1>
      <p>SYSTEM ONLINE</p>
    </div>
  {/if}
</div>

<style>
  .boot-screen {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    background-color: #000;
    z-index: 9999;
    display: flex;
    flex-direction: column;
    padding: 2rem;
    font-family: var(--font-main);
    color: var(--color-text);
  }

  .terminal-text {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .line {
    font-size: 1.2rem;
  }

  .title-container {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    text-align: center;
    animation: fadeIn 1s forwards;
  }

  h1 {
    font-size: 4rem;
    margin: 0;
    letter-spacing: 0.5rem;
  }

  p {
    font-size: 1.5rem;
    margin-top: 1rem;
    letter-spacing: 0.2rem;
  }
</style>
