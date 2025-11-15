function toggleSidebar() {
    const sidebar = document.querySelector('.sidebar');
    const mainContent = document.querySelector('.main-content');
    const navbar = document.querySelector('.navbar');
    const footer = document.querySelector('footer');

    if (window.innerWidth <= 768) {
        // Mobile behavior: toggle show class
        sidebar.classList.toggle('show');
        if (sidebar.classList.contains('show')) {
            footer.style.marginLeft = '240px';
        } else {
            footer.style.marginLeft = '0';
        }
    } else {
        // Desktop behavior: toggle visibility by adjusting margins
        if (sidebar.style.transform === 'translateX(-100%)' || sidebar.style.transform === '') {
            sidebar.style.transform = 'translateX(0)';
            mainContent.style.marginLeft = '240px';
            navbar.style.marginLeft = '240px';
            footer.style.marginLeft = '240px';
        } else {
            sidebar.style.transform = 'translateX(-100%)';
            mainContent.style.marginLeft = '0';
            navbar.style.marginLeft = '0';
            footer.style.marginLeft = '0';
        }
    }
}

// Close sidebar when clicking outside on mobile
document.addEventListener('click', function(event) {
    const sidebar = document.querySelector('.sidebar');
    const hamburger = document.querySelector('.hamburger');
    const footer = document.querySelector('footer');
    if (window.innerWidth <= 768 && !sidebar.contains(event.target) && !hamburger.contains(event.target)) {
        sidebar.classList.remove('show');
        footer.style.marginLeft = '0';
    }
});

// File upload functionality for halaman edit (similar to admin_logo.js)
document.addEventListener('DOMContentLoaded', function() {
    const editForm = document.getElementById('editForm');
    const fileInputs = document.querySelectorAll('.file-upload');

    // Handle file input changes for preview
    fileInputs.forEach(input => {
        input.addEventListener('change', function(event) {
            const file = event.target.files[0];
            const target = this.getAttribute('data-target');
            const previewImg = document.getElementById('preview_' + target);

            if (file && previewImg) {
                // Validasi tipe file
                const allowedTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/gif'];
                if (!allowedTypes.includes(file.type)) {
                    alert('Format file tidak didukung. Gunakan JPG, PNG, atau GIF.');
                    this.value = '';
                    return;
                }

                // Validasi ukuran file (2MB)
                if (file.size > 2 * 1024 * 1024) {
                    alert('Ukuran file terlalu besar. Maksimal 2MB.');
                    this.value = '';
                    return;
                }

                // Preview gambar
                const reader = new FileReader();
                reader.onload = function(e) {
                    previewImg.src = e.target.result;
                    previewImg.style.display = 'block';
                };
                reader.readAsDataURL(file);
            }
        });
    });

    // Handle form submission with file uploads
    if (editForm) {
        editForm.addEventListener('submit', function(event) {
            event.preventDefault();

            // Collect text content based on slug
            const slug = 'visi-misi';
            const konten = {};

            konten.visi = document.getElementById('visi_visi_misi').value;

            // Collect misi as array
            const misiInputs = document.querySelectorAll('.misi-input');
            konten.misi = [];
            misiInputs.forEach(input => {
                if (input.value.trim() !== '') {
                    konten.misi.push(input.value.trim());
                }
            });

            // Set hidden konten field
            document.getElementById('konten').value = JSON.stringify(konten);

            // Send data via AJAX like admin_logo.js
            const judul = document.getElementById('judul_halaman').value;

            fetch(`/admin/halaman/update/${slug}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    judul: judul,
                    konten: JSON.stringify(konten)
                })
            })
            .then(response => response.json())
            .then(data => {
                console.log('Response data:', data); // Debug log
                if (data.success) {
                    alert('Halaman berhasil diperbarui!');
                    // Optional: reload page or update UI
                    // window.location.reload();
                } else {
                    throw new Error(data.message || 'Gagal memperbarui halaman');
                }
            })
            .catch(error => {
                console.error('Error:', error);
                alert('Gagal memperbarui halaman: ' + error.message);
            });
        });
    }

    // Handle add misi button
    const addMisiBtn = document.getElementById('add-misi');
    const misiContainer = document.getElementById('misi-container');

    if (addMisiBtn && misiContainer) {
        addMisiBtn.addEventListener('click', function() {
            const misiItem = document.createElement('div');
            misiItem.className = 'misi-item mb-2';
            misiItem.innerHTML = `
                <div class="input-group">
                    <textarea class="form-control misi-input" name="misi[]" rows="2" placeholder="Masukkan poin misi"></textarea>
                    <button type="button" class="btn btn-outline-danger remove-misi">
                        <i class="fas fa-trash"></i>
                    </button>
                </div>
            `;
            misiContainer.appendChild(misiItem);
            updateRemoveButtons();
        });
    }

    // Handle remove misi buttons
    function updateRemoveButtons() {
        const misiItems = document.querySelectorAll('.misi-item');
        misiItems.forEach((item, index) => {
            const removeBtn = item.querySelector('.remove-misi');
            if (misiItems.length > 1) {
                removeBtn.style.display = 'block';
            } else {
                removeBtn.style.display = 'none';
            }
        });
    }

    // Event delegation for remove buttons
    if (misiContainer) {
        misiContainer.addEventListener('click', function(event) {
            if (event.target.classList.contains('remove-misi') || event.target.closest('.remove-misi')) {
                const misiItem = event.target.closest('.misi-item');
                if (misiItem && document.querySelectorAll('.misi-item').length > 1) {
                    misiItem.remove();
                    updateRemoveButtons();
                }
            }
        });
    }

    // Initialize remove buttons on page load
    updateRemoveButtons();
});
