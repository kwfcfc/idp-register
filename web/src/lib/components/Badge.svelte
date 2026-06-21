<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  let { status }: { status: string } = $props();

  const labels: Record<string, string> = {
    pending: '待审核',
    provisioning: '创建中',
    provisioning_failed: '创建失败',
    approved: '已批准',
    rejected: '已拒绝',
    needs_changes: '需补充',
    active: '有效',
    revoked: '已停用',
    exhausted: '已用尽',
    expired: '已过期'
  };
  const colors: Record<string, string> = {
    pending: 'amber',
    provisioning: 'blue',
    provisioning_failed: 'red',
    approved: 'green',
    rejected: 'gray',
    needs_changes: 'blue',
    active: 'green',
    revoked: 'red',
    exhausted: 'gray',
    expired: 'gray'
  };
</script>

<span class={`badge ${colors[status] || 'gray'}`}>{labels[status] || status}</span>
