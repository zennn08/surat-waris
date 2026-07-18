<script>
  import { onMount } from 'svelte'
  import { api } from '../lib/api.js'
  import { navigate } from '../lib/router.js'
  import { notify } from '../lib/stores.js'
  import { fmtDate, digitsOnly } from '../lib/format.js'

  const AGAMA = ['Islam', 'Kristen', 'Katolik', 'Hindu', 'Buddha', 'Khonghucu']

  const today = new Date().toISOString().slice(0, 10)

  // Prasyarat: pejabat aktif (Camat & Lurah) + pengaturan lengkap.
  let ready = false
  let syarat = [] // {label, ok, href, aksi}
  let peng = null

  onMount(async () => {
    try {
      const [pejabat, pengaturan] = await Promise.all([api.get('/api/pejabat'), api.get('/api/pengaturan')])
      peng = pengaturan
      const camatOk = pejabat.some((p) => p.jabatan === 'camat' && p.aktif)
      const lurahOk = pejabat.some((p) => p.jabatan === 'lurah' && p.aktif)
      const pengOk = ['nama_kelurahan', 'kecamatan', 'kota', 'kode_kecamatan', 'kode_kelurahan', 'instansi_kematian']
        .every((k) => (pengaturan[k] || '').trim() !== '')
      syarat = [
        { label: 'Pejabat Camat (aktif) sudah diisi', ok: camatOk, href: '#/pejabat', aksi: 'Buka Halaman Pejabat' },
        { label: 'Pejabat Lurah (aktif) sudah diisi', ok: lurahOk, href: '#/pejabat', aksi: 'Buka Halaman Pejabat' },
        { label: 'Pengaturan wilayah sudah lengkap', ok: pengOk, href: '#/pengaturan', aksi: 'Buka Halaman Pengaturan' },
      ]
    } catch (e) {
      error = e.message
    } finally {
      ready = true
    }
  })
  $: siap = syarat.length > 0 && syarat.every((s) => s.ok)

  let tanggal_surat = today
  let tempat_tinggal_pewaris = ''

  let pewaris = [emptyPewaris('suami')]
  let ahli_waris = [emptyAhli()]
  let saksi = [emptySaksi(), emptySaksi()] // tepat 2
  let kuasa = ['']
  let penerima_kuasa_index = null

  let error = ''
  let busy = false

  // Wizard
  const STEPS = ['Data Berkas', 'Pewaris', 'Ahli Waris', 'Saksi', 'Surat Kuasa', 'Periksa & Simpan']
  let step = 1
  function goTo(n) {
    step = Math.min(Math.max(n, 1), STEPS.length)
    window.scrollTo(0, 0)
  }
  function next() { goTo(step + 1) }
  function back() { goTo(step - 1) }

  function emptyPewaris(status = 'suami') {
    return { nama: '', nik: '', status, tgl_meninggal: '', instansi_kematian: '', no_surat_kematian: '', tgl_surat_kematian: '' }
  }
  function emptyAhli() {
    return { nama: '', nik: '', umur: '', jenis_kelamin: 'L', agama: '', alamat: '', keterangan: '', tempat_lahir: '', tgl_lahir: '', pekerjaan: '' }
  }
  function emptySaksi() {
    return { nama: '', tempat_lahir: '', tgl_lahir: '', alamat: '', nik: '', hubungan: '' }
  }

  function addPewaris() { if (pewaris.length < 2) pewaris = [...pewaris, emptyPewaris('istri')] }
  function removePewaris(i) { if (pewaris.length > 1) pewaris = pewaris.filter((_, x) => x !== i) }

  function addAhli() { ahli_waris = [...ahli_waris, emptyAhli()] }
  function removeAhli(i) {
    if (ahli_waris.length <= 1) return
    ahli_waris = ahli_waris.filter((_, x) => x !== i)
    if (penerima_kuasa_index === i) penerima_kuasa_index = null
    else if (penerima_kuasa_index > i) penerima_kuasa_index -= 1
  }

  function addKuasa() { kuasa = [...kuasa, ''] }
  function removeKuasa(i) { kuasa = kuasa.filter((_, x) => x !== i) }

  // Isi data contoh (alat bantu testing). NIK diacak agar tidak kena lock saat tes berulang.
  function randNik() {
    let s = '1472010'
    for (let i = 0; i < 9; i++) s += Math.floor(Math.random() * 10)
    return s
  }
  function fillSample() {
    tanggal_surat = today
    tempat_tinggal_pewaris = 'Jl. Merdeka Baru RT.007'
    pewaris = [
      { nama: 'FAOZIDUHU TAFONAO', nik: randNik(), status: 'suami', tgl_meninggal: '2023-12-27', instansi_kematian: '', no_surat_kematian: '1472-KM-05012024-0005', tgl_surat_kematian: '2024-01-05' },
      { nama: 'SARITISA TAFONAO', nik: randNik(), status: 'istri', tgl_meninggal: '2024-02-10', instansi_kematian: '', no_surat_kematian: '1472-KM-9', tgl_surat_kematian: '2024-02-15' },
    ]
    ahli_waris = [
      { nama: 'ANGERAGO TAFONAO', nik: randNik(), umur: 31, jenis_kelamin: 'L', agama: 'Kristen', alamat: 'Jl. Sabar Menanti', keterangan: 'Anak', tempat_lahir: 'Doli-doli', tgl_lahir: '1994-12-08', pekerjaan: 'Pelajar/Mahasiswa' },
      { nama: 'ELViNA TAFONAO', nik: randNik(), umur: 27, jenis_kelamin: 'P', agama: 'Kristen', alamat: 'Jl. Sabar Menanti', keterangan: 'Anak', tempat_lahir: '', tgl_lahir: '', pekerjaan: '' },
    ]
    saksi = [
      { nama: 'Rahmat Hidayat', tempat_lahir: 'Bogor', tgl_lahir: '1970-05-12', alamat: 'Jl. Anggrek 3', nik: randNik(), hubungan: 'Tetangga' },
      { nama: 'Siti Aminah', tempat_lahir: 'Bogor', tgl_lahir: '1972-08-08', alamat: 'Jl. Anggrek 5', nik: randNik(), hubungan: 'Famili' },
    ]
    kuasa = ['Pengurusan administrasi kartu BPJS Ketenagakerjaan dengan Nomor 23137224459 an. Almarhummah SARITISA TAFONAO Kepada an. ANGERAGO TAFONAO (Anak)']
    penerima_kuasa_index = 0
  }

  async function submit() {
    error = ''
    busy = true
    try {
      const payload = {
        tanggal_surat,
        tempat_tinggal_pewaris,
        pewaris: pewaris.map((p) => ({ ...p })),
        ahli_waris: ahli_waris.map((a) => ({
          nama: a.nama, nik: a.nik,
          umur: a.umur === '' || a.umur === null ? null : Number(a.umur),
          jenis_kelamin: a.jenis_kelamin, agama: a.agama, alamat: a.alamat, keterangan: a.keterangan,
          tempat_lahir: a.tempat_lahir, tgl_lahir: a.tgl_lahir, pekerjaan: a.pekerjaan,
        })),
        saksi: saksi.map((s) => ({ ...s })),
        penerima_kuasa_index: penerima_kuasa_index === null ? null : Number(penerima_kuasa_index),
        kuasa: kuasa.map((k) => k.trim()).filter(Boolean),
      }
      const created = await api.post('/api/berkas', payload)
      notify('Berkas berhasil dibuat: ' + created.reg_no_camat, 'success')
      navigate('/berkas/' + created.id)
    } catch (e) {
      error = e.message
      window.scrollTo(0, 0)
    } finally {
      busy = false
    }
  }

  function namaAtau(a, i) { return a.nama ? a.nama : 'Ahli Waris ' + (i + 1) }
</script>

<div class="card-title">
  <h1 class="mb-0">Buat Berkas Waris</h1>
  <div class="flex gap">
    {#if siap}<button type="button" class="btn btn-sm" on:click={fillSample}>Isi Data Contoh</button>{/if}
    <a class="btn btn-ghost" href="#/">Batal</a>
  </div>
</div>

{#if !ready}
  <div class="spinner">Memuat…</div>
{:else if !siap}
  {#if error}<div class="alert alert-error">{error}</div>{/if}
  <div class="card">
    <h2>Lengkapi Dulu Sebelum Membuat Berkas</h2>
    <div class="section-sub">
      Data berikut dipakai pada isi surat dan nomor registrasi, jadi wajib diisi
      sekali di awal. Setelah lengkap, buka kembali halaman ini.
    </div>
    {#each syarat as s}
      <div class="review-row">
        <div class="rv-value">
          {#if s.ok}<span class="badge badge-green">✓ Sudah</span>{:else}<span class="badge badge-gray">Belum</span>{/if}
          &nbsp;{s.label}
        </div>
        {#if !s.ok}<a class="btn btn-primary" href={s.href}>{s.aksi}</a>{/if}
      </div>
    {/each}
  </div>
{:else}
<p class="page-sub">
  Ikuti langkah satu per satu. Isian Anda tidak hilang saat berpindah langkah,
  dan semuanya bisa diperiksa lagi di langkah terakhir sebelum disimpan.
</p>

<!-- Indikator langkah -->
<div class="wiz-steps" role="navigation" aria-label="Langkah pengisian">
  {#each STEPS as label, i}
    <button
      type="button"
      class="wiz-step"
      class:done={step > i + 1}
      class:active={step === i + 1}
      class:clickable={step > i + 1}
      disabled={step <= i + 1}
      on:click={() => step > i + 1 && goTo(i + 1)}
    >
      <span class="num">{step > i + 1 ? '✓' : i + 1}</span>
      <span class="lbl">{label}</span>
    </button>
  {/each}
</div>

{#if error}<div class="alert alert-error">{error}</div>{/if}

<!-- Langkah 1: Data Berkas -->
{#if step === 1}
  <form on:submit|preventDefault={next}>
    <div class="card">
      <h2>Langkah 1: Data Berkas</h2>
      <div class="section-sub">Tanggal yang tercetak pada surat dan alamat tinggal terakhir almarhum/ah.</div>
      <div class="row row-2">
        <div class="field">
          <label for="tgl">Tanggal Surat</label>
          <input id="tgl" type="date" bind:value={tanggal_surat} required />
        </div>
        <div class="field">
          <label for="tt">Tempat Tinggal Terakhir Pewaris</label>
          <input id="tt" bind:value={tempat_tinggal_pewaris} placeholder="contoh: Jl. Merdeka Baru RT.007" required />
        </div>
      </div>
    </div>
    <div class="wiz-actions">
      <span></span>
      <button class="btn btn-primary btn-lg">Lanjut →</button>
    </div>
  </form>
{/if}

<!-- Langkah 2: Pewaris -->
{#if step === 2}
  <form on:submit|preventDefault={next}>
    <div class="card">
      <h2>Langkah 2: Pewaris (yang meninggal)</h2>
      <div class="section-sub">Orang yang meninggal dan mewariskan. Bisa 1 orang, atau 2 bila pasangan suami-istri sudah sama-sama meninggal.</div>
      {#each pewaris as p, i}
        <div class="item-card">
          <div class="item-head">
            <strong>Pewaris {i + 1}</strong>
            {#if pewaris.length > 1}<button type="button" class="btn btn-sm btn-danger" on:click={() => removePewaris(i)}>Hapus</button>{/if}
          </div>
          <div class="row row-3">
            <div class="field"><label>Nama Lengkap</label><input bind:value={p.nama} required /></div>
            <div class="field"><label>NIK</label><input bind:value={p.nik} use:digitsOnly class="mono" inputmode="numeric" maxlength="16" pattern={'[0-9]{16}'} title="NIK harus 16 digit angka" required /></div>
            <div class="field"><label>Status</label>
              <select bind:value={p.status}><option value="suami">Suami</option><option value="istri">Istri</option></select>
            </div>
          </div>
          <div class="row row-2">
            <div class="field"><label>Tanggal Meninggal</label><input type="date" bind:value={p.tgl_meninggal} required /></div>
            <div class="field">
              <label>Instansi Penerbit Surat Kematian</label>
              <input bind:value={p.instansi_kematian} placeholder="boleh dikosongkan" />
              <div class="help">Bila kosong, otomatis diisi: {peng?.instansi_kematian || 'instansi dari halaman Pengaturan'}.</div>
            </div>
          </div>
          <div class="row row-2">
            <div class="field"><label>No. Surat Kematian</label><input bind:value={p.no_surat_kematian} required /></div>
            <div class="field">
              <label>Tanggal Surat Kematian</label>
              <input type="date" bind:value={p.tgl_surat_kematian} min={p.tgl_meninggal || undefined} required />
              <div class="help">Tidak boleh lebih awal dari tanggal meninggal.</div>
            </div>
          </div>
        </div>
      {/each}
      {#if pewaris.length < 2}
        <button type="button" class="btn" on:click={addPewaris}>+ Tambah Pewaris (pasangan)</button>
      {/if}
    </div>
    <div class="wiz-actions">
      <button type="button" class="btn btn-lg" on:click={back}>← Kembali</button>
      <button class="btn btn-primary btn-lg">Lanjut →</button>
    </div>
  </form>
{/if}

<!-- Langkah 3: Ahli Waris -->
{#if step === 3}
  <form on:submit|preventDefault={next}>
    <div class="card">
      <h2>Langkah 3: Ahli Waris</h2>
      <div class="section-sub">Semua penerima waris, sesuai urutan yang akan tercetak di surat.</div>
      {#each ahli_waris as a, i}
        <div class="item-card">
          <div class="item-head">
            <strong>Ahli Waris {i + 1}</strong>
            {#if ahli_waris.length > 1}<button type="button" class="btn btn-sm btn-danger" on:click={() => removeAhli(i)}>Hapus</button>{/if}
          </div>
          <div class="row row-2">
            <div class="field"><label>Nama Lengkap</label><input bind:value={a.nama} required /></div>
            <div class="field"><label>NIK</label><input bind:value={a.nik} use:digitsOnly class="mono" inputmode="numeric" maxlength="16" pattern={'[0-9]{16}'} title="NIK harus 16 digit angka" required /></div>
          </div>
          <div class="row row-3">
            <div class="field"><label>Umur</label><input type="number" min="0" bind:value={a.umur} /></div>
            <div class="field"><label>Jenis Kelamin</label>
              <select bind:value={a.jenis_kelamin}><option value="L">Laki-laki</option><option value="P">Perempuan</option></select>
            </div>
            <div class="field"><label>Agama</label>
              <select bind:value={a.agama}>
                <option value="">— Pilih —</option>
                {#each AGAMA as ag}<option value={ag}>{ag}</option>{/each}
              </select>
            </div>
          </div>
          <div class="row row-2">
            <div class="field"><label>Alamat</label><input bind:value={a.alamat} /></div>
            <div class="field"><label>Hubungan dengan Pewaris</label><input bind:value={a.keterangan} placeholder="contoh: Anak" /></div>
          </div>
        </div>
      {/each}
      <button type="button" class="btn" on:click={addAhli}>+ Tambah Ahli Waris</button>
    </div>
    <div class="wiz-actions">
      <button type="button" class="btn btn-lg" on:click={back}>← Kembali</button>
      <button class="btn btn-primary btn-lg">Lanjut →</button>
    </div>
  </form>
{/if}

<!-- Langkah 4: Saksi -->
{#if step === 4}
  <form on:submit|preventDefault={next}>
    <div class="card">
      <h2>Langkah 4: Saksi</h2>
      <div class="section-sub">Tepat 2 orang yang menyaksikan dan ikut menandatangani surat.</div>
      {#each saksi as s, i}
        <div class="item-card">
          <div class="item-head"><strong>Saksi {i + 1}</strong></div>
          <div class="row row-3">
            <div class="field"><label>Nama Lengkap</label><input bind:value={s.nama} required /></div>
            <div class="field"><label>Tempat Lahir</label><input bind:value={s.tempat_lahir} /></div>
            <div class="field"><label>Tanggal Lahir</label><input type="date" bind:value={s.tgl_lahir} /></div>
          </div>
          <div class="row row-3">
            <div class="field"><label>NIK</label><input bind:value={s.nik} use:digitsOnly class="mono" inputmode="numeric" maxlength="16" pattern={'[0-9]{16}'} title="NIK harus 16 digit angka (boleh dikosongkan)" /></div>
            <div class="field"><label>Alamat</label><input bind:value={s.alamat} /></div>
            <div class="field"><label>Hubungan dengan Almarhum/ah</label><input bind:value={s.hubungan} placeholder="contoh: Tetangga" /></div>
          </div>
        </div>
      {/each}
    </div>
    <div class="wiz-actions">
      <button type="button" class="btn btn-lg" on:click={back}>← Kembali</button>
      <button class="btn btn-primary btn-lg">Lanjut →</button>
    </div>
  </form>
{/if}

<!-- Langkah 5: Surat Kuasa -->
{#if step === 5}
  <form on:submit|preventDefault={next}>
    <div class="card">
      <h2>Langkah 5: Surat Kuasa</h2>
      <div class="section-sub">
        Satu ahli waris ditunjuk sebagai penerima kuasa; ahli waris lainnya otomatis menjadi pemberi kuasa.
        Bagian ini masih bisa diubah setelah berkas disimpan.
      </div>

      <div class="field" style="max-width:460px;">
        <label>Penerima Kuasa</label>
        <select bind:value={penerima_kuasa_index}>
          <option value={null}>— Belum dipilih —</option>
          {#each ahli_waris as a, i}
            <option value={i}>{namaAtau(a, i)}</option>
          {/each}
        </select>
      </div>

      {#if penerima_kuasa_index !== null}
        <div class="section-sub">Data pelengkap penerima kuasa — tercetak pada Surat Kuasa.</div>
        <div class="row row-3">
          <div class="field"><label>Tempat Lahir</label><input bind:value={ahli_waris[penerima_kuasa_index].tempat_lahir} /></div>
          <div class="field"><label>Tanggal Lahir</label><input type="date" bind:value={ahli_waris[penerima_kuasa_index].tgl_lahir} /></div>
          <div class="field"><label>Pekerjaan</label><input bind:value={ahli_waris[penerima_kuasa_index].pekerjaan} /></div>
        </div>
      {/if}

      <div class="divider"></div>

      <div class="card-title">
        <div>
          <h3 class="mb-0">Urusan yang Dikuasakan</h3>
          <div class="section-sub mb-0">Tuliskan tiap urusan apa adanya, seperti akan tercetak di surat.</div>
        </div>
        <button type="button" class="btn btn-sm" on:click={addKuasa}>+ Tambah Urusan</button>
      </div>
      {#if kuasa.length === 0}<div class="muted small">Belum ada urusan yang dikuasakan.</div>{/if}
      {#each kuasa as _, i}
        <div class="flex gap items-center mt-1">
          <textarea class="grow" rows="2" placeholder="contoh: Pengurusan administrasi kartu BPJS Ketenagakerjaan Nomor … an. …" bind:value={kuasa[i]}></textarea>
          <button type="button" class="btn btn-sm btn-danger" on:click={() => removeKuasa(i)}>Hapus</button>
        </div>
      {/each}
    </div>
    <div class="wiz-actions">
      <button type="button" class="btn btn-lg" on:click={back}>← Kembali</button>
      <button class="btn btn-primary btn-lg">Lanjut →</button>
    </div>
  </form>
{/if}

<!-- Langkah 6: Periksa & Simpan -->
{#if step === 6}
  <div class="card">
    <h2>Langkah 6: Periksa Kembali</h2>
    <div class="section-sub">Pastikan semua data benar. Klik “Ubah” untuk kembali ke langkah terkait.</div>

    <div class="review-row">
      <div class="rv-label">Data Berkas</div>
      <div class="rv-value">Tanggal surat {fmtDate(tanggal_surat)} · Tempat tinggal: {tempat_tinggal_pewaris || '—'}</div>
      <button type="button" class="btn btn-sm" on:click={() => goTo(1)}>Ubah</button>
    </div>
    <div class="review-row">
      <div class="rv-label">Pewaris</div>
      <div class="rv-value">
        {#each pewaris as p, i}
          <div>{i + 1}. {p.nama || '—'} ({p.status === 'istri' ? 'Istri' : 'Suami'}) — meninggal {fmtDate(p.tgl_meninggal)}</div>
        {/each}
      </div>
      <button type="button" class="btn btn-sm" on:click={() => goTo(2)}>Ubah</button>
    </div>
    <div class="review-row">
      <div class="rv-label">Ahli Waris</div>
      <div class="rv-value">
        {#each ahli_waris as a, i}
          <div>
            {i + 1}. {a.nama || '—'}{a.keterangan ? ` (${a.keterangan})` : ''}
            {#if penerima_kuasa_index === i}&nbsp;<span class="badge badge-blue">Penerima Kuasa</span>{/if}
          </div>
        {/each}
      </div>
      <button type="button" class="btn btn-sm" on:click={() => goTo(3)}>Ubah</button>
    </div>
    <div class="review-row">
      <div class="rv-label">Saksi</div>
      <div class="rv-value">
        {#each saksi as s, i}<div>{i + 1}. {s.nama || '—'}{s.hubungan ? ` (${s.hubungan})` : ''}</div>{/each}
      </div>
      <button type="button" class="btn btn-sm" on:click={() => goTo(4)}>Ubah</button>
    </div>
    <div class="review-row">
      <div class="rv-label">Surat Kuasa</div>
      <div class="rv-value">
        {#if penerima_kuasa_index === null}
          <div>Penerima kuasa belum dipilih.</div>
        {:else}
          <div>Penerima kuasa: {namaAtau(ahli_waris[penerima_kuasa_index], penerima_kuasa_index)}</div>
        {/if}
        {#each kuasa.filter((k) => k.trim()) as k, i}<div>{i + 1}. {k}</div>{/each}
      </div>
      <button type="button" class="btn btn-sm" on:click={() => goTo(5)}>Ubah</button>
    </div>
  </div>

  <div class="notice mt-2">
    Setelah disimpan, nomor registrasi Camat &amp; Lurah langsung terbit dan data
    <strong>tidak bisa diubah lagi</strong> — kecuali bagian Surat Kuasa.
  </div>

  <div class="wiz-actions">
    <button type="button" class="btn btn-lg" on:click={back}>← Kembali</button>
    <button type="button" class="btn btn-primary btn-lg" disabled={busy} on:click={submit}>
      {busy ? 'Menyimpan…' : 'Simpan Berkas'}
    </button>
  </div>
{/if}
{/if}
