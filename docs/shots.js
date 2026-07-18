// Ambil screenshot alur lengkap SIWARIS untuk dokumentasi.
// Jalankan: yarn node shots.js <baseURL> <outDir>
const { chromium } = require('playwright-core')

const BASE = process.argv[2]
const OUT = process.argv[3]
const fs = require('fs')
fs.mkdirSync(OUT, { recursive: true })

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

;(async () => {
  const browser = await chromium.launch({ channel: 'msedge', headless: true })
  const page = await browser.newPage({ viewport: { width: 1200, height: 850 }, deviceScaleFactor: 1.25 })

  const shot = async (name, opts = {}) => {
    await sleep(400)
    await page.screenshot({ path: `${OUT}/${name}.png`, ...opts })
    console.log('OK', name)
  }
  // Cari kolom isian berdasarkan teks label persis
  const field = (label) =>
    page.locator('.field').filter({ has: page.locator('label', { hasText: new RegExp('^' + label + '$') }) })
      .locator('input, select, textarea').first()

  // 1. Login
  await page.goto(BASE)
  await page.waitForSelector('#u')
  await page.fill('#u', 'admin')
  await page.fill('#p', 'admin123')
  await shot('01-login')
  await page.click('button:has-text("Masuk")')

  // 2. Wajib ganti password
  await page.waitForSelector('#op')
  await page.fill('#op', 'admin123')
  await page.fill('#np', 'passwordbaru')
  await page.fill('#cp', 'passwordbaru')
  await shot('02-ganti-password')
  await page.click('button:has-text("Simpan Password")')

  // 3. Daftar berkas kosong
  await page.waitForSelector('h1:has-text("Daftar Berkas Waris")')
  await sleep(4300) // biarkan toast hilang
  await shot('03-daftar-kosong')

  // 4. Prasyarat belum lengkap
  await page.click('.nav a:has-text("Buat Berkas")')
  await page.waitForSelector('h2:has-text("Lengkapi Dulu")')
  await shot('04-prasyarat')

  // 5. Isi pejabat: Camat lalu Lurah
  await page.click('a.btn:has-text("Buka Halaman Pejabat")')
  await page.waitForSelector('h1:has-text("Pejabat Penandatangan")')
  await field('Jabatan').selectOption('camat')
  await field('Nama').fill('NAMA CAMAT, S.STP')
  await field('NIP').fill('197001011990011001')
  await page.click('button:has-text("Tambah")')
  await page.waitForSelector('td:has-text("Camat")')
  await field('Jabatan').selectOption('lurah')
  await field('Nama').fill('NAMA LURAH, S.Sos')
  await field('NIP').fill('198001011999031002')
  await shot('05-pejabat-form')
  await page.click('button:has-text("Tambah")')
  await page.waitForSelector('td:has-text("Lurah")')
  await sleep(4300)
  await shot('06-pejabat', { fullPage: true })

  // 6. Pengaturan wilayah
  await page.click('.nav a:has-text("Pengaturan")')
  await page.waitForSelector('h1:has-text("Pengaturan")')
  await field('Nama Kelurahan').fill('Jaya Mukti')
  await field('Kecamatan').fill('Dumai Timur')
  await field('Kota').fill('Dumai')
  await field('Kode Kecamatan').fill('DT')
  await field('Kode Kelurahan').fill('JM')
  await shot('07-pengaturan', { fullPage: true })
  await page.click('button:has-text("Simpan Pengaturan")')
  await sleep(4300)

  // 7. Wizard buat berkas (pakai data contoh agar semua kolom terlihat terisi)
  await page.click('.nav a:has-text("Buat Berkas")')
  await page.waitForSelector('.wiz-steps')
  await page.click('button:has-text("Isi Data Contoh")')
  await sleep(300)
  await shot('08-langkah1-data-berkas')
  await page.click('button:has-text("Lanjut")')
  await shot('09-langkah2-pewaris', { fullPage: true })
  await page.click('button:has-text("Lanjut")')
  await shot('10-langkah3-ahli-waris', { fullPage: true })
  await page.click('button:has-text("Lanjut")')
  await shot('11-langkah4-saksi', { fullPage: true })
  await page.click('button:has-text("Lanjut")')
  await shot('12-langkah5-surat-kuasa', { fullPage: true })
  await page.click('button:has-text("Lanjut")')
  await page.waitForSelector('h2:has-text("Periksa Kembali")')
  await shot('13-langkah6-periksa', { fullPage: true })
  await page.click('button:has-text("Simpan Berkas")')

  // 8. Detail berkas
  await page.waitForSelector('h1:has-text("Berkas Waris")')
  await sleep(4300)
  await shot('14-detail', { fullPage: true })

  // 9. Halaman cetak
  await page.goto(BASE + '/berkas/1/cetak')
  await page.waitForSelector('.judul')
  await shot('15-cetak')

  // 10. Daftar dengan isi + pencarian
  await page.goto(BASE + '/#/')
  await page.waitForSelector('td:has-text("SKAW")')
  await shot('16-daftar')

  await browser.close()
  console.log('SELESAI')
})().catch((e) => { console.error(e); process.exit(1) })
