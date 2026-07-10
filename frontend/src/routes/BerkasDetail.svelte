<script>
  import { onMount } from 'svelte'
  import { api } from '../lib/api.js'
  import { notify } from '../lib/stores.js'

  export let id

  let berkas = null
  let loading = true
  let error = ''

  // state edit
  let penerimaSel = null
  let newHarta = ''
  let hartaEdits = {} // id -> deskripsi (buffer)

  async function load() {
    loading = true
    error = ''
    try {
      berkas = await api.get('/api/berkas/' + id)
      penerimaSel = berkas.penerima_kuasa_ahli_waris_id
      hartaEdits = {}
      for (const h of berkas.harta) hartaEdits[h.id] = h.deskripsi
    } catch (e) {
      error = e.message
    } finally {
      loading = false
    }
  }

  async function savePenerima() {
    try {
      berkas = await api.put(`/api/berkas/${id}/penerima-kuasa`, {
        ahli_waris_id: penerimaSel === null ? null : Number(penerimaSel),
      })
      notify('Penerima kuasa disimpan', 'success')
    } catch (e) { notify(e.message, 'error') }
  }

  async function addHarta() {
    const d = newHarta.trim()
    if (!d) return
    try {
      await api.post(`/api/berkas/${id}/harta`, { deskripsi: d })
      newHarta = ''
      await load()
      notify('Harta ditambahkan', 'success')
    } catch (e) { notify(e.message, 'error') }
  }

  async function saveHarta(h) {
    const d = (hartaEdits[h.id] || '').trim()
    if (!d) { notify('Deskripsi tidak boleh kosong', 'error'); return }
    try {
      await api.put(`/api/berkas/${id}/harta/${h.id}`, { deskripsi: d })
      notify('Harta disimpan', 'success')
    } catch (e) { notify(e.message, 'error') }
  }

  async function delHarta(h) {
    if (!confirm('Hapus harta ini?')) return
    try {
      await api.del(`/api/berkas/${id}/harta/${h.id}`)
      await load()
      notify('Harta dihapus', 'success')
    } catch (e) { notify(e.message, 'error') }
  }

  function jk(v) { return v === 'L' ? 'Laki-laki' : v === 'P' ? 'Perempuan' : (v || '—') }
  function namaAhli(aid) {
    const a = berkas.ahli_waris.find((x) => x.id === aid)
    return a ? a.nama : '—'
  }

  onMount(load)
</script>

{#if loading}
  <div class="spinner">Memuat…</div>
{:else if error}
  <div class="alert alert-error">{error}</div>
  <a class="btn" href="#/">← Kembali</a>
{:else}
  <div class="card-title">
    <div>
      <h1 class="mb-0">Detail Berkas</h1>
      <div class="mono muted">{berkas.nomor_surat}</div>
    </div>
    <div class="flex gap">
      <a class="btn btn-ghost" href="#/">← Daftar</a>
      <a class="btn btn-primary" href={`/berkas/${id}/cetak`} target="_blank" rel="noopener">Cetak 3 Surat</a>
    </div>
  </div>

  <div class="alert alert-warn small">
    Data dasar berkas <strong>terkunci</strong> (read-only). Hanya <strong>Penerima Kuasa</strong> dan <strong>Daftar Harta</strong> yang dapat diubah.
  </div>

  <!-- Info berkas -->
  <div class="card">
    <h3>Informasi Berkas</h3>
    <div class="row row-3">
      <div><div class="muted small">Nomor Surat</div><div class="mono">{berkas.nomor_surat}</div></div>
      <div><div class="muted small">Tanggal</div><div>{berkas.tanggal}</div></div>
      <div><div class="muted small">Status</div><span class="badge badge-green">{berkas.status}</span></div>
    </div>
    <div class="mt-2"><div class="muted small">Tempat Tinggal Pewaris</div><div>{berkas.tempat_tinggal_pewaris}</div></div>
  </div>

  <!-- Pewaris -->
  <div class="card">
    <h3>Pewaris (Almarhum/ah)</h3>
    <table>
      <thead><tr><th>Nama</th><th>NIK</th><th>Tgl Meninggal</th><th>No. Surat Kematian</th><th>Tgl Surat Kematian</th></tr></thead>
      <tbody>
        {#each berkas.pewaris as p}
          <tr><td>{p.nama}</td><td class="mono">{p.nik}</td><td>{p.tgl_meninggal}</td><td>{p.no_surat_kematian}</td><td>{p.tgl_surat_kematian}</td></tr>
        {/each}
      </tbody>
    </table>
  </div>

  <!-- Ahli Waris -->
  <div class="card">
    <h3>Ahli Waris</h3>
    <table>
      <thead><tr><th>Nama</th><th>NIK</th><th>Umur</th><th>JK</th><th>Agama</th><th>Alamat</th><th>Keterangan</th></tr></thead>
      <tbody>
        {#each berkas.ahli_waris as a}
          <tr>
            <td>{a.nama}{#if a.id === berkas.penerima_kuasa_ahli_waris_id}&nbsp;<span class="badge badge-green">Penerima Kuasa</span>{/if}</td>
            <td class="mono">{a.nik}</td><td>{a.umur ?? '—'}</td><td>{jk(a.jenis_kelamin)}</td>
            <td>{a.agama || '—'}</td><td>{a.alamat || '—'}</td><td>{a.keterangan || '—'}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>

  <!-- Saksi -->
  <div class="card">
    <h3>Saksi</h3>
    <table>
      <thead><tr><th>Nama</th><th>TTL</th><th>NIK</th><th>Alamat</th><th>Hubungan</th></tr></thead>
      <tbody>
        {#each berkas.saksi as s}
          <tr><td>{s.nama}</td><td>{s.ttl || '—'}</td><td class="mono">{s.nik || '—'}</td><td>{s.alamat || '—'}</td><td>{s.hubungan || '—'}</td></tr>
        {/each}
      </tbody>
    </table>
  </div>

  <!-- EDIT: Surat Kuasa -->
  <div class="card" style="border-color: var(--primary-soft); box-shadow: 0 0 0 2px var(--primary-soft);">
    <h3>Bagian Surat Kuasa <span class="badge badge-gray">dapat diedit</span></h3>

    <div class="field" style="max-width:420px;">
      <label>Penerima Kuasa</label>
      <div class="flex gap">
        <select class="grow" bind:value={penerimaSel}>
          <option value={null}>— Belum dipilih —</option>
          {#each berkas.ahli_waris as a}<option value={a.id}>{a.nama}</option>{/each}
        </select>
        <button class="btn btn-primary" on:click={savePenerima}>Simpan</button>
      </div>
    </div>

    <div class="divider"></div>

    <div class="card-title">
      <h3 class="mb-0" style="font-size:0.95rem;">Daftar Harta / Yang Dikuasakan</h3>
    </div>
    {#if berkas.harta.length === 0}
      <div class="muted small">Belum ada harta.</div>
    {/if}
    {#each berkas.harta as h}
      <div class="flex gap items-center mt-1">
        <input class="grow" bind:value={hartaEdits[h.id]} />
        <button class="btn btn-sm" on:click={() => saveHarta(h)}>Simpan</button>
        <button class="btn btn-sm btn-danger" on:click={() => delHarta(h)}>Hapus</button>
      </div>
    {/each}

    <div class="flex gap items-center mt-2">
      <input class="grow" placeholder="Tambah harta baru…" bind:value={newHarta} on:keydown={(e) => e.key === 'Enter' && addHarta()} />
      <button class="btn btn-primary btn-sm" on:click={addHarta}>+ Tambah</button>
    </div>
  </div>
{/if}
