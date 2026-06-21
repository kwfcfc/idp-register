<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { fmtDate } from '$lib/format';
  let { data } = $props();

  // Details is a raw JSON string; pretty-print it, falling back to the raw text.
  function pretty(details: string): string {
    if (!details) return '—';
    try {
      return JSON.stringify(JSON.parse(details));
    } catch {
      return details;
    }
  }
</script>

<svelte:head><title>审计日志 · Invite Console</title></svelte:head>

<div class="page-header">
  <div>
    <h1>审计日志</h1>
    <p>管理员操作记录（最近 200 条）。邀请码明文绝不写入日志（不变量 #3）。</p>
  </div>
</div>

<section class="card">
  {#if data.entries.length}
    <div class="table-wrap">
      <table>
        <thead>
          <tr><th>时间</th><th>操作人</th><th>操作</th><th>对象</th><th>详情</th></tr>
        </thead>
        <tbody>
          {#each data.entries as e (e.id)}
            <tr>
              <td class="secondary-text">{fmtDate(e.createdAt)}</td>
              <td>{e.actorEmail || e.actorSub}</td>
              <td><code>{e.action}</code></td>
              <td class="secondary-text">{e.targetType}{e.targetId ? ` · ${e.targetId}` : ''}</td>
              <td style="max-width:360px">
                <div
                  class="secondary-text"
                  style="white-space:nowrap;overflow:hidden;text-overflow:ellipsis"
                  title={pretty(e.details)}
                >
                  {pretty(e.details)}
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {:else}
    <div class="empty">暂无审计记录。</div>
  {/if}
</section>
