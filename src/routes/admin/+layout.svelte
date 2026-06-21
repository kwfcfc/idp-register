<script lang="ts">
  import '../../app.css';
  import { page } from '$app/state';
  let { data, children } = $props();

  const nav = [
    { href: '/admin', label: '概览', icon: '⌂' },
    { href: '/admin/applications', label: '注册申请', icon: '▤' },
    { href: '/admin/invites', label: '邀请码', icon: '◇' },
    { href: '/admin/profiles', label: '权限模板', icon: '⌘' }
  ];
  const isActive = (href: string) => href === '/admin' ? page.url.pathname === href : page.url.pathname.startsWith(href);
  const initial = (data.user.displayName || data.user.email).slice(0, 1).toUpperCase();
</script>

<div class="app-shell">
  <aside class="sidebar">
    <div class="brand">
      <div class="brand-mark">◈</div>
      <div>
        <div class="brand-title">Invite Console</div>
        <div class="brand-subtitle">Rauthy Access Gateway</div>
      </div>
    </div>
    <div class="nav-label">管理</div>
    <nav class="nav-list" aria-label="主导航">
      {#each nav as item}
        <a class:active={isActive(item.href)} class="nav-link" href={item.href}>
          <span class="nav-icon">{item.icon}</span>{item.label}
        </a>
      {/each}
    </nav>
    <div class="sidebar-footer">
      <div class="user-line">
        <div class="avatar">{initial}</div>
        <div class="user-meta">
          <div class="user-name">{data.user.displayName}</div>
          <div class="user-email">{data.user.email}</div>
        </div>
      </div>
      <form class="logout-form" method="POST" action="/logout">
        <button class="logout-button" type="submit">退出登录</button>
      </form>
    </div>
  </aside>
  <main class="main">
    <header class="topbar">
      <div class="breadcrumb">注册与邀请管理</div>
      <div class="environment"><span class="environment-dot"></span>Rauthy 已连接</div>
    </header>
    <div class="content">{@render children()}</div>
  </main>
</div>
