<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import Badge from '$lib/components/Badge.svelte';
  import { invalidateAll } from '$app/navigation';
  import { apiSend, ApiError } from '$lib/api';
  import { fmtDate } from '$lib/format';

  let { data } = $props();
  // `a` tracks data.application reactively so it updates after invalidateAll().
  const a = $derived(data.application);
  const decidable = $derived(a.status === 'pending' || a.status === 'provisioning_failed');

  let profileId = $state('');
  let approveNote = $state('');
  let rejectNote = $state('');

  let busy = $state(false);
  let error = $state('');
  let success = $state('');

  // Default the profile select to the first profile / the application's prior choice.
  $effect(() => {
    if (!profileId) profileId = a.approvedProfileId ?? data.profiles[0]?.id ?? '';
  });

  async function run(fn: () => Promise<void>, ok: string) {
    error = '';
    success = '';
    busy = true;
    try {
      await fn();
      await invalidateAll();
      success = ok;
    } catch (e) {
      error = e instanceof ApiError ? e.message : '操作失败，请重试。';
    } finally {
      busy = false;
    }
  }

  const approve = () =>
    run(async () => {
      if (!profileId) throw new ApiError(400, '请选择权限模板。');
      await apiSend('POST', `/api/admin/applications/${a.id}/approve`, {
        profileId,
        note: approveNote.trim()
      });
    }, '已批准并触发目标 IdP 用户创建。');

  const reject = () =>
    run(async () => {
      await apiSend('POST', `/api/admin/applications/${a.id}/decide`, {
        status: 'rejected',
        note: rejectNote.trim()
      });
    }, '已拒绝该申请。');
</script>

<svelte:head><title>{a.username} · 注册申请</title></svelte:head>

<div class="page-header">
  <div>
    <div class="secondary-text"><a class="link" href="/admin/applications">← 返回申请列表</a></div>
    <h1 style="margin-top:10px">{a.username || a.email}</h1>
    <p>{a.email}</p>
  </div>
  <Badge status={a.status} />
</div>

{#if success}<div class="alert success">{success}</div>{/if}
{#if error}<div class="alert error">{error}</div>{/if}
{#if a.provisioningError}<div class="alert error"><strong>上次创建失败：</strong>{a.provisioningError}</div>{/if}

<div class="detail-grid">
  <div style="display:grid;gap:18px">
    <section class="card">
      <div class="card-header"><h2>申请资料</h2></div>
      <div class="card-body">
        <div class="info-list">
          <div class="info-key">邮箱</div><div class="info-value">{a.email}</div>
          <div class="info-key">用户名</div><div class="info-value"><code>{a.username}</code></div>
          <div class="info-key">请求服务</div>
          <div class="info-value">
            <div class="group-list">
              {#each a.requestedServices ?? [] as service}<span class="group-chip">{service}</span>{:else}—{/each}
            </div>
          </div>
          <div class="info-key">邀请码</div>
          <div class="info-value">{a.tokenId ? '已使用邀请码' : '未使用'}</div>
          <div class="info-key">提交时间</div><div class="info-value">{fmtDate(a.createdAt)}</div>
          <div class="info-key">人机验证</div><div class="info-value">{a.captchaProvider || '—'}</div>
          <div class="info-key">来源 IP</div><div class="info-value"><code>{a.submittedIp || '—'}</code></div>
          {#if a.approvedProfileId}
            <div class="info-key">批准模板</div><div class="info-value"><code>{a.approvedProfileId}</code></div>
          {/if}
          {#if a.providerUserId}
            <div class="info-key">目标 IdP User ID</div><div class="info-value"><code>{a.providerUserId}</code></div>
          {/if}
        </div>
      </div>
    </section>
    <section class="card">
      <div class="card-header"><h2>供管理员审核的说明</h2></div>
      <div class="card-body"><div class="review-text">{a.reviewText || '（无）'}</div></div>
    </section>
  </div>

  <aside class="card">
    <div class="card-header"><h2>审核决定</h2></div>
    <div class="card-body">
      {#if decidable}
        <div class="field">
          <label for="profileId">批准后的权限模板</label>
          <select id="profileId" bind:value={profileId} required>
            {#each data.profiles as profile (profile.id)}<option value={profile.id}>{profile.label}</option>{/each}
          </select>
          <div class="help">审批时由服务端展开为目标 IdP groups。</div>
        </div>
        <div class="field" style="margin-top:14px">
          <label for="approve-note">审核备注</label>
          <textarea id="approve-note" bind:value={approveNote} placeholder="说明批准依据或需要留存的信息"></textarea>
        </div>
        <button class="button primary" style="width:100%;margin-top:14px" disabled={busy} onclick={approve}>
          批准并创建用户
        </button>

        <hr style="border:0;border-top:1px solid var(--line);margin:22px 0" />

        <div class="field">
          <label for="reject-note">拒绝原因</label>
          <textarea id="reject-note" bind:value={rejectNote} placeholder="内部记录"></textarea>
        </div>
        <button
          class="button danger"
          style="width:100%;margin-top:12px"
          disabled={busy}
          onclick={reject}
        >
          拒绝申请
        </button>
      {:else}
        <div class="info-list" style="grid-template-columns:100px 1fr">
          <div class="info-key">处理状态</div><div><Badge status={a.status} /></div>
          <div class="info-key">处理时间</div><div>{fmtDate(a.reviewedAt ?? a.updatedAt)}</div>
          <div class="info-key">处理人</div><div>{a.reviewedByEmail || '—'}</div>
          <div class="info-key">审核备注</div><div>{a.decisionNote || '—'}</div>
        </div>
      {/if}
    </div>
  </aside>
</div>
