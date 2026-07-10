<script>
  import { api } from '../lib/api.js'
  import { navigate } from '../lib/router.js'
  import { notify } from '../lib/stores.js'

  const today = new Date().toISOString().slice(0, 10)

  let tanggal = today
  let tempat_tinggal_pewaris = ''

  let pewaris = [emptyPewaris()]
  let ahli_waris = [emptyAhli()]
  let saksi = [emptySaksi(), emptySaksi()] // tepat 2
  let harta = ['']
  let penerima_kuasa_index = null

  let error = ''
  let busy = false

  function emptyPewaris() {
    return { nama: '', nik: '', tgl_meninggal: '', no_surat_kematian: '', tgl_surat_kematian: '' }
  }
  function emptyAhli() {
    return { nama: '', nik: '', umur: '', jenis_kelamin: 'L', agama: '', alamat: '', keterangan: '' }
  }
  function emptySaksi() {
    return { nama: '', ttl: '', alamat: '', nik: '', hubungan: '' }
  }

  function addPewaris() { if (pewaris.length < 2) pewaris = [...pewaris, emptyPewaris()] }
  function removePewaris(i) { if (pewaris.length > 1) pewaris = pewaris.filter((_, x) => x !== i) }

  function addAhli() { ahli_waris = [...ahli_waris, emptyAhli()] }
  function removeAhli(i) {
    if (ahli_waris.length <= 1) return
    ahli_waris = ahli_waris.filter((_, x) => x !== i)
    if (penerima_kuasa_index === i) penerima_kuasa_index = null
    else if (penerima_kuasa_index > i) penerima_kuasa_index -= 1
  }

  function addHarta() { harta = [...harta, ''] }
  function removeHarta(i) { harta = harta.filter((_, x) => x !== i) }

  // Isi data contoh (alat bantu testing). NIK diacak agar tidak kena lock saat tes berulang.
  function randNik() {
    let s = '32010'
    for (let i = 0; i < 11; i++) s += Math.floor(Math.random() * 10)
    return s
  }
  function fillSample() {
    tanggal = today
    tempat_tinggal_pewaris = 'Jl. Melati No. 10, RT 002/003, Kel. Sukamaju'
    pewaris = [{
      nama: 'Alm. Sutrisno bin Karta', nik: randNik(),
      tgl_meninggal: '2025-03-15', no_surat_kematian: '474.3/012/2025', tgl_surat_kematian: '2025-03-20',
    }]
    ahli_waris = [
      { nama: 'Andi Sutrisno', nik: randNik(), umur: 30, jenis_kelamin: 'L', agama: 'Islam', alamat: 'Jl. Melati No. 10', keterangan: 'Anak Kandung' },
      { nama: 'Bunga Sutrisno', nik: randNik(), umur: 27, jenis_kelamin: 'P', agama: 'Islam', alamat: 'Jl. Melati No. 10', keterangan: 'Anak Kandung' },
    ]
    saksi = [
      { nama: 'Rahmat Hidayat', ttl: 'Bogor, 12 Mei 1970', alamat: 'Jl. Anggrek 3', nik: randNik(), hubungan: 'Ketua RT' },
      { nama: 'Siti Aminah', ttl: 'Bogor, 8 Agustus 1972', alamat: 'Jl. Anggrek 5', nik: randNik(), hubungan: 'Ketua RW' },
    ]
    harta = ['Tanah SHM No. 123 luas 200 m²', 'Sepeda motor Honda B 1234 XYZ']
    penerima_kuasa_index = 0
  }

  async function submit() {
    error = ''
    busy = true
    try {
      const payload = {
        tanggal,
        tempat_tinggal_pewaris,
        pewaris: pewaris.map((p) => ({ ...p })),
        ahli_waris: ahli_waris.map((a) => ({
          nama: a.nama, nik: a.nik,
          umur: a.umur === '' || a.umur === null ? null : Number(a.umur),
          jenis_kelamin: a.jenis_kelamin, agama: a.agama, alamat: a.alamat, keterangan: a.keterangan,
        })),
        saksi: saksi.map((s) => ({ ...s })),
        penerima_kuasa_index: penerima_kuasa_index === null ? null : Number(penerima_kuasa_index),
        harta: harta.map((h) => h.trim()).filter(Boolean),
      }
      const created = await api.post('/api/berkas', payload)
      notify('Berkas berhasil dibuat: ' + created.nomor_surat, 'success')
      navigate('/berkas/' + created.id)
    } catch (e) {
      error = e.message
    } finally {
      busy = false
    }
  }
</script>

<div class="card-title">
  <h1 class="mb-0">Buat Berkas Waris</h1>
  <div class="flex gap">
    <button type="button" class="btn btn-sm" on:click={fillSample}>Isi Data Contoh</button>
    <a class="btn btn-ghost" href="#/">← Kembali</a>
  </div>
</div>

{#if error}<div class="alert alert-error">{error}</div>{/if}

<form on:submit|preventDefault={submit}>
  <!-- Data berkas -->
  <div class="card">
    <h3>Data Berkas</h3>
    <div class="row row-2">
      <div class="field">
        <label for="tgl">Tanggal Surat</label>
        <input id="tgl" type="date" bind:value={tanggal} required />
      </div>
      <div class="field">
        <label for="tt">Tempat Tinggal Pewaris</label>
        <input id="tt" bind:value={tempat_tinggal_pewaris} placeholder="Alamat tempat tinggal pewaris" required />
      </div>
    </div>
  </div>

  <!-- Pewaris -->
  <div class="card">
    <div class="card-title">
      <div><h3 class="mb-0">Pewaris (Almarhum/ah)</h3><div class="section-sub">Minimal 1, maksimal 2 (mis. suami-istri)</div></div>
      <button type="button" class="btn btn-sm" on:click={addPewaris} disabled={pewaris.length >= 2}>+ Tambah</button>
    </div>
    {#each pewaris as p, i}
      <div class="item-card">
        <div class="item-head">
          <strong>Pewaris {i + 1}</strong>
          {#if pewaris.length > 1}<button type="button" class="btn btn-sm btn-danger" on:click={() => removePewaris(i)}>Hapus</button>{/if}
        </div>
        <div class="row row-2">
          <div class="field"><label>Nama</label><input bind:value={p.nama} required /></div>
          <div class="field"><label>NIK</label><input bind:value={p.nik} class="mono" required /></div>
        </div>
        <div class="row row-3">
          <div class="field"><label>Tgl Meninggal</label><input type="date" bind:value={p.tgl_meninggal} required /></div>
          <div class="field"><label>No. Surat Kematian</label><input bind:value={p.no_surat_kematian} required /></div>
          <div class="field"><label>Tgl Surat Kematian</label><input type="date" bind:value={p.tgl_surat_kematian} required /></div>
        </div>
      </div>
    {/each}
  </div>

  <!-- Ahli Waris -->
  <div class="card">
    <div class="card-title">
      <div><h3 class="mb-0">Ahli Waris</h3><div class="section-sub">Tambah sesuai jumlah ahli waris</div></div>
      <button type="button" class="btn btn-sm" on:click={addAhli}>+ Tambah</button>
    </div>
    {#each ahli_waris as a, i}
      <div class="item-card">
        <div class="item-head">
          <strong>Ahli Waris {i + 1}</strong>
          {#if ahli_waris.length > 1}<button type="button" class="btn btn-sm btn-danger" on:click={() => removeAhli(i)}>Hapus</button>{/if}
        </div>
        <div class="row row-2">
          <div class="field"><label>Nama</label><input bind:value={a.nama} required /></div>
          <div class="field"><label>NIK</label><input bind:value={a.nik} class="mono" required /></div>
        </div>
        <div class="row row-3">
          <div class="field"><label>Umur</label><input type="number" min="0" bind:value={a.umur} /></div>
          <div class="field"><label>Jenis Kelamin</label>
            <select bind:value={a.jenis_kelamin}><option value="L">Laki-laki</option><option value="P">Perempuan</option></select>
          </div>
          <div class="field"><label>Agama</label><input bind:value={a.agama} /></div>
        </div>
        <div class="row row-2">
          <div class="field"><label>Alamat</label><input bind:value={a.alamat} /></div>
          <div class="field"><label>Keterangan (mis. Anak, Istri)</label><input bind:value={a.keterangan} /></div>
        </div>
      </div>
    {/each}
  </div>

  <!-- Saksi -->
  <div class="card">
    <h3>Saksi</h3>
    <div class="section-sub">Tepat 2 orang</div>
    {#each saksi as s, i}
      <div class="item-card">
        <div class="item-head"><strong>Saksi {i + 1}</strong></div>
        <div class="row row-2">
          <div class="field"><label>Nama</label><input bind:value={s.nama} required /></div>
          <div class="field"><label>TTL (Tempat, Tgl Lahir)</label><input bind:value={s.ttl} /></div>
        </div>
        <div class="row row-3">
          <div class="field"><label>NIK</label><input bind:value={s.nik} class="mono" /></div>
          <div class="field"><label>Alamat</label><input bind:value={s.alamat} /></div>
          <div class="field"><label>Hubungan dgn Alm.</label><input bind:value={s.hubungan} /></div>
        </div>
      </div>
    {/each}
  </div>

  <!-- Surat Kuasa -->
  <div class="card">
    <h3>Bagian Surat Kuasa</h3>
    <div class="section-sub">Penerima kuasa dipilih dari ahli waris; harta = yang dikuasakan (bisa diedit nanti)</div>

    <div class="field">
      <label>Penerima Kuasa (diberi kuasa oleh ahli waris lain)</label>
      <select bind:value={penerima_kuasa_index}>
        <option value={null}>— Belum dipilih —</option>
        {#each ahli_waris as a, i}
          <option value={i}>{a.nama ? a.nama : 'Ahli Waris ' + (i + 1)}</option>
        {/each}
      </select>
    </div>

    <div class="divider"></div>

    <div class="card-title">
      <h3 class="mb-0" style="font-size:0.95rem;">Daftar Harta / Yang Dikuasakan</h3>
      <button type="button" class="btn btn-sm" on:click={addHarta}>+ Tambah Harta</button>
    </div>
    {#if harta.length === 0}<div class="muted small">Belum ada harta.</div>{/if}
    {#each harta as _, i}
      <div class="flex gap items-center mt-1">
        <input class="grow" placeholder="mis. Tanah SHM No. 123, luas 200 m²" bind:value={harta[i]} />
        <button type="button" class="btn btn-sm btn-danger" on:click={() => removeHarta(i)}>Hapus</button>
      </div>
    {/each}
  </div>

  <div class="flex between mt-2">
    <a class="btn btn-ghost" href="#/">Batal</a>
    <button class="btn btn-primary" disabled={busy}>{busy ? 'Menyimpan…' : 'Simpan Berkas & Buat Nomor'}</button>
  </div>
</form>
