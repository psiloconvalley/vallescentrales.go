/* ============================================================================
   Wizard Client Script — Auto-save, exit handler, progress feedback
   Vanilla ES6+, zero dependencies, zero external loads.
   ============================================================================ */

(function () {
    'use strict';

    const AUTOSAVE_DEBOUNCE_MS = 1000;
    const AUTOSAVE_INDICATOR_HIDE_MS = 3000;

    const form = document.querySelector('[data-wizard-form]');
    const exitBtn = document.querySelector('[data-wizard-exit]');
    const indicator = document.querySelector('[data-wizard-autosave]');
    const indicatorText = indicator ? indicator.querySelector('.wizard-autosave-text') : null;

    const autosaveUrl = form ? form.getAttribute('data-autosave-url') : null;

    let debounceTimer = null;
    let hideTimer = null;
    let isSaving = false;

    function getCSRFToken() {
        const meta = document.querySelector('meta[name="csrf-token"]');
        if (meta && meta.content) return meta.content;
        const input = document.querySelector('input[name="csrf_token"]');
        return input ? input.value : '';
    }

    function getFormJSON() {
        if (!form) return {};
        const data = {};
        const formData = new FormData(form);

        // Detect current step from action URL or input (e.g. /publicar/paso/1)
        const match = window.location.pathname.match(/paso\/(\d+)/);
        if (match) {
            data['_step'] = parseInt(match[1], 10);
        }

        formData.forEach((value, key) => {
            if (key === 'csrf_token') return;
            // Ignore file inputs in standard JSON autosave
            if (value instanceof File) return;
            data[key] = value;
        });

        return data;
    }

    function showIndicator(text, state) {
        if (!indicator) return;
        if (indicatorText) indicatorText.textContent = text;
        indicator.className = 'wizard-autosave-indicator is-visible';
        if (state) indicator.classList.add('is-' + state);
        clearTimeout(hideTimer);
        hideTimer = setTimeout(() => {
            indicator.classList.remove('is-visible', 'is-saving', 'is-saved', 'is-error');
        }, AUTOSAVE_INDICATOR_HIDE_MS);
    }

    async function autosave() {
        if (!autosaveUrl || !form || isSaving) return;
        isSaving = true;

        const payload = getFormJSON();
        const csrfToken = getCSRFToken();

        try {
            showIndicator('Guardando…', 'saving');
            const res = await fetch(autosaveUrl, {
                method: 'POST',
                body: JSON.stringify(payload),
                credentials: 'same-origin',
                headers: {
                    'Content-Type': 'application/json',
                    'X-CSRF-Token': csrfToken,
                    'X-Requested-With': 'fetch'
                }
            });

            if (res.ok) {
                showIndicator('Borrador guardado', 'saved');
            } else {
                showIndicator('No se pudo guardar', 'error');
            }
        } catch (_) {
            showIndicator('Sin conexión', 'error');
        } finally {
            isSaving = false;
        }
    }

    function scheduleAutosave() {
        clearTimeout(debounceTimer);
        debounceTimer = setTimeout(autosave, AUTOSAVE_DEBOUNCE_MS);
    }

    if (form) {
        // Trigger autosave on typing or selecting options
        form.addEventListener('input', scheduleAutosave);
        form.addEventListener('change', scheduleAutosave);

        // Enter key should not accidentally submit the form
        form.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && e.target.tagName === 'INPUT' && e.target.type !== 'submit') {
                e.preventDefault();
            }
        });
    }

    // "Guardar y salir" button: save immediately, then navigate
    if (exitBtn) {
        exitBtn.addEventListener('click', async (e) => {
            e.preventDefault();
            const targetHref = exitBtn.getAttribute('href') || '/cuenta';
            const originalSpan = exitBtn.querySelector('span');
            const originalText = originalSpan ? originalSpan.textContent : '';

            if (originalSpan) originalSpan.textContent = 'Guardando…';
            exitBtn.classList.add('is-busy');

            clearTimeout(debounceTimer);
            if (form && autosaveUrl) {
                await autosave();
            }

            window.location.href = targetHref;
        });
    }

    // Page unload fallback
    window.addEventListener('pagehide', () => {
        clearTimeout(debounceTimer);
        if (navigator.sendBeacon && autosaveUrl && form) {
            const payload = JSON.stringify(getFormJSON());
            const blob = new Blob([payload], { type: 'application/json' });
            navigator.sendBeacon(autosaveUrl, blob);
        }
    });
})();
