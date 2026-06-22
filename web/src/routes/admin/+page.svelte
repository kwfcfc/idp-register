<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import Badge from '$lib/components/Badge.svelte';
  import StatCard from '$lib/components/StatCard.svelte';
  import { fmtDate } from '$lib/format';
  let { data } = $props();
</script>

<svelte:head><title>概览 · Invite Console</title></svelte:head>

<div class="page-header">
  <div>
    <h1>管理概览</h1>
    <p>集中处理注册申请、邀请码和下游应用权限。</p>
  </div>
  <div class="header-actions">
    <a class="button primary" href="/admin/invites#new">＋ 新建邀请码</a>
  </div>
</div>

<div class="grid-4">
  <StatCard label="待审核申请" value={data.stats.pending} note="需要管理员作出决定" icon="▤" />
  <StatCard label="已批准" value={data.stats.approved} note="已创建下游账户" icon="✓" />
  <StatCard label="有效邀请码" value={data.stats.activeInvites} note="未过期且仍有可用次数" icon="◇" />
  <StatCard label="创建失败" value={data.stats.failed} note="修正后可重试，或拒绝释放占用" icon="!" />
</div>

<div class="card" style="margin-top: 18px">
  <div class="card-header">
    <h2>最近申请</h2>
    <a class="link" href="/admin/applications">查看全部 →</a>
  </div>
  {#if data.recent.length}
    <div class="table-wrap">
      <table>
        <thead>
          <tr><th>申请人</th><th>用户名</th><th>请求服务</th><th>状态</th><th>提交时间</th><th></th></tr>
        </thead>
        <tbody>
          {#each data.recent as item (item.id)}
            <tr>
              <td><div class="primary-text">{item.email}</div></td>
              <td><code>{item.username}</code></td>
              <td>
                <div class="group-list">
                  {#each item.requestedServices ?? [] as service}<span class="group-chip">{service}</span>{/each}
                </div>
              </td>
              <td><Badge status={item.status} /></td>
              <td class="secondary-text">{fmtDate(item.createdAt)}</td>
              <td><a class="link" href={`/admin/applications/${item.id}`}>审阅</a></td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {:else}
    <div class="empty">目前没有注册申请。</div>
  {/if}
</div>
