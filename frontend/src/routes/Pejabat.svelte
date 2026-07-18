<script>
  import { onMount } from 'svelte'
  import { api } from '../lib/api.js'
  import { notify } from '../lib/stores.js'

  let items = []
  let loading = true

  let editingId = null
  let form = blank()
  function blank() { return { jabatan: 'lurah', nama: '', nip: '', aktif: true } }

  async function load() {
    loading = true
    try { items = await api.get('/api/pejabat') } catch (e) { notify(e.message, 'error') } finally { loading = false }
  }

  function startEdit(p) {
    editingId = p.id
    form = { jabatan: p.jabatan, nama: p.nama, nip: p.nip, aktif: p.aktif }
  }
  function cancelEdit() { editingId = null; form = blank() }

  async function save() {
    try {
      if (editingId) await api.put('/api/pejabat/' + editingId, form)
      else await api.post('/api/pejabat', form)
      cancelEdit()
      await load()
      notify('Pejabat disimpan', 'success')
    } catch (e) { notify(e.message, 'error') }
  }

  async function del(p) {
    if (!confirm(`Hapus pejabat ${p.nama}?`)) return
    try { await api.del('/api/pejabat/' + p.id); await load(); notify('Pejabat dihapus', 'success') }
    catch (e) { notify(e.message, 'error') }
  }

  function cap(s) { return s ? s.charAt(0).toUpperCase() + s.slice(1) : s }

  onMount(load)
</script>

<h1>Pejabat Penandatangan</h1>
<div class="section-sub">Lurah / Camat yang dipakai pada surat. Hanya satu pejabat aktif per jabatan.</div>

<div class="card">
  <h3>{editingId ? 'Edit Pejabat' : 'Tambah Pejabat'}</h3>
  <form on:submit|preventDefault={save}>
    <div class="row row-2">
      <div class="field"><label>Jabatan</label>
        <select bind:value={form.jabatan}><option value="lurah">Lurah</option><option value="camat">Camat</option></select>
      </div>
      <div class="field"><label>Nama</label><input bind:value={form.nama} required /></div>
    </div>
    <div class="row row-2">
      <div class="field"><label>NIP</label><input bind:value={form.nip} class="mono" required /></div>
      <div class="field"><label>&nbsp;</label>
        <label class="flex items-center gap" style="font-weight:500;color:var(--text);text-transform:none;">
          <input type="checkbox" style="width:auto;" bind:checked={form.aktif} /> Jadikan pejabat aktif
        </label>
      </div>
    </div>
    <div class="flex gap">
      <button class="btn btn-primary">{editingId ? 'Simpan Perubahan' : 'Tambah'}</button>
      {#if editingId}<button type="button" class="btn btn-ghost" on:click={cancelEdit}>Batal</button>{/if}
    </div>
  </form>
</div>

<div class="card mt-2">
  {#if loading}
    <div class="spinner">Memuat…</div>
  {:else if items.length === 0}
    <div class="empty">Belum ada pejabat.</div>
  {:else}
    <div class="table-wrap">
    <table>
      <thead><tr><th>Jabatan</th><th>Nama</th><th>NIP</th><th>Status</th><th style="width:140px;"></th></tr></thead>
      <tbody>
        {#each items as p}
          <tr>
            <td>{cap(p.jabatan)}</td><td>{p.nama}</td><td class="mono">{p.nip}</td>
            <td>{#if p.aktif}<span class="badge badge-green">Aktif</span>{:else}<span class="badge badge-gray">Non-aktif</span>{/if}</td>
            <td><div class="table-actions">
              <button class="btn btn-sm" on:click={() => startEdit(p)}>Edit</button>
              <button class="btn btn-sm btn-danger" on:click={() => del(p)}>Hapus</button>
            </div></td>
          </tr>
        {/each}
      </tbody>
    </table>
    </div>
  {/if}
</div>
