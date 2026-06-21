// auth.js — Password visibility toggle + strength indicator
// SVG icons with 2.5px stroke for premium weight.

document.addEventListener('DOMContentLoaded', function() {

    var eyeOpen = '<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>';

    var eyeClosed = '<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>';

    var toggle = document.getElementById('passwordToggle');
    var passwordInput = document.getElementById('password');

    if (toggle && passwordInput) {
        toggle.innerHTML = eyeOpen;

        toggle.addEventListener('click', function() {
            if (passwordInput.type === 'password') {
                passwordInput.type = 'text';
                toggle.innerHTML = eyeClosed;
                toggle.setAttribute('aria-label', 'Ocultar contraseña');
            } else {
                passwordInput.type = 'password';
                toggle.innerHTML = eyeOpen;
                toggle.setAttribute('aria-label', 'Mostrar contraseña');
            }
        });

        toggle.addEventListener('keydown', function(e) {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                toggle.click();
            }
        });
    }

    var strengthFill = document.getElementById('strengthFill');
    var strengthText = document.getElementById('strengthText');

    if (passwordInput && strengthFill && strengthText) {
        passwordInput.addEventListener('input', function() {
            var val = passwordInput.value;
            var score = 0;
            var label = '';
            var level = '';

            if (val.length === 0) {
                strengthFill.className = 'strength-fill';
                strengthText.className = 'strength-text';
                strengthText.textContent = '';
                return;
            }

            if (val.length >= 8)  score++;
            if (val.length >= 12) score++;
            if (/[a-z]/.test(val) && /[A-Z]/.test(val)) score++;
            if (/[0-9]/.test(val)) score++;
            if (/[^a-zA-Z0-9]/.test(val)) score++;

            if (score <= 1)      { level = 'weak';   label = 'Débil'; }
            else if (score <= 2) { level = 'fair';   label = 'Regular'; }
            else if (score <= 3) { level = 'good';   label = 'Buena'; }
            else                 { level = 'strong'; label = 'Fuerte'; }

            strengthFill.className = 'strength-fill ' + level;
            strengthText.className = 'strength-text ' + level;
            strengthText.textContent = label;
        });
    }
});
