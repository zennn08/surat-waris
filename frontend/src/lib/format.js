// Format tanggal ISO (YYYY-MM-DD) → "12 Juli 2026". Nilai lain dikembalikan apa adanya.
const fmt = new Intl.DateTimeFormat('id-ID', { day: 'numeric', month: 'long', year: 'numeric' })

export function fmtDate(s) {
  if (!s || !/^\d{4}-\d{2}-\d{2}/.test(s)) return s || '—'
  const d = new Date(s)
  return isNaN(d) ? s : fmt.format(d)
}
