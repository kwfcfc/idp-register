<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  let {
    label,
    value,
    note,
    icon
  }: { label: string; value: number | string; note: string; icon: string } = $props();
</script>

<div class="card stat-card">
  <div class="stat-icon">{icon}</div>
  <div class="stat-label">{label}</div>
  <div class="stat-value">{value}</div>
  <div class="stat-note">{note}</div>
</div>
