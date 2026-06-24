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


    // ── Avatar file input ──
    var avatarInput = document.getElementById("avatar");
    var avatarName = document.getElementById("avatarFileName");
    var avatarBtn = document.getElementById("avatarSubmitBtn");

    if (avatarInput && avatarName && avatarBtn) {
        avatarInput.addEventListener("change", function () {
            if (avatarInput.files.length > 0) {
                avatarName.textContent = avatarInput.files[0].name;
                avatarBtn.classList.remove("hidden");
            } else {
                avatarName.textContent = "";
                avatarBtn.classList.add("hidden");
            }
        });
    }

    // ── Listing photo file input ──
    var photoInput = document.getElementById("photos");
    var photoName = document.getElementById("listingPhotoFileName");
    var photoBtn = document.getElementById("listingPhotoSubmitBtn");

    if (photoInput && photoName && photoBtn) {
        photoInput.addEventListener("change", function () {
            if (photoInput.files.length > 0) {
                photoName.textContent = photoInput.files.length + " archivo(s) seleccionado(s)";
                photoBtn.classList.remove("hidden");
            } else {
                photoName.textContent = "";
                photoBtn.classList.add("hidden");
            }
        });
    }
});
