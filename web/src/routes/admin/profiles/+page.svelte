<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  let { data } = $props();
</script>

<svelte:head><title>权限模板 · Invite Console</title></svelte:head>

<div class="page-header">
  <div>
    <h1>权限模板</h1>
    <p>把人类可理解的套餐映射为目标 IdP groups，避免审批时手工拼写权限。</p>
  </div>
</div>

<section class="card">
  <div class="card-header"><h2>现有模板</h2></div>
  <div class="card-body" style="display:grid;gap:14px">
    {#if data.profiles.length}
      {#each data.profiles as profile (profile.id)}
        <div style="padding:14px;border:1px solid var(--line);border-radius:12px">
          <div class="primary-text">
            {profile.label}
            <code style="font-size:10px;color:var(--muted)">{profile.id}</code>
          </div>
          <div class="secondary-text" style="margin:6px 0 10px">{profile.description}</div>
          <div class="group-list">
            {#each profile.groups as group}<span class="group-chip">{group}</span>{:else}<span class="secondary-text">（无 groups）</span>{/each}
          </div>
        </div>
      {/each}
    {:else}
      <div class="empty">尚未配置权限模板。</div>
    {/if}
  </div>
</section>

<div class="alert" style="margin-top:18px;background:var(--blue-soft);color:#2b5482;border:1px solid #cfe0f3">
  权限模板目前为只读：请在服务端配置（数据库或后续提供的管理接口）。公开表单只能引用模板，无法直接设置 groups（不变量 #4）。
</div>
