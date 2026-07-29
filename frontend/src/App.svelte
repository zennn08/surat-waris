<script>
  import { onMount } from 'svelte'
  import { user, loadSession, logout, toast, notify } from './lib/stores.js'
  import { path, navigate, match } from './lib/router.js'
  import ConfirmDialog from './lib/ConfirmDialog.svelte'

  import Login from './routes/Login.svelte'
  import ChangePassword from './routes/ChangePassword.svelte'
  import BerkasList from './routes/BerkasList.svelte'
  import BerkasForm from './routes/BerkasForm.svelte'
  import BerkasDetail from './routes/BerkasDetail.svelte'
  import Pejabat from './routes/Pejabat.svelte'
  import Pengaturan from './routes/Pengaturan.svelte'

  let version = ''
  onMount(() => {
    loadSession()
    fetch('/api/version').then((r) => r.json()).then((v) => (version = v.version)).catch(() => {})
  })

  // Pembaruan aplikasi: cek sekali setelah login; kalau offline, diam saja.
  let update = null
  let updating = false
  let updateChecked = false
  let dlgUpdate
  $: if ($user && !$user.must_change_password && !updateChecked) checkUpdate()
  async function checkUpdate() {
    updateChecked = true
    try {
      const r = await fetch('/api/update/check', { credentials: 'include' })
      if (!r.ok) return
      const d = await r.json()
      if (d.available) update = d
    } catch {}
  }
  async function applyUpdate() {
    const ok = await dlgUpdate.ask(
      `Aplikasi akan diperbarui ke versi ${update.latest} lalu dimuat ulang otomatis. ` +
      'Pastikan tidak ada pengisian berkas yang belum disimpan.'
    )
    if (!ok) return
    updating = true
    try {
      const r = await fetch('/api/update/apply', { method: 'POST', credentials: 'include' })
      const d = await r.json().catch(() => null)
      if (!r.ok) throw new Error((d && d.error) || `Gagal (HTTP ${r.status})`)
    } catch (e) {
      notify(e.message, 'error')
      updating = false
      return
    }
    // tunggu proses baru hidup dengan versi baru, lalu muat ulang
    for (let i = 0; i < 60; i++) {
      await new Promise((res) => setTimeout(res, 2000))
      try {
        const v = await fetch('/api/version').then((r) => r.json())
        if (v.version === update.latest) { location.reload(); return }
      } catch {}
    }
    notify('Pembaruan berjalan lama; coba muat ulang halaman ini secara manual', 'error')
    updating = false
  }

  // Resolusi route → { component, props, key }
  $: route = resolve($path)
  function resolve(p) {
    let m
    if (p === '/' || p === '') return { c: BerkasList, props: {} }
    if (p === '/berkas/baru') return { c: BerkasForm, props: {} }
    if ((m = match('/berkas/:id/edit', p))) return { c: BerkasForm, props: { id: m.id } }
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
    {#if update}
      <div class="update-banner">
        <span>Versi baru <strong>{update.latest}</strong> tersedia (sekarang {update.current}).</span>
        <button class="btn btn-sm btn-primary" disabled={updating} on:click={applyUpdate}>
          {updating ? 'Memperbarui, jangan tutup jendela...' : 'Perbarui Sekarang'}
        </button>
      </div>
      <ConfirmDialog bind:this={dlgUpdate} title="Perbarui aplikasi?" confirmLabel="Ya, Perbarui" />
    {/if}
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

<footer class="app-footer">SIWARIS{version ? ` ${version}` : ''} · © Kukerta UNRI Kec. Dumai Timur 2026</footer>

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
