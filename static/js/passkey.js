// passkey.js — WebAuthn browser implementation
// Handles passkey registration and login via the WebAuthn API.
// No dependencies. Pure vanilla JS.
// Works with Face ID, Touch ID, Windows Hello, YubiKey.

'use strict';

// ─── Utility ────────────────────────────────────────────────────────────────

function base64urlToBuffer(base64url) {
    const base64 = base64url.replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64.padEnd(base64.length + (4 - base64.length % 4) % 4, '=');
    const binary = atob(padded);
    const buffer = new ArrayBuffer(binary.length);
    const bytes = new Uint8Array(buffer);
    for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i);
    }
    return buffer;
}

function bufferToBase64url(buffer) {
    const bytes = new Uint8Array(buffer);
    let binary = '';
    for (let i = 0; i < bytes.byteLength; i++) {
        binary += String.fromCharCode(bytes[i]);
    }
    return btoa(binary)
        .replace(/\+/g, '-')
        .replace(/\//g, '_')
        .replace(/=/g, '');
}

function encodeOptions(options) {
    if (options.challenge) {
        options.challenge = base64urlToBuffer(options.challenge);
    }
    if (options.user?.id) {
        options.user.id = base64urlToBuffer(options.user.id);
    }
    if (options.excludeCredentials) {
        options.excludeCredentials = options.excludeCredentials.map(c => ({
            ...c,
            id: base64urlToBuffer(c.id),
        }));
    }
    if (options.allowCredentials) {
        options.allowCredentials = options.allowCredentials.map(c => ({
            ...c,
            id: base64urlToBuffer(c.id),
        }));
    }
    return options;
}

function encodeCredential(credential) {
    const result = {
        id:    credential.id,
        rawId: bufferToBase64url(credential.rawId),
        type:  credential.type,
        response: {},
    };

    if (credential.response.clientDataJSON) {
        result.response.clientDataJSON = bufferToBase64url(credential.response.clientDataJSON);
    }
    if (credential.response.attestationObject) {
        result.response.attestationObject = bufferToBase64url(credential.response.attestationObject);
    }
    if (credential.response.authenticatorData) {
        result.response.authenticatorData = bufferToBase64url(credential.response.authenticatorData);
        result.response.signature = bufferToBase64url(credential.response.signature);
        if (credential.response.userHandle) {
            result.response.userHandle = bufferToBase64url(credential.response.userHandle);
        }
    }

    return result;
}

// ─── Support Check ───────────────────────────────────────────────────────────

function passkeySupported() {
    return !!(navigator.credentials &&
              window.PublicKeyCredential &&
              typeof window.PublicKeyCredential === 'function');
}

function getDeviceName() {
    const ua = navigator.userAgent;
    if (/iPhone/.test(ua))  return 'iPhone';
    if (/iPad/.test(ua))    return 'iPad';
    if (/Android/.test(ua)) return 'Android';
    if (/Mac/.test(ua))     return 'Mac';
    if (/Windows/.test(ua)) return 'Windows PC';
    return 'Unknown Device';
}

// ─── Registration ────────────────────────────────────────────────────────────

async function registerPasskey(deviceName) {
    const beginResp = await fetch('/auth/passkey/register/begin', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
    });

    if (!beginResp.ok) {
        const err = await beginResp.json();
        throw new Error(err.error || 'Registration failed');
    }

    const { flow_id, options } = await beginResp.json();
    const encodedOptions = encodeOptions(options.publicKey);

    let credential;
    try {
        credential = await navigator.credentials.create({ publicKey: encodedOptions });
    } catch (e) {
        if (e.name === 'NotAllowedError') throw new Error('Passkey creation cancelled');
        throw new Error('Passkey creation failed: ' + e.message);
    }

    const finishResp = await fetch('/auth/passkey/register/finish', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            flow_id:     flow_id,
            device_name: deviceName || getDeviceName(),
            credential:  encodeCredential(credential),
        }),
    });

    if (!finishResp.ok) {
        const err = await finishResp.json();
        throw new Error(err.error || 'Registration failed');
    }

    return await finishResp.json();
}

// ─── Login ───────────────────────────────────────────────────────────────────

async function loginWithPasskey() {
    const beginResp = await fetch('/auth/passkey/login/begin', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
    });

    if (!beginResp.ok) {
        const err = await beginResp.json();
        throw new Error(err.error || 'Login failed');
    }

    const { flow_id, options } = await beginResp.json();
    const encodedOptions = encodeOptions(options.publicKey);

    let assertion;
    try {
        assertion = await navigator.credentials.get({ publicKey: encodedOptions });
    } catch (e) {
        if (e.name === 'NotAllowedError') throw new Error('Passkey login cancelled');
        throw new Error('Passkey login failed: ' + e.message);
    }

    const finishResp = await fetch('/auth/passkey/login/finish', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            flow_id:    flow_id,
            credential: encodeCredential(assertion),
        }),
    });

    if (!finishResp.ok) {
        const err = await finishResp.json();
        throw new Error(err.error || 'Login failed');
    }

    const result = await finishResp.json();
    if (result.redirect) {
        window.location.href = result.redirect;
    }

    return result;
}

// ─── DOM Wiring ──────────────────────────────────────────────────────────────

document.addEventListener('DOMContentLoaded', function () {

    // Hide passkey UI if browser does not support it
    if (!passkeySupported()) {
        const section = document.getElementById('passkeySection');
        if (section) section.style.display = 'none';
        const registerSection = document.getElementById('passkeyRegisterSection');
        if (registerSection) registerSection.style.display = 'none';
        return;
    }

    // ── Login button ──
    const loginBtn = document.getElementById('passkeyLoginBtn');
    if (loginBtn) {
        loginBtn.addEventListener('click', async function () {
            const status = document.getElementById('passkeyLoginStatus');
            loginBtn.disabled = true;
            loginBtn.textContent = 'Verificando...';

            try {
                await loginWithPasskey();
            } catch (e) {
                if (status) {
                    status.textContent = e.message;
                    status.className = 'passkey-status passkey-error';
                }
                loginBtn.disabled = false;
                loginBtn.innerHTML = '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="margin-right:8px"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>Continuar con Passkey';
            }
        });
    }

    // ── Register button ──
    const registerBtn = document.getElementById('registerPasskeyBtn');
    if (registerBtn) {
        registerBtn.addEventListener('click', async function () {
            const deviceName = document.getElementById('passkeyDeviceName')?.value || '';
            const status = document.getElementById('passkeyStatus');

            registerBtn.disabled = true;
            registerBtn.textContent = 'Verificando...';

            try {
                await registerPasskey(deviceName);
                if (status) {
                    status.textContent = '✓ Passkey registrado correctamente';
                    status.className = 'passkey-status passkey-success';
                }
                setTimeout(() => window.location.reload(), 1500);
            } catch (e) {
                if (status) {
                    status.textContent = e.message;
                    status.className = 'passkey-status passkey-error';
                }
                registerBtn.disabled = false;
                registerBtn.textContent = 'Agregar Passkey';
            }
        });
    }
});
