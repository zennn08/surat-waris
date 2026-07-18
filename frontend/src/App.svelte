<script>
  import { onMount } from 'svelte'
  import { user, loadSession, logout, toast } from './lib/stores.js'
  import { path, navigate, match } from './lib/router.js'

  import Login from './routes/Login.svelte'
  import ChangePassword from './routes/ChangePassword.svelte'
  import BerkasList from './routes/BerkasList.svelte'
  import BerkasForm from './routes/BerkasForm.svelte'
  import BerkasDetail from './routes/BerkasDetail.svelte'
  import Pejabat from './routes/Pejabat.svelte'
  import Pengaturan from './routes/Pengaturan.svelte'

  onMount(loadSession)

  // Resolusi route → { component, props, key }
  $: route = resolve($path)
  function resolve(p) {
    let m
    if (p === '/' || p === '') return { c: BerkasList, props: {} }
    if (p === '/berkas/baru') return { c: BerkasForm, props: {} }
    if ((m = match('/berkas/:id', p))) return { c: BerkasDetail, props: { id: m.id }, key: m.id }
    if (p === '/pejabat') return { c: Pejabat, props: {} }
    if (p === '/pengaturan') return { c: Pengaturan, props: {} }
    return { c: BerkasList, props: {} }
  }

  async function doLogout() {
    await logout()
    navigate('/')
  }

  const links = [
    { to: '/', label: 'Daftar Berkas' },
    { to: '/berkas/baru', label: 'Buat Berkas' },
    { to: '/pejabat', label: 'Pejabat' },
    { to: '/pengaturan', label: 'Pengaturan' },
  ]
  function isActive(p, to) {
    if (to === '/') return p === '/' || p.startsWith('/berkas/') && p !== '/berkas/baru'
    return p === to
  }
</script>

{#if $user === undefined}
  <div class="spinner">Memuat…</div>
{:else if $user === null}
  <Login />
{:else if $user.must_change_password}
  <ChangePassword forced={true} />
{:else}
  <div class="app-shell">
    <header class="topbar">
      <div class="topbar-inner">
        <span class="brand">SIWARIS</span>
        <nav class="nav">
          {#each links as l}
            <a href={'#' + l.to} class:active={isActive($path, l.to)}>{l.label}</a>
          {/each}
        </nav>
        <span class="user">{$user.nama}</span>
        <button class="btn btn-sm btn-ghost" on:click={doLogout}>Keluar</button>
      </div>
    </header>
    <main class="container">
      {#key $path}
        <svelte:component this={route.c} {...route.props} />
      {/key}
    </main>
  </div>
{/if}

{#if $toast}
  <div class="toast {$toast.type}">{$toast.message}</div>
{/if}

<footer class="app-footer">SIWARIS · © Kukerta UNRI Kec. Dumai Timur 2026</footer>

<style>
  .app-footer {
    flex-shrink: 0;
    text-align: center;
    padding: 14px 16px 18px;
    margin-top: 8px;
    font-size: 12px;
    color: var(--muted, #64748b);
  }
</style>
