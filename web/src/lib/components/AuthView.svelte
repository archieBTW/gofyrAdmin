<script lang="ts">
  import { api, setToken } from '../api';
  import type { AdminUser } from '../types';

  let { onSuccess = (user: AdminUser) => {} } = $props<{
    onSuccess: (user: AdminUser) => void;
  }>();

  // Stage: 'login' | 'mfa_verify' | 'mfa_setup'
  let stage = $state<'login' | 'mfa_verify' | 'mfa_setup'>('login');
  let email = $state('');
  let password = $state('');
  let preAuthToken = $state('');
  let isSubmitting = $state(false);
  let errorMessage = $state<string | null>(null);

  // Forced password reset on first login
  let mustResetPassword = $state(false);
  let newPassword = $state('');
  let confirmPassword = $state('');
  let showPassword = $state(false);

  // OTP segmented inputs (6 digits)
  let otpDigits = $state<string[]>(['', '', '', '', '', '']);
  let otpInputs: HTMLInputElement[] = [];
  let useRecoveryCode = $state(false);
  let recoveryCodeInput = $state('');

  // MFA Setup credentials
  let mfaSecret = $state('');
  let mfaQRCode = $state('');
  let recoveryCodes = $state<string[]>([]);
  let copiedSecret = $state(false);
  let copiedRecoveryCodes = $state(false);

  async function handleLoginSubmit(e: Event) {
    e.preventDefault();
    if (!email || !password) return;

    isSubmitting = true;
    errorMessage = null;

    try {
      const res = await api.login(email, password);
      preAuthToken = res.pre_auth_token;
      mustResetPassword = !!res.must_reset_password;

      if (res.must_reset_password || !res.mfa_enrolled) {
        // Needs initial security setup (password reset + MFA setup)
        await startMFASetup();
      } else {
        stage = 'mfa_verify';
        otpDigits = ['', '', '', '', '', ''];
        setTimeout(() => otpInputs[0]?.focus(), 100);
      }
    } catch (err: any) {
      errorMessage = err.message || 'Login failed';
    } finally {
      isSubmitting = false;
    }
  }

  async function startMFASetup() {
    isSubmitting = true;
    errorMessage = null;
    try {
      const setupRes = await api.setupMFA(preAuthToken);
      mfaSecret = setupRes.secret;
      mfaQRCode = setupRes.qr_code_base64;
      recoveryCodes = setupRes.recovery_codes;
      stage = 'mfa_setup';
      otpDigits = ['', '', '', '', '', ''];
      setTimeout(() => otpInputs[0]?.focus(), 100);
    } catch (err: any) {
      errorMessage = err.message || 'Failed to initialize MFA setup';
    } finally {
      isSubmitting = false;
    }
  }

  function handleOtpInput(index: number, e: Event) {
    const input = e.target as HTMLInputElement;
    const val = input.value;

    if (val.length > 1) {
      handleOtpPaste(val);
      return;
    }

    otpDigits[index] = val;

    if (val && index < 5) {
      otpInputs[index + 1]?.focus();
    }

    checkAndSubmitOtp();
  }

  function handleOtpKeydown(index: number, e: KeyboardEvent) {
    if (e.key === 'Backspace' && !otpDigits[index] && index > 0) {
      otpInputs[index - 1]?.focus();
    }
  }

  function handleOtpPaste(pastedText: string) {
    const clean = pastedText.replace(/[^0-9]/g, '');
    if (!clean) return;

    for (let i = 0; i < 6; i++) {
      otpDigits[i] = clean[i] || '';
    }

    const nextIndex = Math.min(clean.length, 5);
    otpInputs[nextIndex]?.focus();

    if (clean.length >= 6) {
      checkAndSubmitOtp();
    }
  }

  function checkAndSubmitOtp() {
    const code = otpDigits.join('');
    if (code.length === 6) {
      // If forced password reset is active, require password inputs before auto-submitting
      if (mustResetPassword) {
        if (!newPassword || newPassword.length < 8 || newPassword !== confirmPassword) {
          return;
        }
      }
      submitVerifyMFA(code);
    }
  }

  async function submitVerifyMFA(codeToVerify?: string) {
    const code = codeToVerify || (useRecoveryCode ? recoveryCodeInput.trim() : otpDigits.join(''));
    if (!code) return;

    if (mustResetPassword) {
      if (!newPassword || newPassword.length < 8) {
        errorMessage = 'New password must be at least 8 characters long';
        return;
      }
      if (newPassword !== confirmPassword) {
        errorMessage = 'Passwords do not match';
        return;
      }
    }

    isSubmitting = true;
    errorMessage = null;

    try {
      const res = await api.verifyMFA(
        preAuthToken,
        code,
        mustResetPassword ? newPassword : undefined
      );
      setToken(res.token);
      onSuccess(res.user);
    } catch (err: any) {
      errorMessage = err.message || 'Invalid authentication code';
      otpDigits = ['', '', '', '', '', ''];
      otpInputs[0]?.focus();
    } finally {
      isSubmitting = false;
    }
  }

  function copySecret() {
    navigator.clipboard.writeText(mfaSecret);
    copiedSecret = true;
    setTimeout(() => (copiedSecret = false), 2000);
  }

  function copyAllRecoveryCodes() {
    navigator.clipboard.writeText(recoveryCodes.join('\n'));
    copiedRecoveryCodes = true;
    setTimeout(() => (copiedRecoveryCodes = false), 2000);
  }

  function backToLogin() {
    stage = 'login';
    preAuthToken = '';
    errorMessage = null;
    useRecoveryCode = false;
    recoveryCodeInput = '';
    mustResetPassword = false;
    newPassword = '';
    confirmPassword = '';
  }
</script>

<div class="min-h-screen flex items-center justify-center p-4 bg-[#090d16] bg-[radial-gradient(ellipse_80%_80%_at_50%_-20%,rgba(99,102,241,0.15),rgba(255,255,255,0))]">
  <div class="w-full max-w-md">
    <!-- Brand Header -->
    <div class="text-center mb-8">
      <h1 class="text-3xl font-black tracking-tight text-white font-mono">
        gofyr<span style="color: var(--accent-primary, #a78bfa);">Admin</span>
      </h1>
      <p class="text-xs text-slate-400 mt-2 font-medium">
        Gin + GORM Full-Stack Workbench & Panel
      </p>
    </div>

    <!-- Auth Card -->
    <div class="bg-[#111221]/90 backdrop-blur-xl border border-[#212338] rounded-2xl p-7 shadow-2xl relative overflow-hidden">
      <!-- Glow Accent Line -->
      <div class="absolute top-0 left-0 right-0 h-1" style="background: linear-gradient(90deg, var(--accent-primary, #a78bfa), var(--accent-hover, #8b5cf6));"></div>

      {#if errorMessage}
        <div class="mb-5 p-3.5 rounded-xl bg-rose-950/70 border border-rose-800/80 text-rose-300 text-sm flex items-start gap-2.5">
          <svg class="w-5 h-5 flex-shrink-0 text-rose-400 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span class="leading-snug">{errorMessage}</span>
        </div>
      {/if}

      <!-- STAGE 1: LOGIN -->
      {#if stage === 'login'}
        <form onsubmit={handleLoginSubmit} class="space-y-4">
          <div>
            <label for="email" class="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
              Email Address
            </label>
            <input
              id="email"
              type="email"
              required
              bind:value={email}
              placeholder="admin@example.com"
              class="w-full px-4 py-2.5 rounded-xl bg-slate-800/90 border border-slate-700/80 text-white placeholder-slate-500 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 text-sm transition-all"
            />
          </div>

          <div>
            <label for="password" class="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
              Password
            </label>
            <input
              id="password"
              type="password"
              required
              bind:value={password}
              placeholder="••••••••••••"
              class="w-full px-4 py-2.5 rounded-xl bg-slate-800/90 border border-slate-700/80 text-white placeholder-slate-500 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 text-sm transition-all"
            />
          </div>

          <button
            type="submit"
            disabled={isSubmitting}
            class="w-full py-3 px-4 rounded-xl font-bold text-sm transition-all flex items-center justify-center gap-2 disabled:opacity-50 mt-6 shadow-lg hover:opacity-90"
            style="background-color: var(--accent-primary, #a78bfa); color: #0b0c16;"
          >
            {#if isSubmitting}
              <svg class="animate-spin w-4 h-4 text-white" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
              </svg>
              <span>Authenticating...</span>
            {:else}
              <span>Continue to Verification</span>
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
              </svg>
            {/if}
          </button>
        </form>

      <!-- STAGE 2: MFA VERIFICATION (FOR ALREADY ENROLLED ADMINS) -->
      {:else if stage === 'mfa_verify'}
        <div class="space-y-6">
          <div class="text-center">
            <div class="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-indigo-950 border border-indigo-700/40 text-indigo-400 mb-3">
              <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
              </svg>
            </div>
            <h2 class="text-lg font-bold text-white">Two-Factor Authentication</h2>
            <p class="text-xs text-slate-400 mt-1">
              Enter the 6-digit code from your authenticator app for <span class="text-slate-300 font-medium">{email}</span>
            </p>
          </div>

          {#if !useRecoveryCode}
            <!-- 6-digit Segmented OTP inputs -->
            <div>
              <div class="flex justify-center gap-2">
                {#each [0, 1, 2, 3, 4, 5] as idx}
                  <input
                    bind:this={otpInputs[idx]}
                    type="text"
                    inputmode="numeric"
                    pattern="[0-9]*"
                    maxlength="1"
                    value={otpDigits[idx]}
                    oninput={(e) => handleOtpInput(idx, e)}
                    onkeydown={(e) => handleOtpKeydown(idx, e)}
                    onpaste={(e) => {
                      e.preventDefault();
                      const text = e.clipboardData?.getData('text') || '';
                      handleOtpPaste(text);
                    }}
                    class="w-12 h-14 text-center text-xl font-mono font-bold rounded-xl bg-slate-800/90 border border-slate-700 text-white focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 transition-all"
                  />
                {/each}
              </div>

              <div class="mt-4 flex items-center justify-between text-xs">
                <button
                  type="button"
                  onclick={() => (useRecoveryCode = true)}
                  class="text-indigo-400 hover:text-indigo-300 font-medium underline"
                >
                  Use recovery code instead
                </button>
                <button
                  type="button"
                  onclick={backToLogin}
                  class="text-slate-500 hover:text-slate-400 font-medium"
                >
                  Back to login
                </button>
              </div>
            </div>
          {:else}
            <!-- Recovery Code Form -->
            <form onsubmit={(e) => { e.preventDefault(); submitVerifyMFA(); }} class="space-y-4">
              <div>
                <label for="recovery-code" class="block text-xs font-semibold uppercase tracking-wider text-slate-300 mb-1.5">
                  Single-Use Recovery Code
                </label>
                <input
                  id="recovery-code"
                  type="text"
                  required
                  bind:value={recoveryCodeInput}
                  placeholder="e.g. 4F7A-99B2"
                  class="w-full px-4 py-2.5 rounded-xl bg-slate-800/90 border border-slate-700/80 text-white font-mono placeholder-slate-500 focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 text-sm uppercase text-center tracking-widest transition-all"
                />
              </div>

              <button
                type="submit"
                disabled={isSubmitting || !recoveryCodeInput.trim()}
                class="w-full py-2.5 px-4 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold text-sm transition-all disabled:opacity-50"
              >
                Verify Recovery Code
              </button>

              <div class="flex items-center justify-between text-xs">
                <button
                  type="button"
                  onclick={() => (useRecoveryCode = false)}
                  class="text-indigo-400 hover:text-indigo-300 font-medium underline"
                >
                  Use 6-digit TOTP code
                </button>
                <button
                  type="button"
                  onclick={backToLogin}
                  class="text-slate-500 hover:text-slate-400 font-medium"
                >
                  Back to login
                </button>
              </div>
            </form>
          {/if}
        </div>

      <!-- STAGE 3: INITIAL SECURITY SETUP (MANDATORY PASSWORD RESET & MFA ENROLLMENT) -->
      {:else if stage === 'mfa_setup'}
        <div class="space-y-5">
          <div class="text-center">
            <span class="inline-block px-2.5 py-0.5 rounded-full text-[11px] font-semibold bg-amber-950 text-amber-400 border border-amber-800/60 mb-2">
              {#if mustResetPassword}
                Required Initial Setup
              {:else}
                First-Time Security Setup
              {/if}
            </span>
            <h2 class="text-lg font-bold text-white">
              {#if mustResetPassword}
                Reset Password & Setup MFA
              {:else}
                Enroll TOTP Authenticator
              {/if}
            </h2>
            <p class="text-xs text-slate-400 mt-1">
              {#if mustResetPassword}
                Change your default password and pair your Authenticator app to secure this account.
              {:else}
                Scan this QR code with Google Authenticator, 1Password, or Authy
              {/if}
            </p>
          </div>

          <!-- Step 1: Password Reset (if mandatory) -->
          {#if mustResetPassword}
            <div class="p-4 rounded-xl bg-slate-900/90 border border-slate-800 space-y-3">
              <div class="text-xs font-semibold text-slate-200 uppercase tracking-wider flex items-center gap-1.5">
                <svg class="w-4 h-4 text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
                </svg>
                Step 1: Choose New Password
              </div>

              <div>
                <label for="new-password" class="block text-[11px] font-medium text-slate-400 mb-1">
                  New Password (min. 8 characters)
                </label>
                <input
                  id="new-password"
                  type={showPassword ? 'text' : 'password'}
                  required
                  minlength="8"
                  bind:value={newPassword}
                  placeholder="Enter secure password..."
                  class="w-full px-3.5 py-2 rounded-lg bg-slate-800 border border-slate-700 text-white placeholder-slate-500 focus:border-indigo-500 text-sm transition-all"
                />
              </div>

              <div>
                <label for="confirm-password" class="block text-[11px] font-medium text-slate-400 mb-1">
                  Confirm New Password
                </label>
                <input
                  id="confirm-password"
                  type={showPassword ? 'text' : 'password'}
                  required
                  bind:value={confirmPassword}
                  placeholder="Re-type new password..."
                  class="w-full px-3.5 py-2 rounded-lg bg-slate-800 border border-slate-700 text-white placeholder-slate-500 focus:border-indigo-500 text-sm transition-all"
                />
              </div>

              <div class="flex items-center justify-between pt-1">
                <label class="flex items-center gap-2 cursor-pointer text-xs text-slate-400 select-none">
                  <input type="checkbox" bind:checked={showPassword} class="rounded bg-slate-800 border-slate-700 text-indigo-500 focus:ring-0" />
                  Show passwords
                </label>
                {#if newPassword && confirmPassword}
                  <span class="text-xs font-medium {newPassword === confirmPassword && newPassword.length >= 8 ? 'text-emerald-400' : 'text-rose-400'}">
                    {#if newPassword !== confirmPassword}
                      Passwords do not match
                    {:else if newPassword.length < 8}
                      Password too short
                    {:else}
                      ✓ Passwords match
                    {/if}
                  </span>
                {/if}
              </div>
            </div>
          {/if}

          <!-- Step 2: Authenticator QR Code & Secret -->
          <div class="space-y-3">
            {#if mustResetPassword}
              <div class="text-xs font-semibold text-slate-200 uppercase tracking-wider flex items-center gap-1.5">
                <svg class="w-4 h-4 text-indigo-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />
                </svg>
                Step 2: Connect Authenticator App
              </div>
            {/if}

            {#if mfaQRCode}
              <div class="flex justify-center">
                <div class="p-3 bg-white rounded-2xl shadow-xl">
                  <img src={mfaQRCode} alt="MFA QR Code" class="w-40 h-40 rounded-lg" />
                </div>
              </div>
            {/if}

            <!-- Secret Manual Entry -->
            <div class="p-2.5 rounded-xl bg-slate-800/70 border border-slate-700/70 flex items-center justify-between gap-2">
              <div class="min-w-0">
                <div class="text-[10px] text-slate-400 uppercase tracking-wider font-semibold">Manual Secret Key</div>
                <div class="font-mono text-xs text-indigo-300 truncate font-bold">{mfaSecret}</div>
              </div>
              <button
                type="button"
                onclick={copySecret}
                class="px-2.5 py-1 rounded-lg bg-slate-700 hover:bg-slate-600 text-xs text-white font-medium flex items-center gap-1 transition-colors flex-shrink-0"
              >
                {#if copiedSecret}
                  <span class="text-emerald-400">Copied!</span>
                {:else}
                  <span>Copy</span>
                {/if}
              </button>
            </div>

            <!-- Backup Recovery Codes to Save -->
            <div>
              <div class="flex items-center justify-between mb-1.5">
                <span class="text-[11px] font-semibold text-slate-300 uppercase tracking-wider">
                  Save Your Backup Recovery Codes
                </span>
                <button
                  type="button"
                  onclick={copyAllRecoveryCodes}
                  class="text-xs text-indigo-400 hover:text-indigo-300 font-medium underline"
                >
                  {copiedRecoveryCodes ? 'Copied All!' : 'Copy All'}
                </button>
              </div>
              <div class="grid grid-cols-2 gap-1.5 p-2.5 rounded-xl bg-slate-950/60 border border-slate-800 font-mono text-[11px] text-slate-300">
                {#each recoveryCodes as code}
                  <div class="px-2 py-1 bg-slate-900 rounded border border-slate-800 text-center select-all">
                    {code}
                  </div>
                {/each}
              </div>
            </div>
          </div>

          <!-- Step 3: Verification OTP Input to Confirm -->
          <div class="pt-3 border-t border-slate-800 space-y-3">
            <label for="otp-confirm" class="block text-xs font-semibold text-center text-slate-300">
              {#if mustResetPassword}
                Step 3: Enter 6-digit Authenticator code to finish:
              {:else}
                Enter 6-digit code to activate MFA:
              {/if}
            </label>
            <div class="flex justify-center gap-2">
              {#each [0, 1, 2, 3, 4, 5] as idx}
                <input
                  id={idx === 0 ? "otp-confirm" : undefined}
                  bind:this={otpInputs[idx]}
                  type="text"
                  inputmode="numeric"
                  pattern="[0-9]*"
                  maxlength="1"
                  value={otpDigits[idx]}
                  oninput={(e) => handleOtpInput(idx, e)}
                  onkeydown={(e) => handleOtpKeydown(idx, e)}
                  class="w-11 h-12 text-center text-lg font-mono font-bold rounded-xl bg-slate-800/90 border border-slate-700 text-white focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 transition-all"
                />
              {/each}
            </div>

            <!-- Submit Button -->
            <button
              type="button"
              onclick={() => submitVerifyMFA()}
              disabled={isSubmitting || (mustResetPassword && (!newPassword || newPassword.length < 8 || newPassword !== confirmPassword)) || otpDigits.join('').length < 6}
              class="w-full py-3 px-4 rounded-xl font-bold text-sm transition-all flex items-center justify-center gap-2 disabled:opacity-50 shadow-lg hover:opacity-90 mt-2"
              style="background-color: var(--accent-primary, #a78bfa); color: #0b0c16;"
            >
              {#if isSubmitting}
                <svg class="animate-spin w-4 h-4 text-white" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
                </svg>
                <span>Saving Security Settings...</span>
              {:else}
                <span>Complete Setup & Sign In</span>
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                </svg>
              {/if}
            </button>

            <div class="text-center pt-1">
              <button
                type="button"
                onclick={backToLogin}
                class="text-xs text-slate-500 hover:text-slate-400 font-medium"
              >
                Cancel and return to login
              </button>
            </div>
          </div>
        </div>
      {/if}
    </div>
  </div>
</div>
