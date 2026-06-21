<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { page } from '$app/state';
  const failed = page.url.searchParams.get('error') === 'auth';
</script>

<svelte:head><title>管理员登录 · idp-register</title></svelte:head>

<main class="center-shell">
  <section class="center-card">
    <div class="brand-mark" style="display:grid;place-items:center">◈</div>
    <p class="eyebrow">INVITE CONSOLE</p>
    <h1>注册与邀请管理</h1>
    <p class="lede">使用你的身份提供方登录。只有属于管理员组（或在允许名单中）的账户可以进入。</p>

    {#if failed}<div class="alert error">登录失败或未被授权，请重试。</div>{/if}

    <!-- data-sveltekit-reload: this is a backend route, force a full navigation
         so the OIDC redirect flow runs instead of client-side routing. -->
    <a class="button primary" href="/auth/login" data-sveltekit-reload>
      登录
      <span aria-hidden="true">→</span>
    </a>

    <div class="security-note">
      <span>●</span>
      密码、Passkey 和多因素认证均由身份提供方处理。
    </div>
  </section>
</main>
