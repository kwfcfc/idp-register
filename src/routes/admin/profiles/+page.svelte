<script lang="ts">let { data, form } = $props();</script>
<svelte:head><title>权限模板 · Invite Console</title></svelte:head>
<div class="page-header"><div><h1>权限模板</h1><p>把人类可理解的套餐映射为 Rauthy groups，避免审批时手工拼写权限。</p></div></div>
<div class="grid-2">
  <section class="card">
    <div class="card-header"><h2>现有模板</h2></div>
    <div class="card-body" style="display:grid;gap:14px">
      {#each data.profiles as profile}
        <div style="padding:14px;border:1px solid var(--line);border-radius:12px">
          <div class="primary-text">{profile.label} <code style="font-size:10px;color:var(--muted)">{profile.id}</code></div>
          <div class="secondary-text" style="margin:6px 0 10px">{profile.description}</div>
          <div class="group-list">{#each profile.groups as group}<span class="group-chip">{group}</span>{/each}</div>
        </div>
      {/each}
    </div>
  </section>
  <section class="card">
    <div class="card-header"><h2>新增或更新模板</h2></div>
    <div class="card-body">
      {#if form?.error}<div class="alert error">{form.error}</div>{/if}
      <form method="POST" action="?/upsert">
        <div class="form-grid">
          <div class="field"><label for="id">模板 ID</label><input id="id" name="id" placeholder="developer" required /></div>
          <div class="field"><label for="label">显示名称</label><input id="label" name="label" placeholder="开发者" required /></div>
          <div class="field full"><label for="description">说明</label><input id="description" name="description" placeholder="可访问 Matrix、GoToSocial 和 Forgejo" /></div>
          <div class="field full"><label for="groups">Rauthy groups</label><textarea id="groups" name="groups" placeholder={'svc:matrix:user\nsvc:gotosocial:user\nsvc:forgejo:user'} required></textarea><div class="help">每行一个，或使用逗号分隔。</div></div>
        </div>
        <div class="form-actions"><button class="button primary" type="submit">保存模板</button></div>
      </form>
    </div>
  </section>
</div>
