<script>
  import { onMount } from 'svelte'
  import { api } from '../lib/api.js'
  import { notify } from '../lib/stores.js'
  import ChangePassword from './ChangePassword.svelte'
  import ConfirmDialog from '../lib/ConfirmDialog.svelte'

  let form = { nama_kelurahan: '', kecamatan: '', kota: '', kode_kecamatan: '', kode_kelurahan: '', instansi_kematian: '' }
  let loading = true
  let showPw = false

  // Nomor urut awal per tahun
  const thisYear = new Date().getFullYear()
  let nomorAwal = []
  let naForm = { tahun: thisYear, urutan_awal: 0 }

  async function load() {
    loading = true
    try {
      form = await api.get('/api/pengaturan')
      nomorAwal = await api.get('/api/nomor-awal')
    } catch (e) { notify(e.message, 'error') } finally { loading = false }
  }

  async function save() {
    try { await api.put('/api/pengaturan', form); notify('Pengaturan disimpan', 'success') }
    catch (e) { notify(e.message, 'error') }
  }

  async function saveNomorAwal() {
    try {
      await api.put('/api/nomor-awal', { tahun: Number(naForm.tahun), urutan_awal: Number(naForm.urutan_awal) })
      nomorAwal = await api.get('/api/nomor-awal')
      notify('Nomor urut awal disimpan', 'success')
    } catch (e) { notify(e.message, 'error') }
  }

  // Cadangan & pindah data
  let dlgImpor
  let fileInput
  let importing = false
  async function onPickImport(e) {
    const file = e.target.files[0]
    e.target.value = ''
    if (!file) return
    const ok = await dlgImpor.ask(
      `Seluruh data saat ini akan diganti dengan isi "${file.name}". ` +
      'Data lama otomatis dicadangkan ke file di samping aplikasi, dan Anda akan diminta login ulang.'
    )
    if (!ok) return
    importing = true
    try {
      const fd = new FormData()
      fd.append('file', file)
      const res = await fetch('/api/database/import', { method: 'POST', credentials: 'include', body: fd })
      const data = await res.json().catch(() => null)
      if (!res.ok) throw new Error((data && data.error) || `Gagal (HTTP ${res.status})`)
      notify('Impor berhasil. Memuat ulang aplikasi...', 'success')
      setTimeout(() => location.reload(), 1500)
    } catch (err) {
      notify(err.message, 'error')
      importing = false
    }
  }

  let dlgHapus
  async function delNomorAwal(tahun) {
    if (!(await dlgHapus.ask(`Setelan nomor urut awal tahun ${tahun} akan dihapus. Penomoran tahun itu kembali mulai dari 1.`))) return
    try {
      await api.del('/api/nomor-awal/' + tahun)
      nomorAwal = await api.get('/api/nomor-awal')
      notify('Dihapus', 'success')
    } catch (e) { notify(e.message, 'error') }
  }

  onMount(load)
</script>

<h1>Pengaturan</h1>
<div class="section-sub">Identitas kelurahan dipakai di isi surat & penomoran (bukan kop/logo).</div>

{#if loading}
  <div class="spinner">Memuat…</div>
{:else}
  <div class="card">
    <h3>Identitas Kelurahan</h3>
    <form on:submit|preventDefault={save}>
      <div class="row row-2">
        <div class="field"><label>Nama Kelurahan</label><input bind:value={form.nama_kelurahan} placeholder="mis. Teluk Binjai" /></div>
        <div class="field"><label>Kecamatan</label><input bind:value={form.kecamatan} placeholder="mis. Dumai Timur" /></div>
      </div>
      <div class="row row-3">
        <div class="field"><label>Kota</label><input bind:value={form.kota} placeholder="mis. Dumai" /></div>
        <div class="field"><label>Kode Kecamatan</label><input class="mono" bind:value={form.kode_kecamatan} placeholder="mis. DT" /></div>
        <div class="field"><label>Kode Kelurahan</label><input class="mono" bind:value={form.kode_kelurahan} placeholder="mis. TB" /></div>
      </div>
      <div class="field">
        <label>Instansi Penerbit Surat Kematian (default)</label>
        <input bind:value={form.instansi_kematian} placeholder="mis. Dinas Kependudukan dan Pencatatan Sipil Kota Dumai" />
      </div>
      <div class="small muted mt-1">
        Reg. No. Camat = <span class="mono">&#123;urutan&#125;/SKAW/&#123;kode_kecamatan&#125;/&#123;tahun&#125;</span> ·
        Reg. No. Lurah = <span class="mono">&#123;urutan&#125;/SKAW/&#123;kode_kelurahan&#125;-&#123;kode_kecamatan&#125;/&#123;tahun&#125;</span>
      </div>
      <button class="btn btn-primary mt-1">Simpan Pengaturan</button>
    </form>
  </div>

  <div class="card mt-2">
    <h3>Nomor Urut Awal per Tahun</h3>
    <div class="section-sub">
      Untuk migrasi dari manual ke digital: isi <strong>nomor terakhir</strong> yang sudah dipakai manual pada tahun tsb.
      Berkas digital berikutnya otomatis lanjut dari nomor+1. Menurunkan nilai tidak akan memundurkan nomor yang sudah terpakai.
    </div>
    <form on:submit|preventDefault={saveNomorAwal}>
      <div class="row row-3" style="align-items:end;">
        <div class="field mb-0"><label>Tahun</label><input type="number" min="2000" bind:value={naForm.tahun} /></div>
        <div class="field mb-0"><label>Nomor Terakhir (manual)</label><input type="number" min="0" bind:value={naForm.urutan_awal} /></div>
        <div class="field mb-0"><button class="btn btn-primary">Simpan</button></div>
      </div>
    </form>

    {#if nomorAwal.length > 0}
      <div class="table-wrap mt-2">
      <table>
        <thead><tr><th>Tahun</th><th>Nomor Terakhir Manual</th><th>Digital Mulai Dari</th><th style="width:80px;"></th></tr></thead>
        <tbody>
          {#each nomorAwal as n}
            <tr>
              <td>{n.tahun}</td>
              <td>{n.urutan_awal}</td>
              <td class="mono">{n.urutan_awal + 1}</td>
              <td class="right"><button class="btn btn-sm btn-danger" on:click={() => delNomorAwal(n.tahun)}>Hapus</button></td>
            </tr>
          {/each}
        </tbody>
      </table>
      </div>
    {:else}
      <div class="muted small mt-2">Belum ada setelan. Tanpa setelan, penomoran mulai dari 1 tiap tahun.</div>
    {/if}
  </div>

  <div class="card mt-2">
    <h3>Cadangan &amp; Pindah Data</h3>
    <div class="section-sub">
      Untuk pindah komputer atau berjaga-jaga: unduh seluruh data sebagai satu file cadangan,
      lalu impor file itu di aplikasi tujuan.
    </div>
    <div class="flex gap" style="flex-wrap:wrap;">
      <a class="btn btn-primary" href="/api/database/export" download>Unduh Cadangan (.db)</a>
      <input type="file" accept=".db" bind:this={fileInput} style="display:none;" on:change={onPickImport} />
      <button class="btn" disabled={importing} on:click={() => fileInput.click()}>
        {importing ? 'Mengimpor...' : 'Impor dari File Cadangan'}
      </button>
    </div>
    <div class="small muted mt-1">
      Impor mengganti seluruh data saat ini (berkas, pejabat, pengaturan, akun).
      Data lama otomatis dicadangkan dulu ke file di samping aplikasi.
    </div>
  </div>

  <div class="card mt-2">
    <div class="card-title">
      <h3 class="mb-0">Keamanan Akun</h3>
      <button class="btn btn-sm" on:click={() => (showPw = !showPw)}>{showPw ? 'Tutup' : 'Ganti Password'}</button>
    </div>
    {#if showPw}
      <div style="max-width:420px;"><ChangePassword forced={false} /></div>
    {/if}
  </div>

  <ConfirmDialog bind:this={dlgHapus} title="Hapus setelan nomor awal?" confirmLabel="Ya, Hapus" danger />
  <ConfirmDialog bind:this={dlgImpor} title="Impor data dari cadangan?" confirmLabel="Ya, Impor & Ganti Data" danger />
{/if}
