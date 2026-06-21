<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import Badge from '$lib/components/Badge.svelte';
  import { fmtDate } from '$lib/format';
  let { data } = $props();

  const filters: [string, string][] = [
    ['pending', '待审核'],
    ['provisioning_failed', '创建失败'],
    ['needs_changes', '需补充'],
    ['approved', '已批准'],
    ['rejected', '已拒绝'],
    ['provisioning', '创建中']
  ];
</script>

<svelte:head><title>注册申请 · Invite Console</title></svelte:head>

<div class="page-header">
  <div>
    <h1>注册申请</h1>
    <p>人工审阅申请说明，并在批准时选择权限模板（服务端展开为目标 IdP groups）。</p>
  </div>
</div>

<div class="filters">
  {#each filters as [value, label] (value)}
    <a class:active={data.status === value} class="filter" href={`/admin/applications?status=${value}`}>
      {label}
    </a>
  {/each}
</div>

<section class="card">
  {#if data.applications.length}
    <div class="table-wrap">
      <table>
        <thead>
          <tr><th>申请人</th><th>用户名</th><th>申请说明</th><th>服务</th><th>来源</th><th>提交时间</th><th></th></tr>
        </thead>
        <tbody>
          {#each data.applications as item (item.id)}
            <tr>
              <td>
                <div class="primary-text">{item.email}</div>
                <div class="secondary-text">{item.tokenId ? '使用邀请码' : '普通申请'}</div>
              </td>
              <td><code>{item.username}</code></td>
              <td style="max-width:320px">
                <div style="white-space:nowrap;overflow:hidden;text-overflow:ellipsis">{item.reviewText}</div>
              </td>
              <td>
                <div class="group-list">
                  {#each item.requestedServices ?? [] as service}<span class="group-chip">{service}</span>{/each}
                </div>
              </td>
              <td><Badge status={item.status} /></td>
              <td class="secondary-text">{fmtDate(item.createdAt)}</td>
              <td><a class="link" href={`/admin/applications/${item.id}`}>查看</a></td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {:else}
    <div class="empty">这个状态下没有申请。</div>
  {/if}
</section>
