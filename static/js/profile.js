// profile.js — Profile page interactions
// Bio character counter + username availability check

'use strict';

document.addEventListener('DOMContentLoaded', function () {

    // ── Bio character counter ──
    var bio = document.getElementById('bio');
    var bioCount = document.getElementById('bioCount');

    if (bio && bioCount) {
        bio.addEventListener('input', function () {
            var len = Array.from(bio.value).length;
            bioCount.textContent = len;

            if (len > 140) {
                bioCount.style.color = 'var(--error)';
            } else {
                bioCount.style.color = '';
            }
        });
    }

    // ── Username validation feedback ──
    var username = document.getElementById('username');

    if (username) {
        username.addEventListener('input', function () {
            var val = username.value.toLowerCase();
            username.value = val; // force lowercase as they type

            // Remove invalid characters visually
            var valid = val.replace(/[^a-z0-9_]/g, '');
            if (valid !== val) {
                username.value = valid;
            }
        });
    }

});
