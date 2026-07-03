<!-- SPDX-License-Identifier: GPL-3.0-or-later -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/state';
  import { apiSend, ApiError } from '$lib/api';

  let { data } = $props();

  // Optional Cloudflare Turnstile. The site key is a public value served by the
  // backend via GET /api/form (ADR-0016), so the same build works for every
  // deployment; when empty, the backend has no secret either and skips
  // verification.
  const siteKey = $derived(data.turnstileSiteKey);

  let email = $state('');
  let username = $state('');
  let reviewText = $state('');
  let inviteCode = $state(page.url.searchParams.get('token') ?? '');
  // Single-select for now (ADR-0012); '' means no service chosen. The picker
  // maps an admin-curated profile to a user-facing "service" — the public form
  // only ever sees the opaque id, never the underlying IdP groups (invariant #4).
  let service = $state('');
  let turnstileToken = $state('');
  let termsAccepted = $state(false);

  let submitting = $state(false);
  let done = $state(false);
  let error = $state('');

  onMount(() => {
    if (!siteKey) return;
    // Turnstile invokes this global with the solved token; feed it into state.
    (window as unknown as { onTurnstileToken?: (t: string) => void }).onTurnstileToken = (t) => {
      turnstileToken = t;
    };
    const s = document.createElement('script');
    s.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js';
    s.async = true;
    s.defer = true;
    document.head.appendChild(s);
  });

  async function submit(event: SubmitEvent) {
    event.preventDefault();
    error = '';
    submitting = true;
    try {
      await apiSend('POST', '/api/register', {
        email: email.trim(),
        username: username.trim(),
        reviewText: reviewText.trim(),
        inviteCode: inviteCode.trim(),
        services: service ? [service] : [],
        turnstileToken,
        termsAccepted
      });
      done = true;
    } catch (e) {
      error = e instanceof ApiError ? e.message : '提交失败，请稍后重试。';
    } finally {
      submitting = false;
    }
  }
</script>

<svelte:head><title>申请注册 · idp-register</title></svelte:head>

<main class="center-shell">
  <section class="center-card">
    <div class="brand-mark" style="display:grid;place-items:center">◈</div>
    <p class="eyebrow">REGISTRATION</p>
    <h1>申请注册</h1>

    {#if done}
      <div class="alert success" style="margin-top:20px">
        申请已收到。如果通过审核，你会收到一封用于设置账户的邮件。
      </div>
      <p class="lede" style="margin-bottom:0">你可以关闭此页面了。</p>
    {:else}
      <p class="lede">填写下面的信息提交注册申请。持有邀请码可被自动批准；否则将由管理员人工审核。</p>

      {#if data.rulesText}
        <!-- Deployer-provided registration rules (runtime config, GET /api/form).
             Rendered as plain text with line breaks preserved — never as HTML. -->
        <div class="rules-panel">{data.rulesText}</div>
      {/if}

      {#if error}<div class="alert error">{error}</div>{/if}

      <form onsubmit={submit}>
        <div class="field">
          <label for="email">邮箱</label>
          <input id="email" type="email" bind:value={email} required placeholder="you@example.com" />
        </div>
        <div class="field">
          <label for="username">用户名</label>
          <input id="username" bind:value={username} placeholder="希望使用的用户名" />
        </div>
        {#if data.services.length}
          <div class="field">
            <label for="service">申请的服务</label>
            <select id="service" bind:value={service}>
              <option value="">不指定（由管理员决定）</option>
              {#each data.services as svc (svc.id)}
                <option value={svc.id}>{svc.label}</option>
              {/each}
            </select>
            <div class="help">
              {#if service}
                {data.services.find((s) => s.id === service)?.description || '选择你希望注册使用的服务。'}
              {:else}
                选择你希望注册使用的服务；具体权限在审批时由管理员确定。
              {/if}
            </div>
          </div>
        {/if}
        <div class="field">
          <label for="reviewText">申请说明</label>
          <textarea id="reviewText" bind:value={reviewText} placeholder="简要说明你是谁、为什么申请。"
          ></textarea>
        </div>
        <div class="field">
          <label for="inviteCode">邀请码（可选）</label>
          <input id="inviteCode" bind:value={inviteCode} placeholder="若持有邀请码请填写" />
          <div class="help">有效邀请码可自动批准；留空则进入人工审核。</div>
        </div>

        {#if data.requiresConsent}
          <label class="consent-row">
            <input type="checkbox" bind:checked={termsAccepted} required />
            <span>
              我已阅读并同意{#if data.rulesText}上述注册规则{/if}{#if data.rulesText && data.termsUrl}与{/if}{#if data.termsUrl}<a
                  href={data.termsUrl}
                  target="_blank"
                  rel="noopener noreferrer">服务条款</a
                >{/if}。
            </span>
          </label>
        {/if}

        {#if siteKey}
          <div
            class="cf-turnstile"
            style="margin-top:16px"
            data-sitekey={siteKey}
            data-callback="onTurnstileToken"
          ></div>
        {/if}

        <div class="form-actions">
          <button class="button primary" type="submit" disabled={submitting}>
            {submitting ? '提交中…' : '提交申请'}
          </button>
        </div>
      </form>

      <div class="security-note">
        <span>●</span>
        密码与多因素认证均由身份提供方在激活邮件中设置，本服务不接触你的密码。
      </div>
    {/if}
  </section>
</main>
