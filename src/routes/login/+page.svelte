<script lang="ts">
  let { data } = $props();
</script>

<svelte:head><title>管理员登录 · Invite Console</title></svelte:head>

<main class="login-shell">
  <section class="login-card">
    <div class="brand-mark">◈</div>
    <p class="eyebrow">INVITE CONSOLE</p>
    <h1>注册与邀请管理</h1>
    <p class="muted">使用 Rauthy 统一身份登录。只有属于管理员组的账户可以进入。</p>
    <a class="primary-button full" href={`/auth/login?next=${encodeURIComponent(data.next)}`}>
      使用 Rauthy 登录
      <span aria-hidden="true">→</span>
    </a>
    <div class="security-note">
      <span>●</span>
      密码、Passkey 和多因素认证均由 Rauthy 处理
    </div>
  </section>
</main>

<style>
  :global(body) { margin: 0; font-family: Inter, ui-sans-serif, system-ui, sans-serif; background: #f3f0e9; color: #17201c; }
  .login-shell { min-height: 100vh; display: grid; place-items: center; padding: 24px; background: radial-gradient(circle at 20% 10%, rgba(52, 112, 84, .14), transparent 30%), #f3f0e9; }
  .login-card { width: min(440px, 100%); padding: 42px; border-radius: 24px; background: rgba(255,255,255,.84); border: 1px solid rgba(23,32,28,.11); box-shadow: 0 24px 70px rgba(26,46,36,.12); backdrop-filter: blur(16px); }
  .brand-mark { width: 48px; height: 48px; display: grid; place-items: center; border-radius: 14px; background: #1f5b42; color: white; font-size: 26px; }
  .eyebrow { margin: 26px 0 8px; font-size: 12px; font-weight: 800; letter-spacing: .16em; color: #527263; }
  h1 { margin: 0; font-size: 32px; letter-spacing: -.04em; }
  .muted { color: #65726b; line-height: 1.7; margin: 16px 0 28px; }
  .primary-button { display: flex; justify-content: space-between; align-items: center; background: #1f5b42; color: white; text-decoration: none; padding: 14px 16px; border-radius: 12px; font-weight: 750; }
  .primary-button:hover { background: #174a35; }
  .security-note { display: flex; gap: 9px; margin-top: 20px; font-size: 12px; line-height: 1.5; color: #778079; }
  .security-note span { color: #51a16e; }
</style>
