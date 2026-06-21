// nav.js — mobile hamburger menu toggle
// Pure vanilla JS. No dependencies.

document.addEventListener('DOMContentLoaded', function() {
    var toggle = document.getElementById('navToggle');
    var menu = document.getElementById('navMenu');
    var close = document.getElementById('navClose');

    if (!toggle || !menu) return;

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
        if (e.target === menu) {
            closeMenu();
        }
    });

    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape' && menu.classList.contains('active')) {
            closeMenu();
        }
    });
});
