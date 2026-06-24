// nav.js — Mobile hamburger + desktop avatar dropdown
// Pure vanilla JS. No dependencies.

document.addEventListener('DOMContentLoaded', function() {

    // ── Mobile hamburger ──
    var toggle = document.getElementById('navToggle');
    var menu = document.getElementById('navMenu');
    var close = document.getElementById('navClose');

    if (toggle && menu) {
        toggle.addEventListener('click', function() {
            menu.classList.add('active');
            toggle.classList.add('hidden');
            toggle.setAttribute('aria-expanded', 'true');
            document.body.style.overflow = 'hidden';
        });

        function closeMenu() {
            menu.classList.remove('active');
            toggle.classList.remove('hidden');
            toggle.setAttribute('aria-expanded', 'false');
            document.body.style.overflow = '';
        }

        if (close) {
            close.addEventListener('click', closeMenu);
        }

        menu.addEventListener('click', function(e) {
            if (e.target === menu) closeMenu();
        });

        document.addEventListener('keydown', function(e) {
            if (e.key === 'Escape' && menu.classList.contains('active')) closeMenu();
        });
    }

    // ── Desktop avatar dropdown ──
    var avatarBtn = document.getElementById('userMenuToggle');
    var dropdown = document.getElementById('userDropdown');

    if (avatarBtn && dropdown) {
        avatarBtn.addEventListener('click', function(e) {
            e.stopPropagation();
            var isOpen = dropdown.classList.contains('active');
            dropdown.classList.toggle('active');
            avatarBtn.setAttribute('aria-expanded', isOpen ? 'false' : 'true');
        });

        // Close on click outside
        document.addEventListener('click', function(e) {
            if (!dropdown.contains(e.target) && e.target !== avatarBtn) {
                dropdown.classList.remove('active');
                avatarBtn.setAttribute('aria-expanded', 'false');
            }
        });

        // Close on Escape
        document.addEventListener('keydown', function(e) {
            if (e.key === 'Escape' && dropdown.classList.contains('active')) {
                dropdown.classList.remove('active');
                avatarBtn.setAttribute('aria-expanded', 'false');
            }
        });
    }
});
