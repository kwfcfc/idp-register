<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { apiSend, ApiError } from '$lib/api';
  import type { PermissionProfile } from '$lib/types';

  let { data } = $props();

  // Editor state. `editing` is the id of the profile being edited, or '' when
  // creating a new one (id is then a free-text, immutable-on-create field).
  let editing = $state<string | null>(null);
  let id = $state('');
  let label = $state('');
  let description = $state('');
  let selectedGroups = $state<string[]>([]);
  let publicSelectable = $state(false);
  let publicLabel = $state('');
  let sortOrder = $state<number | ''>(0);

  let busy = $state(false);
  let error = $state('');
  let success = $state('');

  const isEdit = $derived(editing !== null);

  function reset() {
    editing = null;
    id = '';
    label = '';
    description = '';
    selectedGroups = [];
    publicSelectable = false;
    publicLabel = '';
    sortOrder = 0;
    error = '';
  }

  function startEdit(p: PermissionProfile) {
    editing = p.id;
    id = p.id;
    label = p.label;
    description = p.description;
    selectedGroups = [...p.groups];
    publicSelectable = p.publicSelectable;
    publicLabel = p.publicLabel;
    sortOrder = p.sortOrder;
    error = '';
    success = '';
  }

  function toggleGroup(name: string, checked: boolean) {
    selectedGroups = checked
      ? [...selectedGroups, name]
      : selectedGroups.filter((g) => g !== name);
  }

  async function save(event: SubmitEvent) {
    event.preventDefault();
    error = '';
    success = '';
    busy = true;
    const body = {
      id: id.trim(),
      label: label.trim(),
      description: description.trim(),
      groups: selectedGroups,
      publicSelectable,
      publicLabel: publicLabel.trim(),
      sortOrder: sortOrder === '' ? 0 : Number(sortOrder)
    };
    try {
      if (isEdit) {
        await apiSend('PUT', `/api/admin/profiles/${encodeURIComponent(editing!)}`, body);
        success = '已更新权限模板。';
      } else {
        await apiSend('POST', '/api/admin/profiles', body);
        success = '已创建权限模板。';
      }
      reset();
      await invalidateAll();
    } catch (e) {
      error = e instanceof ApiError ? e.message : '保存失败，请检查输入。';
    } finally {
      busy = false;
    }
  }

  async function remove(p: PermissionProfile) {
    if (!confirm(`删除权限模板「${p.label}」？被邀请码或申请引用时无法删除。`)) return;
    error = '';
    success = '';
    busy = true;
    try {
      await apiSend('DELETE', `/api/admin/profiles/${encodeURIComponent(p.id)}`);
      if (editing === p.id) reset();
      success = '已删除权限模板。';
      await invalidateAll();
    } catch (e) {
      // 409 = still referenced by a token/application (FK guard preserves audit).
      error = e instanceof ApiError ? e.message : '删除失败。';
    } finally {
      busy = false;
    }
  }
</script>

<svelte:head><title>权限模板 · Invite Console</title></svelte:head>

<div class="page-header">
  <div>
    <h1>权限模板</h1>
    <p>把人类可理解的套餐映射为目标 IdP groups，并决定哪些作为「服务」出现在公开注册表单中。</p>
  </div>
</div>

{#if success}<div class="alert success">{success}</div>{/if}

<div class="grid-2">
  <section class="card" id="editor">
    <div class="card-header"><h2>{isEdit ? '编辑模板' : '新建模板'}</h2></div>
    <div class="card-body">
      {#if error}<div class="alert error">{error}</div>{/if}
      {#if data.groupsError}
        <div class="alert error">
          无法加载目标 IdP 的 group 目录：{data.groupsError}
          模板的 groups 由服务端依据实时目录校验，未加载目录时无法保存。
        </div>
      {/if}

      <form onsubmit={save}>
        <div class="form-grid">
          <div class="field full">
            <label for="id">模板 ID</label>
            <input
              id="id"
              bind:value={id}
              disabled={isEdit}
              placeholder="developer"
              required
            />
            <div class="help">{isEdit ? 'ID 创建后不可更改。' : '唯一、无空格的标识符，创建后不可更改。'}</div>
          </div>
          <div class="field full">
            <label for="label">名称</label>
            <input id="label" bind:value={label} placeholder="开发者" required />
          </div>
          <div class="field full">
            <label for="description">描述（仅管理员可见）</label>
            <input id="description" bind:value={description} placeholder="内部说明此模板的用途" />
          </div>
        </div>

        <div class="field" style="margin-top:8px">
          <span class="field-caption">目标 IdP groups</span>
          <div class="help">仅可选择目标 IdP 实际存在、且未被屏蔽的 group（不变量 #4）。</div>
          {#if data.groups.length}
            <div class="group-picker">
              {#each data.groups as g (g.id)}
                <label class="group-option">
                  <input
                    type="checkbox"
                    checked={selectedGroups.includes(g.name)}
                    onchange={(e) => toggleGroup(g.name, e.currentTarget.checked)}
                  />
                  <span>{g.name}</span>
                </label>
              {/each}
            </div>
          {:else}
            <div class="empty">无可用 group。</div>
          {/if}
        </div>

        <hr style="border:0;border-top:1px solid var(--line);margin:20px 0" />

        <div class="field">
          <label class="group-option" style="font-weight:600">
            <input type="checkbox" bind:checked={publicSelectable} />
            <span>在公开注册表单中作为「服务」展示</span>
          </label>
          <div class="help">勾选后，申请人可在公开表单中选择此服务；表单只看到下方的公开名称，不会看到 group。</div>
        </div>

        {#if publicSelectable}
          <div class="form-grid" style="margin-top:8px">
            <div class="field">
              <label for="publicLabel">公开名称（留空则用「名称」）</label>
              <input id="publicLabel" bind:value={publicLabel} placeholder="例如：Matrix 账号" />
            </div>
            <div class="field">
              <label for="sortOrder">排序（升序）</label>
              <input id="sortOrder" type="number" bind:value={sortOrder} />
            </div>
          </div>
        {/if}

        <div class="form-actions">
          <button class="button primary" type="submit" disabled={busy || !!data.groupsError}>
            {isEdit ? '保存修改' : '创建模板'}
          </button>
          {#if isEdit}
            <button class="button secondary" type="button" onclick={reset} disabled={busy}>取消</button>
          {/if}
        </div>
      </form>
    </div>
  </section>

  <section class="card">
    <div class="card-header"><h2>说明</h2></div>
    <div class="card-body">
      <div class="info-list">
        <div class="info-key">用途</div><div class="info-value">审批时把模板展开为目标 IdP groups，避免手工拼写</div>
        <div class="info-key">权限来源</div><div class="info-value">groups 取自目标 IdP 实时目录减去屏蔽名单（ADR-0012）</div>
        <div class="info-key">公开服务</div><div class="info-value">勾选「公开展示」后出现在注册表单的服务选择中</div>
        <div class="info-key">删除保护</div><div class="info-value">仍被邀请码或申请引用时拒绝删除（409），保留审计</div>
        <div class="info-key">审计</div><div class="info-value">创建、修改与删除均写入审计日志</div>
      </div>
    </div>
  </section>
</div>

<section class="card" style="margin-top:18px">
  <div class="card-header"><h2>现有模板</h2></div>
  <div class="card-body" style="display:grid;gap:14px">
    {#if data.profiles.length}
      {#each data.profiles as profile (profile.id)}
        <div style="padding:14px;border:1px solid var(--line);border-radius:12px">
          <div style="display:flex;justify-content:space-between;align-items:flex-start;gap:12px">
            <div>
              <div class="primary-text">
                {profile.label}
                <code style="font-size:10px;color:var(--muted)">{profile.id}</code>
                {#if profile.publicSelectable}
                  <span class="group-chip" style="background:var(--blue-soft);color:#2b5482">
                    公开：{profile.publicLabel || profile.label}
                  </span>
                {/if}
              </div>
              <div class="secondary-text" style="margin:6px 0 10px">{profile.description || '（无描述）'}</div>
            </div>
            <div style="display:flex;gap:6px;flex-shrink:0">
              <button class="button secondary small" type="button" onclick={() => startEdit(profile)}>编辑</button>
              <button class="button danger small" type="button" onclick={() => remove(profile)}>删除</button>
            </div>
          </div>
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

<style>
  .group-picker {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 8px;
    margin-top: 8px;
    max-height: 240px;
    overflow: auto;
    padding: 12px;
    border: 1px solid var(--line);
    border-radius: 10px;
  }
  .group-option {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
  }
  .group-option input {
    width: auto;
    margin: 0;
  }
  /* Caption for the checkbox group — styled like a field <label> but not a
     label element, since it isn't bound to a single control (a11y). */
  .field-caption {
    font-size: 12px;
    font-weight: 750;
  }
</style>
