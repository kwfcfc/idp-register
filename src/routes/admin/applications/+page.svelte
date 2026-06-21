<script lang="ts">
  import Badge from '$lib/components/Badge.svelte';
  let { data } = $props();
  const date = (value: string | Date) => new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value));
  const filters = [
    ['pending','待审核'], ['provisioning_failed','创建失败'], ['needs_changes','需补充'],
    ['approved','已批准'], ['rejected','已拒绝'], ['provisioning','创建中']
  ];
</script>

<svelte:head><title>注册申请 · Invite Console</title></svelte:head>
<div class="page-header"><div><h1>注册申请</h1><p>人工审阅申请说明，并在批准时选择 Rauthy 权限模板。</p></div></div>
<div class="filters">{#each filters as filter}<a class:active={data.status === filter[0]} class="filter" href={`/admin/applications?status=${filter[0]}`}>{filter[1]}</a>{/each}</div>
<section class="card">
  {#if data.applications.length}
    <div class="table-wrap"><table>
      <thead><tr><th>申请人</th><th>用户名</th><th>申请说明</th><th>服务</th><th>验证</th><th>提交时间</th><th></th></tr></thead>
      <tbody>{#each data.applications as item}<tr>
        <td><div class="primary-text">{item.email}</div><div class="secondary-text">{item.invite_id ? '使用邀请码' : '普通申请'}</div></td>
        <td><code>{item.username}</code></td>
        <td style="max-width:320px"><div style="white-space:nowrap;overflow:hidden;text-overflow:ellipsis">{item.review_text}</div></td>
        <td><div class="group-list">{#each item.requested_services as service}<span class="group-chip">{service}</span>{/each}</div></td>
        <td>{#if item.email_verified_at}<span class="badge green">邮箱已验证</span>{:else}<span class="badge amber">待验证</span>{/if}</td>
        <td class="secondary-text">{date(item.created_at)}</td>
        <td><a class="link" href={`/admin/applications/${item.id}`}>查看</a></td>
      </tr>{/each}</tbody>
    </table></div>
  {:else}<div class="empty">这个状态下没有申请。</div>{/if}
</section>
