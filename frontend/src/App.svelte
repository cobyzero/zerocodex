<script>
  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';
  import { EventsOn } from '../wailsjs/runtime/runtime';
  import {
    Bootstrap,
    DeleteAPIKey,
    OpenProjectDialog,
    RemoveProject,
    RunPrompt,
    SaveAPIKey,
    SelectProject
  } from '../wailsjs/go/main/App';

  let apiKey = '';
  let apiKeyConfigured = false;
  let currentProject = '';
  let status = 'Ready';
  let activity = '';
  let savedProjects = [];
  let transcript = [];
  let prompt = '';
  let busy = false;
  let errorMessage = '';

  onMount(async () => {
    const offProgress = EventsOn('agent:progress', (payload) => {
      if (!payload || typeof payload !== 'object') {
        return;
      }
      if (typeof payload.status === 'string' && payload.status.trim() !== '') {
        status = payload.status;
      }
      if (typeof payload.activity === 'string') {
        activity = payload.activity;
      }
    });
    await refresh();
    return () => {
      if (typeof offProgress === 'function') {
        offProgress();
      }
    };
  });

  async function refresh() {
    try {
      const data = await Bootstrap();
      applyState(data);
      errorMessage = '';
    } catch (error) {
      errorMessage = error?.message || String(error);
    }
  }

  function applyState(data) {
    apiKeyConfigured = !!data.apiKeyConfigured;
    currentProject = data.currentProject || '';
    status = data.status || 'Ready';
    activity = data.activity || '';
    savedProjects = data.savedProjects || [];
    transcript = data.transcript || [];
  }

  async function runAction(action) {
    busy = true;
    errorMessage = '';
    try {
      const data = await action();
      applyState(data);
    } catch (error) {
      errorMessage = error?.message || String(error);
      if (!status) {
        status = 'Ready';
      }
    } finally {
      busy = false;
    }
  }

  async function saveKey() {
    await runAction(() => SaveAPIKey(apiKey));
    apiKey = '';
  }

  async function deleteKey() {
    await runAction(() => DeleteAPIKey());
    apiKey = '';
  }

  async function openProjectPicker() {
    try {
      const path = await OpenProjectDialog();
      if (!path) {
        return;
      }
      await runAction(() => SelectProject(path));
    } catch (error) {
      errorMessage = error?.message || String(error);
    }
  }

  async function loadProject(path) {
    await runAction(() => SelectProject(path));
  }

  async function dropProject(path) {
    await runAction(() => RemoveProject(path));
  }

  async function submitPrompt() {
    if (!prompt.trim()) {
      errorMessage = 'Escribe una instruccion.';
      return;
    }
    const nextPrompt = prompt;
    prompt = '';
    await runAction(() => RunPrompt(nextPrompt));
  }

  function roleLabel(role) {
    if (role === 'user') return 'Tu';
    if (role === 'assistant') return 'ZeroCodex';
    if (role === 'error') return 'Error';
    return 'Sistema';
  }

  function projectName(path) {
    if (!path) return 'Selecciona un proyecto';
    return path.split(/[/\\]/).pop();
  }

  function composerKeydown(event) {
    if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') {
      submitPrompt();
    }
  }
</script>

<main class="shell">
  <aside class="sidebar">
    <section class="brand card">
      <p class="eyebrow">ZeroCodex</p>
      <h1>Desktop Agent</h1>
      <p class="muted">Go + Wails + Svelte</p>
    </section>

    <section class="card key-card">
      <div class="section-head">
        <h2>DeepSeek</h2>
        <span class="badge">{status}</span>
      </div>
      {#if apiKeyConfigured}
        <p class="muted key-state">Key cargada</p>
        <div class="stack compact">
          <button class="ghost" disabled={busy} on:click={deleteKey}>Quitar key</button>
        </div>
      {:else}
        <p class="muted key-state">API key no configurada.</p>
        <input bind:value={apiKey} class="text-input" placeholder="sk-..." type="password" />
        <div class="stack compact">
          <button class="primary" disabled={busy} on:click={saveKey}>Guardar key</button>
        </div>
      {/if}
      <p class="activity">{activity || 'Sin actividad.'}</p>
    </section>

    <section class="card">
      <div class="section-head">
        <h2>Proyectos</h2>
      </div>
      <div class="stack">
        <button class="primary wide" disabled={busy} on:click={openProjectPicker}>Abrir carpeta</button>
      </div>
      <div class="project-list">
        {#if savedProjects.length}
          {#each savedProjects as project}
            <article class:active={project.path === currentProject} class="project-card">
              <div>
                <h3>{project.name}</h3>
                <p>{project.path}</p>
              </div>
              <div class="project-actions">
                <button class="ghost" disabled={busy} on:click={() => loadProject(project.path)}>Abrir</button>
                <button class="ghost" disabled={busy} on:click={() => dropProject(project.path)}>Quitar</button>
              </div>
            </article>
          {/each}
        {:else}
          <div class="empty">No hay proyectos guardados todavia.</div>
        {/if}
      </div>
    </section>
  </aside>

  <section class="workspace">
    <header class="hero card">
      <div class="hero-main">
        <p class="eyebrow">Workspace</p>
        <h2>{projectName(currentProject)}</h2>
      </div>
      <p class="status-line">{status}</p>
    </header>

    {#if activity}
      <div class="activity-banner">{activity}</div>
    {/if}

    {#if errorMessage}
      <div class="error-banner">{errorMessage}</div>
    {/if}

    <section class="chat card">
      {#if transcript.length}
        {#each transcript as entry}
          <article class={`message ${entry.role}`} transition:fade={{ duration: 140 }}>
            <div class="message-role">{roleLabel(entry.role)}</div>
            <pre>{entry.content}</pre>
          </article>
        {/each}
      {:else}
        <div class="empty hero-empty">Aun no hay conversacion para este proyecto.</div>
      {/if}
    </section>

    <section class="composer card">
      <label for="prompt">Instruccion</label>
      <textarea
        id="prompt"
        bind:value={prompt}
        class="prompt"
        disabled={busy}
        on:keydown={composerKeydown}
        placeholder="Pide un cambio, fix, refactor o documentacion."
      ></textarea>
      <div class="composer-foot">
        <div class="composer-meta">
          {#if busy}
            <p class="thinking-line" transition:fade={{ duration: 120 }}>
              <span class="loader-dots" aria-hidden="true"><i></i><i></i><i></i></span>
              Esta pensando...
            </p>
          {:else}
            <p class="muted">Usa `Cmd/Ctrl + Enter` para enviar.</p>
          {/if}
          <div class:thinking-bar={busy} class="thinking-track"></div>
        </div>
        <button class="primary" disabled={busy} on:click={submitPrompt}>
          {#if busy}
            Enviando...
          {:else}
            Enviar
          {/if}
        </button>
      </div>
    </section>
  </section>
</main>
