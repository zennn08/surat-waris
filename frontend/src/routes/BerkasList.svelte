<script>
  import { onMount } from 'svelte'
  import { api } from '../lib/api.js'
  import { navigate } from '../lib/router.js'

  let items = []
  let loading = true
  let q = ''
  let error = ''
  let timer

  async function load() {
    loading = true
    error = ''
    try {
      const path = q.trim() ? `/api/berkas?q=${encodeURIComponent(q.trim())}` : '/api/berkas'
      items = await api.get(path)
    } catch (e) {
      error = e.message
    } finally {
      loading = false
    }
  }

  function onSearch() {
    clearTimeout(timer)
    timer = setTimeout(load, 250)
  }

  function pewarisNames(b) {
    return (b.pewaris || []).map((p) => p.nama).join(', ') || '—'
  }

  onMount(load)
</script>

<div class="card-title">
  <h1 class="mb-0">Daftar Berkas Waris</h1>
  <button class="btn btn-primary" on:click={() => navigate('/berkas/baru')}>+ Buat Berkas Baru</button>
</div>

<div class="card">
  <div class="field mb-0">
    <input placeholder="Cari nomor surat, nama atau NIK pewaris…" bind:value={q} on:input={onSearch} />
  </div>
</div>

{#if error}<div class="alert alert-error mt-2">{error}</div>{/if}

<div class="card mt-2">
  {#if loading}
    <div class="spinner">Memuat…</div>
  {:else if items.length === 0}
    <div class="empty">Belum ada berkas. Klik “Buat Berkas Baru”.</div>
  {:else}
    <table>
      <thead>
        <tr>
          <th style="width:190px;">Nomor Surat</th>
          <th>Pewaris (Alm.)</th>
          <th style="width:110px;">Tanggal</th>
          <th style="width:90px;">Status</th>
          <th style="width:80px;"></th>
        </tr>
      </thead>
      <tbody>
        {#each items as b}
          <tr>
            <td class="mono">{b.nomor_surat}</td>
            <td>{pewarisNames(b)}</td>
            <td>{b.tanggal}</td>
            <td><span class="badge badge-green">{b.status}</span></td>
            <td class="right">
              <a class="btn btn-sm" href={'#/berkas/' + b.id}>Detail</a>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>
