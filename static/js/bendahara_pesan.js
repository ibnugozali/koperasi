// bendahara_pesan.js - AJAX form submission + notifications
document.addEventListener('DOMContentLoaded', function() {
    const form = document.querySelector('form[action="/bendahara/pesan"]');
    const submitBtn = form?.querySelector('button[type="submit"]');
    const searchInput = document.getElementById('pesan-searchAnggota');
    const anggotaSelect = document.getElementById('pesan-anggota');
    
    if (!form || !submitBtn) return;

    // Enhanced search with debounce
    let searchTimeout;
    if (searchInput && anggotaSelect) {
        searchInput.addEventListener('input', function() {
            clearTimeout(searchTimeout);
            searchTimeout = setTimeout(() => performSearch(this.value), 300);
        });
    }

    function performSearch(filter) {
        const options = anggotaSelect.querySelectorAll('option:not([value=""])');
        let visibleCount = 0;
        let visibleOptions = [];

        options.forEach(option => {
            const text = option.text.toLowerCase();
            const match = text.includes(filter.toLowerCase());
            option.style.display = match ? '' : 'none';
            if (match) {
                visibleOptions.push(option);
                visibleCount++;
            }
        });

        // Auto-select if exactly one match
        if (visibleCount === 1) {
            anggotaSelect.value = visibleOptions[0].value;
        } else {
            anggotaSelect.value = '';
        }

        // Visual feedback
        if (filter && visibleCount === 0) {
            showToast('Anggota tidak ditemukan', 'warning');
        }
    }

    // AJAX form submission
    form.addEventListener('submit', function(e) {
        e.preventDefault();
        
        const formData = new FormData(form);
        const originalBtnText = submitBtn.innerHTML;
        
        // Loading state
        submitBtn.innerHTML = '<i class="fa fa-spinner fa-spin me-2"></i>Mengirim...';
        submitBtn.disabled = true;
        searchInput.disabled = true;
        anggotaSelect.disabled = true;

        fetch('/bendahara/pesan', {
            method: 'POST',
            body: formData
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                showToast(data.message || 'Pesan berhasil dikirim!', 'success');
                form.reset();
                // Tampilkan warning jika WA tidak terkirim
                if (data.wa_info) {
                    setTimeout(() => showToast('<i class="fa-brands fa-whatsapp me-1"></i> ' + data.wa_info, 'warning'), 600);
                }
                setTimeout(() => location.reload(), 2800);
            } else {
                showToast(data.message || 'Gagal mengirim pesan', 'danger');
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showToast('Terjadi kesalahan koneksi. Silakan coba lagi.', 'danger');
        })
        .finally(() => {
            // Reset button
            submitBtn.innerHTML = originalBtnText;
            submitBtn.disabled = false;
            searchInput.disabled = false;
            anggotaSelect.disabled = false;
        });
    });

    // Bootstrap Toast Notifications
    function showToast(message, type = 'info') {
        // Remove existing toast
        const existingToast = document.querySelector('.toast');
        if (existingToast) existingToast.remove();

        const toast = document.createElement('div');
        toast.className = `toast align-items-center text-white bg-${type === 'success' ? 'success' : type === 'danger' ? 'danger' : type === 'warning' ? 'warning' : 'primary'} border-0 position-fixed`;
        toast.setAttribute('role', 'alert');
        toast.style.top = '20px';
        toast.style.right = '20px';
        toast.style.zIndex = '9999';
        toast.innerHTML = `
            <div class="d-flex">
                <div class="toast-body">${message}</div>
                <button type="button" class="btn-close btn-close-white me-2 m-auto" data-bs-dismiss="toast"></button>
            </div>
        `;

        document.body.appendChild(toast);

        const bsToast = new bootstrap.Toast(toast);
        bsToast.show();

        // Auto remove after dismiss
        toast.addEventListener('hidden.bs.toast', () => toast.remove());
    }

    // Keyboard shortcuts
    document.addEventListener('keydown', function(e) {
        if (e.ctrlKey && e.key === 'Enter') {
            form.dispatchEvent(new Event('submit'));
        }
    });
});

