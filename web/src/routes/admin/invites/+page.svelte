<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import Badge from '$lib/components/Badge.svelte';
  import { invalidateAll } from '$app/navigation';
  import { apiSend, ApiError } from '$lib/api';
  import { fmtDate } from '$lib/format';
  import { tokenStatus, type RegistrationToken } from '$lib/types';

  let { data } = $props();
  const now = Date.now();

  // Create form.
  let email = $state('');
  let profileId = $state('');
  let token = $state('');
  let expiresInDays = $state<number | ''>(7);
  let maxUses = $state<number | ''>(1);
  let note = $state('');

  let busy = $state(false);
  let createError = $state('');
  let createdCode = $state('');
  let copied = $state(false);

  const profileLabel = (id: string | null) =>
    id ? (data.profiles.find((p) => p.id === id)?.label ?? id) : '任意';
  const usesLabel = (t: RegistrationToken) =>
    `${t.pending + t.completed} / ${t.usesAllowed ?? '∞'}`;

  async function copyCode(code: string) {
    try {
      await navigator.clipboard.writeText(code);
      copied = true;
      setTimeout(() => (copied = false), 1800);
    } catch {
      /* clipboard unavailable; the code is shown inline regardless */
    }
  }

  async function create(event: SubmitEvent) {
    event.preventDefault();
    createError = '';
    createdCode = '';
    busy = true;
    try {
      const created = await apiSend<RegistrationToken>('POST', '/api/admin/tokens', {
        token: token.trim() || undefined,
        usesAllowed: maxUses === '' ? null : Number(maxUses),
        expiryTime: expiresInDays === '' ? null : now + Number(expiresInDays) * 86_400_000,
        emailConstraint: email.trim() || null,
        profileId: profileId || null,
        note: note.trim()
      });
      createdCode = created.token;
      token = '';
      note = '';
      await invalidateAll();
    } catch (e) {
      createError = e instanceof ApiError ? e.message : '创建失败，请检查输入。';
    } finally {
      busy = false;
    }
  }

  async function setActive(t: RegistrationToken, active: boolean) {
    await apiSend('POST', `/api/admin/tokens/${t.id}/active`, { active });
    await invalidateAll();
  }

  async function remove(t: RegistrationToken) {
    if (!confirm('删除该邀请码？此操作不可撤销。')) return;
    await apiSend('DELETE', `/api/admin/tokens/${t.id}`);
    await invalidateAll();
  }
</script>

<svelte:head><title>邀请码 · Invite Console</title></svelte:head>

<div class="page-header">
  <div>
    <h1>邀请码</h1>
    <p>明文、限次、可设有效期的邀请码（与 Synapse 注册令牌语义一致）。有效邀请码可自动批准申请。</p>
  </div>
</div>

<div class="grid-2">
  <section class="card" id="new">
    <div class="card-header"><h2>创建邀请码</h2></div>
    <div class="card-body">
      {#if createError}<div class="alert error">{createError}</div>{/if}
      {#if createdCode}
        <div class="alert success">邀请码已创建。可作为 <code>?token=邀请码</code> 链接分发给申请人。</div>
        <div class="code-box">
          <code>{createdCode}</code>
          <button class="button secondary small" type="button" onclick={() => copyCode(createdCode)}>
            {copied ? '已复制' : '复制'}
          </button>
        </div>
      {/if}
      <form onsubmit={create}>
        <div class="form-grid">
          <div class="field full">
            <label for="email">绑定邮箱（可选）</label>
            <input id="email" type="email" bind:value={email} placeholder="user@example.com" />
            <div class="help">填写后，申请邮箱必须精确匹配才会自动批准。</div>
          </div>
          <div class="field full">
            <label for="profileId">权限模板（可选）</label>
            <select id="profileId" bind:value={profileId}>
              <option value="">不指定（审批时再选）</option>
              {#each data.profiles as profile (profile.id)}
                <option value={profile.id}>{profile.label} — {profile.description}</option>
              {/each}
            </select>
          </div>
          <div class="field">
            <label for="expiresInDays">有效天数（留空=永不过期）</label>
            <input id="expiresInDays" type="number" min="1" max="3650" bind:value={expiresInDays} />
          </div>
          <div class="field">
            <label for="maxUses">最大使用次数（留空=不限）</label>
            <input id="maxUses" type="number" min="1" max="100000" bind:value={maxUses} />
          </div>
          <div class="field full">
            <label for="token">自定义代码（可选）</label>
            <input id="token" bind:value={token} placeholder="留空则随机生成" />
            <div class="help">允许字符：A–Z a–z 0–9 . _ ~ -，最长 64 位。</div>
          </div>
          <div class="field full">
            <label for="note">备注（可选）</label>
            <input id="note" bind:value={note} placeholder="用途说明，仅管理员可见" />
          </div>
        </div>
        <div class="form-actions">
          <button class="button primary" type="submit" disabled={busy}>生成邀请码</button>
        </div>
      </form>
    </div>
  </section>
  <section class="card">
    <div class="card-header"><h2>说明</h2></div>
    <div class="card-body">
      <div class="info-list">
        <div class="info-key">存储方式</div><div class="info-value">明文存储，按代码字符串寻址（ADR-0004）</div>
        <div class="info-key">权限来源</div><div class="info-value">服务端权限模板，公开表单不能自行提交 groups</div>
        <div class="info-key">使用策略</div><div class="info-value">可绑定邮箱、设置有效期与最大使用次数</div>
        <div class="info-key">计数</div><div class="info-value">使用 = 预留(pending) + 已完成(completed)</div>
        <div class="info-key">审计</div><div class="info-value">创建、启停与删除均写入审计日志</div>
      </div>
    </div>
  </section>
</div>

<section class="card" style="margin-top:18px">
  <div class="card-header"><h2>邀请码记录</h2></div>
  {#if data.tokens.length}
    <div class="table-wrap">
      <table>
        <thead>
          <tr><th>代码</th><th>绑定邮箱</th><th>权限模板</th><th>使用</th><th>有效期</th><th>状态</th><th></th></tr>
        </thead>
        <tbody>
          {#each data.tokens as t (t.id)}
            <tr>
              <td><code>{t.token}</code><div class="secondary-text">{fmtDate(t.createdAt)}</div></td>
              <td>{t.emailConstraint || '任意邮箱'}</td>
              <td>{profileLabel(t.profileId)}</td>
              <td>{usesLabel(t)}</td>
              <td>{t.expiryTime ? fmtDate(t.expiryTime) : '永不过期'}</td>
              <td><Badge status={tokenStatus(t, now)} /></td>
              <td>
                <div style="display:flex;gap:6px;justify-content:flex-end">
                  {#if t.active}
                    <button class="button secondary small" type="button" onclick={() => setActive(t, false)}>停用</button>
                  {:else}
                    <button class="button secondary small" type="button" onclick={() => setActive(t, true)}>启用</button>
                  {/if}
                  <button class="button danger small" type="button" onclick={() => remove(t)}>删除</button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {:else}
    <div class="empty">尚未创建邀请码。</div>
  {/if}
</section>
