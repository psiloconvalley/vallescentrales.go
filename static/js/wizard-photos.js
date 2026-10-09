/* ============================================================================
   Wizard Photos — Drag-and-drop, preview, reorder, primary selection
   Vanilla ES6+, zero dependencies.
   ============================================================================ */

(function () {
    'use strict';

    const MAX_FILES = 10;
    const MAX_SIZE = 10 * 1024 * 1024;
    const ALLOWED = ['image/jpeg', 'image/png', 'image/webp'];

    const dropzone = document.querySelector('[data-dropzone]');
    const fileInput = document.querySelector('#images');
    const previewGrid = document.querySelector('[data-preview-grid]');
    if (!dropzone || !fileInput || !previewGrid) return;

    let files = [];

    // --- Drag & Drop ---
    dropzone.addEventListener('click', () => fileInput.click());
    dropzone.addEventListener('dragover', (e) => {
        e.preventDefault();
        dropzone.classList.add('is-dragging');
    });
    dropzone.addEventListener('dragleave', () => dropzone.classList.remove('is-dragging'));
    dropzone.addEventListener('drop', (e) => {
        e.preventDefault();
        dropzone.classList.remove('is-dragging');
        addFiles(e.dataTransfer.files);
    });
    fileInput.addEventListener('change', () => addFiles(fileInput.files));

    function addFiles(newFiles) {
        for (const f of newFiles) {
            if (files.length >= MAX_FILES) {
                alert('Máximo ' + MAX_FILES + ' imágenes.');
                break;
            }
            if (!ALLOWED.includes(f.type)) {
                alert('"' + f.name + '" no es un formato válido. Usa JPG, PNG o WebP.');
                continue;
            }
            if (f.size > MAX_SIZE) {
                alert('"' + f.name + '" excede 10 MB.');
                continue;
            }
            files.push(f);
        }
        renderPreviews();
        syncFileInput();
    }

    function removeFile(index) {
        files.splice(index, 1);
        renderPreviews();
        syncFileInput();
    }

    function setPrimary(index) {
        const f = files.splice(index, 1)[0];
        files.unshift(f);
        renderPreviews();
        syncFileInput();
    }

    function moveFile(from, to) {
        if (to < 0 || to >= files.length) return;
        const f = files.splice(from, 1)[0];
        files.splice(to, 0, f);
        renderPreviews();
        syncFileInput();
    }

    // Keep the hidden file input in sync with our array
    function syncFileInput() {
        const dt = new DataTransfer();
        files.forEach(f => dt.items.add(f));
        fileInput.files = dt.files;
    }

    function renderPreviews() {
        previewGrid.innerHTML = '';
        files.forEach((file, i) => {
            const card = document.createElement('div');
            card.className = 'wizard-preview-card' + (i === 0 ? ' is-primary' : '');
            card.draggable = true;
            card.dataset.index = i;

            const img = document.createElement('img');
            img.className = 'wizard-preview-img';
            img.alt = file.name;
            img.src = URL.createObjectURL(file);
            img.onload = () => URL.revokeObjectURL(img.src);

            const badge = document.createElement('span');
            badge.className = 'wizard-preview-badge';
            badge.textContent = i === 0 ? '★ Principal' : (i + 1);

            const actions = document.createElement('div');
            actions.className = 'wizard-preview-actions';

            if (i > 0) {
                const starBtn = document.createElement('button');
                starBtn.type = 'button';
                starBtn.className = 'wizard-preview-btn';
                starBtn.textContent = '★';
                starBtn.title = 'Hacer foto principal';
                starBtn.addEventListener('click', (e) => { e.stopPropagation(); setPrimary(i); });
                actions.appendChild(starBtn);
            }

            if (i > 0) {
                const leftBtn = document.createElement('button');
                leftBtn.type = 'button';
                leftBtn.className = 'wizard-preview-btn';
                leftBtn.textContent = '←';
                leftBtn.title = 'Mover a la izquierda';
                leftBtn.addEventListener('click', (e) => { e.stopPropagation(); moveFile(i, i - 1); });
                actions.appendChild(leftBtn);
            }

            if (i < files.length - 1) {
                const rightBtn = document.createElement('button');
                rightBtn.type = 'button';
                rightBtn.className = 'wizard-preview-btn';
                rightBtn.textContent = '→';
                rightBtn.title = 'Mover a la derecha';
                rightBtn.addEventListener('click', (e) => { e.stopPropagation(); moveFile(i, i + 1); });
                actions.appendChild(rightBtn);
            }

            const removeBtn = document.createElement('button');
            removeBtn.type = 'button';
            removeBtn.className = 'wizard-preview-btn wizard-preview-btn-remove';
            removeBtn.textContent = '✕';
            removeBtn.title = 'Eliminar foto';
            removeBtn.addEventListener('click', (e) => { e.stopPropagation(); removeFile(i); });
            actions.appendChild(removeBtn);

            card.appendChild(img);
            card.appendChild(badge);
            card.appendChild(actions);

            // Drag reorder
            card.addEventListener('dragstart', (e) => {
                e.dataTransfer.setData('text/plain', i);
                card.classList.add('is-dragging');
            });
            card.addEventListener('dragend', () => card.classList.remove('is-dragging'));
            card.addEventListener('dragover', (e) => e.preventDefault());
            card.addEventListener('drop', (e) => {
                e.preventDefault();
                e.stopPropagation();
                const from = parseInt(e.dataTransfer.getData('text/plain'));
                moveFile(from, i);
            });

            previewGrid.appendChild(card);
        });
    }
})();
