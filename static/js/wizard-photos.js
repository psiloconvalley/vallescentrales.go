/* ============================================================================
   Wizard Photos — Drag-and-drop, AJAX upload to R2, preview, reorder
   Vanilla ES6+, zero dependencies.
   ============================================================================ */

(function () {
    'use strict';

    const MAX_FILES = 10;
    const MAX_SIZE = 10 * 1024 * 1024;
    const ALLOWED = ['image/jpeg', 'image/png', 'image/webp'];
    const UPLOAD_URL = '/publicar/foto';

    const dropzone = document.querySelector('[data-dropzone]');
    const fileInput = document.querySelector('#images');
    const previewGrid = document.querySelector('[data-preview-grid]');
    const photosContainer = document.querySelector('[data-photos-container]');
    const submitBtn = document.querySelector('[data-wizard-form] button[type="submit"]');

    if (!dropzone || !fileInput || !previewGrid) return;

    let files = [];
    const csrfToken = (document.querySelector('meta[name="csrf-token"]') || {}).content || '';

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
    fileInput.addEventListener('change', () => {
        addFiles(fileInput.files);
        fileInput.value = '';
    });

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
            const entry = { file: f, url: null, storageKey: null, status: 'uploading' };
            files.push(entry);
            uploadFile(entry);
        }
        renderPreviews();
        updateSubmitState();
    }

    async function uploadFile(entry) {
        const fd = new FormData();
        fd.append('photo', entry.file);
        try {
            const res = await fetch(UPLOAD_URL, {
                method: 'POST',
                headers: { 'X-CSRF-Token': csrfToken },
                body: fd,
            });
            if (!res.ok) throw new Error('HTTP ' + res.status);
            const data = await res.json();
            entry.url = data.url;
            entry.storageKey = data.storage_key;
            entry.status = 'done';
        } catch (err) {
            console.error('Photo upload error:', err);
            entry.status = 'error';
        }
        renderPreviews();
        syncHiddenInputs();
        updateSubmitState();
    }

    function removeFile(index) {
        files.splice(index, 1);
        renderPreviews();
        syncHiddenInputs();
        updateSubmitState();
    }

    function setPrimary(index) {
        const f = files.splice(index, 1)[0];
        files.unshift(f);
        renderPreviews();
        syncHiddenInputs();
    }

    function moveFile(from, to) {
        if (to < 0 || to >= files.length) return;
        const f = files.splice(from, 1)[0];
        files.splice(to, 0, f);
        renderPreviews();
        syncHiddenInputs();
    }

    function syncHiddenInputs() {
        if (!photosContainer) return;
        photosContainer.innerHTML = '';
        files.forEach(function (entry) {
            if (entry.url) {
                var input = document.createElement('input');
                input.type = 'hidden';
                input.name = 'photos';
                input.value = entry.url;
                photosContainer.appendChild(input);
            }
        });
    }

    function updateSubmitState() {
        if (!submitBtn) return;
        var uploading = files.some(function (f) { return f.status === 'uploading'; });
        submitBtn.disabled = uploading;
        submitBtn.classList.toggle('is-busy', uploading);
    }

    function renderPreviews() {
        previewGrid.innerHTML = '';
        files.forEach(function (entry, i) {
            var card = document.createElement('div');
            card.className = 'wizard-preview-card' + (i === 0 ? ' is-primary' : '');
            if (entry.status === 'uploading') card.classList.add('is-uploading');
            if (entry.status === 'error') card.classList.add('is-error');
            card.draggable = entry.status === 'done';
            card.dataset.index = i;

            var img = document.createElement('img');
            img.className = 'wizard-preview-img';
            img.alt = entry.file.name;
            img.src = URL.createObjectURL(entry.file);
            img.onload = function () { URL.revokeObjectURL(img.src); };

            var badge = document.createElement('span');
            badge.className = 'wizard-preview-badge';
            if (entry.status === 'uploading') {
                badge.textContent = '⏳ Subiendo…';
            } else if (entry.status === 'error') {
                badge.textContent = '✕ Error';
            } else {
                badge.textContent = i === 0 ? '★ Principal' : String(i + 1);
            }

            var actions = document.createElement('div');
            actions.className = 'wizard-preview-actions';

            if (entry.status === 'done') {
                if (i > 0) {
                    var starBtn = document.createElement('button');
                    starBtn.type = 'button';
                    starBtn.className = 'wizard-preview-btn';
                    starBtn.textContent = '★';
                    starBtn.title = 'Hacer foto principal';
                    starBtn.addEventListener('click', function (e) { e.stopPropagation(); setPrimary(i); });
                    actions.appendChild(starBtn);
                }
                if (i > 0) {
                    var leftBtn = document.createElement('button');
                    leftBtn.type = 'button';
                    leftBtn.className = 'wizard-preview-btn';
                    leftBtn.textContent = '←';
                    leftBtn.title = 'Mover a la izquierda';
                    leftBtn.addEventListener('click', function (e) { e.stopPropagation(); moveFile(i, i - 1); });
                    actions.appendChild(leftBtn);
                }
                if (i < files.length - 1) {
                    var rightBtn = document.createElement('button');
                    rightBtn.type = 'button';
                    rightBtn.className = 'wizard-preview-btn';
                    rightBtn.textContent = '→';
                    rightBtn.title = 'Mover a la derecha';
                    rightBtn.addEventListener('click', function (e) { e.stopPropagation(); moveFile(i, i + 1); });
                    actions.appendChild(rightBtn);
                }
            }

            var removeBtn = document.createElement('button');
            removeBtn.type = 'button';
            removeBtn.className = 'wizard-preview-btn wizard-preview-btn-remove';
            removeBtn.textContent = '✕';
            removeBtn.title = 'Eliminar foto';
            removeBtn.addEventListener('click', function (e) { e.stopPropagation(); removeFile(i); });
            actions.appendChild(removeBtn);

            card.appendChild(img);
            card.appendChild(badge);
            card.appendChild(actions);

            if (entry.status === 'done') {
                card.addEventListener('dragstart', function (e) {
                    e.dataTransfer.setData('text/plain', String(i));
                    card.classList.add('is-dragging');
                });
                card.addEventListener('dragend', function () { card.classList.remove('is-dragging'); });
                card.addEventListener('dragover', function (e) { e.preventDefault(); });
                card.addEventListener('drop', function (e) {
                    e.preventDefault();
                    e.stopPropagation();
                    var from = parseInt(e.dataTransfer.getData('text/plain'));
                    moveFile(from, i);
                });
            }

            previewGrid.appendChild(card);
        });
    }
})();
