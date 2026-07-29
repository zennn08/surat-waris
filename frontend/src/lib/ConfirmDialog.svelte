<script>
  export let title = 'Konfirmasi'
  export let confirmLabel = 'Ya, Lanjutkan'
  export let danger = false

  let dlg
  let message = ''
  let resolve = null

  export function ask(msg) {
    message = msg
    dlg.showModal()
    return new Promise((r) => (resolve = r))
  }

  function answer(v) {
    if (resolve) { resolve(v); resolve = null }
    if (dlg.open) dlg.close()
  }
</script>

<!-- on:close menangani tombol Esc: dianggap Batal -->
<dialog class="confirm-dialog" bind:this={dlg} on:close={() => answer(false)}>
  <h3>{title}</h3>
  <p class="muted">{message}</p>
  <div class="dlg-actions">
    <button type="button" class="btn" on:click={() => answer(false)}>Batal</button>
    <button type="button" class="btn {danger ? 'btn-danger' : 'btn-primary'}" on:click={() => answer(true)}>
      {confirmLabel}
    </button>
  </div>
</dialog>
