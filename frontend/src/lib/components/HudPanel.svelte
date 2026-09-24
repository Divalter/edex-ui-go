<script lang="ts">
  let { title, children } = $props();
</script>

<div class="hud-panel box-glow">
  <div class="title-bar text-glow">{title}</div>
  <div class="content">
    {@render children()}
  </div>
  <div class="corner top-left"></div>
  <div class="corner top-right"></div>
  <div class="corner bottom-left"></div>
  <div class="corner bottom-right"></div>
</div>

<style>
  .hud-panel {
    position: relative;
    width: 100%;
    height: 100%;
    background-color: rgba(0, 0, 0, 0.5);
    display: flex;
    flex-direction: column;
    padding: 10px;
    overflow: hidden;
  }

  .title-bar {
    font-size: 14px;
    text-transform: uppercase;
    font-weight: bold;
    border-bottom: 1px solid var(--color-border);
    padding-bottom: 5px;
    margin-bottom: 10px;
    letter-spacing: 2px;
  }

  .content {
    flex: 1;
    overflow: hidden;
    position: relative;
  }

  .corner {
    position: absolute;
    width: 15px;
    height: 15px;
    border: 2px solid var(--color-primary);
  }

  .top-left { top: 0; left: 0; border-right: none; border-bottom: none; }
  .top-right { top: 0; right: 0; border-left: none; border-bottom: none; }
  .bottom-left { bottom: 0; left: 0; border-right: none; border-top: none; }
  .bottom-right { bottom: 0; right: 0; border-left: none; border-top: none; }
</style>
