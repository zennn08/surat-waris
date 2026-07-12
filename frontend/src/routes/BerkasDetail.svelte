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
  let newKuasa = ''
  let kuasaEdits = {} // id -> deskripsi (buffer)
  let pelengkap = { tempat_lahir: '', tgl_lahir: '', pekerjaan: '' }

  async function load() {
    loading = true
    error = ''
    try {
      berkas = await api.get('/api/berkas/' + id)
      penerimaSel = berkas.penerima_kuasa_ahli_waris_id
      kuasaEdits = {}
      for (const k of berkas.kuasa) kuasaEdits[k.id] = k.deskripsi
      syncPelengkap()
    } catch (e) {
      error = e.message
    } finally {
      loading = false
    }
  }

  function syncPelengkap() {
    const pk = berkas.ahli_waris.find((a) => a.id === berkas.penerima_kuasa_ahli_waris_id)
    pelengkap = pk
      ? { tempat_lahir: pk.tempat_lahir || '', tgl_lahir: pk.tgl_lahir || '', pekerjaan: pk.pekerjaan || '' }
      : { tempat_lahir: '', tgl_lahir: '', pekerjaan: '' }
  }

  async function savePenerima() {
    try {
      berkas = await api.put(`/api/berkas/${id}/penerima-kuasa`, {
        ahli_waris_id: penerimaSel === null ? null : Number(penerimaSel),
      })
      syncPelengkap()
      notify('Penerima kuasa disimpan', 'success')
    } catch (e) { notify(e.message, 'error') }
  }

  async function savePelengkap() {
    const pkId = berkas.penerima_kuasa_ahli_waris_id
    if (!pkId) { notify('Pilih penerima kuasa dulu', 'error'); return }
    try {
      berkas = await api.put(`/api/berkas/${id}/ahli-waris/${pkId}/pelengkap`, pelengkap)
      syncPelengkap()
      notify('Data pelengkap penerima kuasa disimpan', 'success')
    } catch (e) { notify(e.message, 'error') }
  }

  async function addKuasa() {
    const d = newKuasa.trim()
    if (!d) return
    try {
      await api.post(`/api/berkas/${id}/kuasa`, { deskripsi: d })
      newKuasa = ''
      await load()
      notify('Item kuasa ditambahkan', 'success')
    } catch (e) { notify(e.message, 'error') }
  }

  async function saveKuasa(k) {
    const d = (kuasaEdits[k.id] || '').trim()
    if (!d) { notify('Deskripsi tidak boleh kosong', 'error'); return }
    try {
      await api.put(`/api/berkas/${id}/kuasa/${k.id}`, { deskripsi: d })
      notify('Item kuasa disimpan', 'success')
    } catch (e) { notify(e.message, 'error') }
  }

  async function delKuasa(k) {
    if (!confirm('Hapus item kuasa ini?')) return
    try {
      await api.del(`/api/berkas/${id}/kuasa/${k.id}`)
      await load()
      notify('Item kuasa dihapus', 'success')
    } catch (e) { notify(e.message, 'error') }
  }

  function jk(v) { return v === 'L' ? 'Laki-laki' : v === 'P' ? 'Perempuan' : (v || '—') }
  function statusLabel(s) { return s === 'istri' ? 'Istri' : 'Suami' }
  function saksiTTL(s) {
    const t = [s.tempat_lahir, s.tgl_lahir].filter(Boolean).join(', ')
    return t || '—'
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
      <div class="mono muted">{berkas.reg_no_camat}</div>
    </div>
    <div class="flex gap">
      <a class="btn btn-ghost" href="#/">← Daftar</a>
      <a class="btn btn-primary" href={`/berkas/${id}/cetak`} target="_blank" rel="noopener">Cetak 3 Surat</a>
    </div>
  </div>

  <div class="alert alert-warn small">
    Data dasar berkas <strong>terkunci</strong> (read-only). Hanya <strong>Penerima Kuasa</strong>, <strong>data pelengkapnya</strong>, dan <strong>Item Kuasa</strong> yang dapat diubah.
  </div>

  <!-- Info berkas -->
  <div class="card">
    <h3>Informasi Berkas</h3>
    <div class="row row-3">
      <div><div class="muted small">Reg. No. Camat</div><div class="mono">{berkas.reg_no_camat}</div></div>
      <div><div class="muted small">Reg. No. Lurah</div><div class="mono">{berkas.reg_no_lurah}</div></div>
      <div><div class="muted small">Status</div><span class="badge badge-green">{berkas.status}</span></div>
    </div>
    <div class="row row-2 mt-2">
      <div><div class="muted small">Tanggal Surat</div><div>{berkas.tanggal_surat}</div></div>
      <div><div class="muted small">Tempat Tinggal Pewaris</div><div>{berkas.tempat_tinggal_pewaris}</div></div>
    </div>
  </div>

  <!-- Pewaris -->
  <div class="card">
    <h3>Pewaris (Almarhum/ah)</h3>
    <table>
      <thead><tr><th>Nama</th><th>Status</th><th>NIK</th><th>Tgl Meninggal</th><th>Instansi Kematian</th><th>No. Surat</th><th>Tgl Surat</th></tr></thead>
      <tbody>
        {#each berkas.pewaris as p}
          <tr><td>{p.nama}</td><td>{statusLabel(p.status)}</td><td class="mono">{p.nik}</td><td>{p.tgl_meninggal}</td><td>{p.instansi_kematian || '—'}</td><td>{p.no_surat_kematian}</td><td>{p.tgl_surat_kematian}</td></tr>
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
      <thead><tr><th>Nama</th><th>Tempat/Tgl Lahir</th><th>NIK</th><th>Alamat</th><th>Hubungan</th></tr></thead>
      <tbody>
        {#each berkas.saksi as s}
          <tr><td>{s.nama}</td><td>{saksiTTL(s)}</td><td class="mono">{s.nik || '—'}</td><td>{s.alamat || '—'}</td><td>{s.hubungan || '—'}</td></tr>
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

    {#if berkas.penerima_kuasa_ahli_waris_id}
      <div class="section-sub mt-2">Data pelengkap penerima kuasa (dipakai di Surat Kuasa)</div>
      <div class="row row-3">
        <div class="field mb-0"><label>Tempat Lahir</label><input bind:value={pelengkap.tempat_lahir} /></div>
        <div class="field mb-0"><label>Tgl Lahir</label><input bind:value={pelengkap.tgl_lahir} placeholder="mis. 08-12-1994" /></div>
        <div class="field mb-0"><label>Pekerjaan</label><input bind:value={pelengkap.pekerjaan} /></div>
      </div>
      <button class="btn btn-sm btn-primary mt-1" on:click={savePelengkap}>Simpan Data Pelengkap</button>
    {/if}

    <div class="divider"></div>

    <div class="card-title">
      <h3 class="mb-0" style="font-size:0.95rem;">Daftar Item Kuasa</h3>
    </div>
    {#if berkas.kuasa.length === 0}
      <div class="muted small">Belum ada item kuasa.</div>
    {/if}
    {#each berkas.kuasa as k}
      <div class="flex gap items-center mt-1">
        <textarea class="grow" rows="2" bind:value={kuasaEdits[k.id]}></textarea>
        <button class="btn btn-sm" on:click={() => saveKuasa(k)}>Simpan</button>
        <button class="btn btn-sm btn-danger" on:click={() => delKuasa(k)}>Hapus</button>
      </div>
    {/each}

    <div class="flex gap items-center mt-2">
      <textarea class="grow" rows="2" placeholder="Tambah item kuasa baru…" bind:value={newKuasa}></textarea>
      <button class="btn btn-primary btn-sm" on:click={addKuasa}>+ Tambah</button>
    </div>
  </div>
{/if}
