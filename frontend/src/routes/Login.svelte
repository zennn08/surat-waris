<script>
  import { login } from '../lib/stores.js'

  let username = ''
  let password = ''
  let error = ''
  let busy = false

  async function submit() {
    error = ''
    busy = true
    try {
      await login(username.trim(), password)
    } catch (e) {
      error = e.message
    } finally {
      busy = false
    }
  }
</script>

<div class="login-wrap">
  <div class="card login-split">
    <div class="photo">
      <img src="/kantor-camat.jpg" alt="Tugu Kantor Camat Dumai Timur" />
      <div class="photo-caption">
        <strong>Kantor Camat Dumai Timur</strong>
        <span>Kota Dumai, Riau</span>
      </div>
    </div>

    <div class="form-side">
      <div style="text-align:center; margin-bottom:1.25rem;">
        <div class="brand" style="font-size:1.4rem;">SIWARIS</div>
        <div class="muted small">Sistem Informasi Surat Ahli Waris<br />Kecamatan Dumai Timur</div>
      </div>

      {#if error}<div class="alert alert-error">{error}</div>{/if}

      <form on:submit|preventDefault={submit}>
        <div class="field">
          <label for="u">Username</label>
          <input id="u" bind:value={username} autocomplete="username" required />
        </div>
        <div class="field">
          <label for="p">Password</label>
          <input id="p" type="password" bind:value={password} autocomplete="current-password" required />
        </div>
        <button class="btn btn-primary" style="width:100%;" disabled={busy}>
          {busy ? 'Masuk…' : 'Masuk'}
        </button>
      </form>
    </div>
  </div>
</div>

<style>
  .login-split {
    display: grid;
    grid-template-columns: 1fr 1fr;
    width: 100%;
    max-width: 860px;
    padding: 0;
    overflow: hidden;
  }

  .photo {
    position: relative;
    min-height: 420px;
  }
  .photo img {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
  /* scrim hijau dinas agar caption terbaca & foto menyatu dengan palet */
  .photo::after {
    content: '';
    position: absolute;
    inset: 0;
    background: linear-gradient(to top, rgba(17, 70, 59, 0.85), rgba(17, 70, 59, 0.05) 55%);
  }
  .photo-caption {
    position: absolute;
    left: 1.4rem;
    right: 1.4rem;
    bottom: 1.25rem;
    z-index: 1;
    color: #fff;
  }
  .photo-caption strong { display: block; font-size: 1.05rem; }
  .photo-caption span { font-size: 0.85rem; opacity: 0.85; }

  .form-side {
    padding: 2.5rem 2.25rem;
    align-self: center;
  }

  @media (max-width: 720px) {
    .login-split { grid-template-columns: 1fr; max-width: 400px; }
    .photo { min-height: 160px; }
    .form-side { padding: 1.75rem 1.5rem; }
  }
</style>
