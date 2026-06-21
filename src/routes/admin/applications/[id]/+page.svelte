<script lang="ts">
  import Badge from '$lib/components/Badge.svelte';
  import { page } from '$app/state';
  let { data, form } = $props();
  const a = data.application;
  const date = (value: string | Date | null) => value ? new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : '—';
</script>

<svelte:head><title>{a.username} · 注册申请</title></svelte:head>
<div class="page-header">
  <div><div class="secondary-text"><a class="link" href="/admin/applications">← 返回申请列表</a></div><h1 style="margin-top:10px">{a.username}</h1><p>{a.email}</p></div>
  <Badge status={a.status} />
</div>

{#if page.url.searchParams.get('approved')}<div class="alert success">Rauthy 用户已创建并完成权限分配。</div>{/if}
{#if form?.error}<div class="alert error">{form.error}</div>{/if}
{#if a.provisioning_error}<div class="alert error"><strong>上次创建失败：</strong>{a.provisioning_error}</div>{/if}

<div class="detail-grid">
  <div style="display:grid;gap:18px">
    <section class="card">
      <div class="card-header"><h2>申请资料</h2></div>
      <div class="card-body"><div class="info-list">
        <div class="info-key">邮箱</div><div class="info-value">{a.email}</div>
        <div class="info-key">用户名</div><div class="info-value"><code>{a.username}</code></div>
        <div class="info-key">请求服务</div><div class="info-value"><div class="group-list">{#each a.requested_services as service}<span class="group-chip">{service}</span>{/each}</div></div>
        <div class="info-key">邀请码</div><div class="info-value">{a.invite_prefix ? `${a.invite_prefix}•••••• · ${a.invite_profile_label}` : '未使用'}</div>
        <div class="info-key">提交时间</div><div class="info-value">{date(a.created_at)}</div>
        <div class="info-key">邮箱验证</div><div class="info-value">{date(a.email_verified_at)}</div>
        <div class="info-key">人机验证</div><div class="info-value">{a.captcha_provider || '—'} · {date(a.captcha_verified_at)}</div>
        <div class="info-key">来源 IP</div><div class="info-value"><code>{a.submitted_ip || '—'}</code></div>
        {#if a.rauthy_user_id}<div class="info-key">Rauthy User ID</div><div class="info-value"><code>{a.rauthy_user_id}</code></div>{/if}
      </div></div>
    </section>
    <section class="card">
      <div class="card-header"><h2>供管理员审核的说明</h2></div>
      <div class="card-body"><div class="review-text">{a.review_text}</div></div>
    </section>
  </div>

  <aside class="card">
    <div class="card-header"><h2>审核决定</h2></div>
    <div class="card-body">
      {#if ['pending','provisioning_failed'].includes(a.status)}
        <form method="POST" action="?/approve">
          <div class="field"><label for="profileId">批准后的权限模板</label><select id="profileId" name="profileId" required>{#each data.profiles as profile}<option value={profile.id}>{profile.label}</option>{/each}</select><div class="help">审批时由服务端展开为 Rauthy groups。</div></div>
          <div class="field" style="margin-top:14px"><label for="approve-note">审核备注</label><textarea id="approve-note" name="note" placeholder="说明批准依据或需要留存的信息" required></textarea></div>
          <button class="button primary" style="width:100%;margin-top:14px" type="submit">批准并创建 Rauthy 用户</button>
        </form>
        <hr style="border:0;border-top:1px solid var(--line);margin:22px 0" />
        <form method="POST" action="?/changes">
          <div class="field"><label for="changes-note">要求补充</label><textarea id="changes-note" name="note" placeholder="需要申请人补充的资料" required></textarea></div>
          <button class="button secondary" style="width:100%;margin-top:12px" type="submit">标记为需补充</button>
        </form>
        <form method="POST" action="?/reject" style="margin-top:10px">
          <div class="field"><label for="reject-note">拒绝原因</label><textarea id="reject-note" name="note" placeholder="内部记录或发送给申请人的理由" required></textarea></div>
          <button class="button danger" style="width:100%;margin-top:12px" type="submit">拒绝申请</button>
        </form>
      {:else}
        <div class="info-list" style="grid-template-columns:100px 1fr">
          <div class="info-key">处理状态</div><div><Badge status={a.status} /></div>
          <div class="info-key">处理时间</div><div>{date(a.updated_at)}</div>
          <div class="info-key">审核备注</div><div>{a.decision_note || '—'}</div>
        </div>
      {/if}
    </div>
  </aside>
</div>
