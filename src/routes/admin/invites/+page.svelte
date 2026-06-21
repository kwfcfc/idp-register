<script lang="ts">
  import Badge from '$lib/components/Badge.svelte';
  let { data, form } = $props();
  const date = (value: string | Date) => new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value));
  let copied = $state(false);
  async function copyCode(code: string) {
    await navigator.clipboard.writeText(code);
    copied = true;
    setTimeout(() => copied = false, 1800);
  }
</script>

<svelte:head><title>邀请码 · Invite Console</title></svelte:head>
<div class="page-header">
  <div><h1>邀请码</h1><p>邀请码只以摘要形式保存；明文只在创建成功时显示一次。</p></div>
</div>

<div class="grid-2">
  <section class="card" id="new">
    <div class="card-header"><h2>创建邀请码</h2></div>
    <div class="card-body">
      {#if form?.createError}<div class="alert error">{form.createError}</div>{/if}
      {#if form?.createdCode}
        <div class="alert success">邀请码已创建。请立即复制，刷新页面后无法再次查看。</div>
        <div class="code-box"><code>{form.createdCode}</code><button class="button secondary small" type="button" onclick={() => copyCode(form.createdCode)}>{copied ? '已复制' : '复制'}</button></div>
      {/if}
      <form method="POST" action="?/create">
        <div class="form-grid">
          <div class="field full"><label for="email">绑定邮箱（可选）</label><input id="email" name="email" type="email" placeholder="user@example.com" /><div class="help">填写后，申请邮箱必须精确匹配。</div></div>
          <div class="field full"><label for="profileId">权限模板</label><select id="profileId" name="profileId" required>{#each data.profiles as profile}<option value={profile.id}>{profile.label} — {profile.description}</option>{/each}</select></div>
          <div class="field"><label for="expiresInDays">有效天数</label><input id="expiresInDays" name="expiresInDays" type="number" min="1" max="365" value="7" required /></div>
          <div class="field"><label for="maxUses">最大使用次数</label><input id="maxUses" name="maxUses" type="number" min="1" max="100" value="1" required /></div>
        </div>
        <div class="form-actions"><button class="button primary" type="submit">生成邀请码</button></div>
      </form>
    </div>
  </section>
  <section class="card">
    <div class="card-header"><h2>设计说明</h2></div>
    <div class="card-body">
      <div class="info-list">
        <div class="info-key">存储方式</div><div class="info-value">HMAC-SHA-256 摘要，不存明文</div>
        <div class="info-key">权限来源</div><div class="info-value">服务端权限模板，客户端不能自行提交 groups</div>
        <div class="info-key">使用策略</div><div class="info-value">可绑定邮箱、有效期和最大使用次数</div>
        <div class="info-key">审计</div><div class="info-value">创建和撤销操作写入 audit_log</div>
      </div>
    </div>
  </section>
</div>

<section class="card" style="margin-top:18px">
  <div class="card-header"><h2>邀请码记录</h2><span class="secondary-text">最近 200 条</span></div>
  {#if data.invites.length}
    <div class="table-wrap"><table>
      <thead><tr><th>代码</th><th>绑定邮箱</th><th>权限模板</th><th>使用</th><th>有效期</th><th>状态</th><th></th></tr></thead>
      <tbody>{#each data.invites as invite}<tr>
        <td><code>{invite.code_prefix}••••••</code><div class="secondary-text">{date(invite.created_at)}</div></td>
        <td>{invite.email_constraint || '任何邮箱'}</td>
        <td>{invite.profile_label}</td>
        <td>{invite.used_count} / {invite.max_uses}</td>
        <td>{date(invite.expires_at)}</td>
        <td><Badge status={invite.computed_status} /></td>
        <td>{#if invite.computed_status === 'active'}<form method="POST" action="?/revoke"><input type="hidden" name="id" value={invite.id} /><button class="button danger small" type="submit">撤销</button></form>{/if}</td>
      </tr>{/each}</tbody>
    </table></div>
  {:else}<div class="empty">尚未创建邀请码。</div>{/if}
</section>
