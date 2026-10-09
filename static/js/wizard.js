/* ============================================================================
   Wizard Client Script — Auto-save, progress feedback, keyboard shortcuts
   Vanilla ES6+, zero dependencies, zero external loads.
   ============================================================================ */

(function () {
    'use strict';

    const AUTOSAVE_DEBOUNCE_MS = 1200;
    const AUTOSAVE_INDICATOR_HIDE_MS = 2400;

    const form = document.querySelector('[data-wizard-form]');
    if (!form) return;

    const autosaveUrl = form.getAttribute('data-autosave-url');
    const indicator = document.querySelector('[data-wizard-autosave]');
    const indicatorText = indicator ? indicator.querySelector('.wizard-autosave-text') : null;

    let debounceTimer = null;
    let hideTimer = null;

    function showIndicator(text) {
        if (!indicator) return;
        if (indicatorText) indicatorText.textContent = text;
        indicator.classList.add('is-visible');
        clearTimeout(hideTimer);
        hideTimer = setTimeout(() => indicator.classList.remove('is-visible'), AUTOSAVE_INDICATOR_HIDE_MS);
    }

    async function autosave() {
        if (!autosaveUrl) return;
        const payload = new FormData(form);
        try {
            showIndicator('Guardando…');
            const res = await fetch(autosaveUrl, {
                method: 'POST',
                body: payload,
                credentials: 'same-origin',
                headers: { 'X-Requested-With': 'fetch' }
            });
            if (res.ok) {
                showIndicator('Borrador guardado');
            } else {
                showIndicator('No se pudo guardar');
            }
        } catch (_) {
            showIndicator('Sin conexión — guardaremos cuando vuelva');
        }
    }

    function scheduleAutosave() {
        clearTimeout(debounceTimer);
        debounceTimer = setTimeout(autosave, AUTOSAVE_DEBOUNCE_MS);
    }

    // Trigger autosave on any input/change in the form
    form.addEventListener('input', scheduleAutosave);
    form.addEventListener('change', scheduleAutosave);

    // Save on tab-away or page-hide (last-chance persistence)
    window.addEventListener('pagehide', () => {
        clearTimeout(debounceTimer);
        navigator.sendBeacon && autosaveUrl
            ? navigator.sendBeacon(autosaveUrl, new FormData(form))
            : autosave();
    });

    // Keyboard: Enter in input should not accidentally submit mid-wizard
    form.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && e.target.tagName === 'INPUT' && e.target.type !== 'submit') {
            e.preventDefault();
        }
    });
})();
