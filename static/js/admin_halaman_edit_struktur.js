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
        let isSubmitting = false; // Flag to prevent multiple submissions

        editForm.addEventListener('submit', async function(event) {
            event.preventDefault();

            if (isSubmitting) {
                return; // Prevent multiple submissions
            }

            isSubmitting = true;

            // Disable submit button
            const submitBtn = editForm.querySelector('button[type="submit"]');
            if (submitBtn) {
                submitBtn.disabled = true;
                submitBtn.innerHTML = '<i class="fas fa-spinner fa-spin me-2"></i>Menyimpan...';
            }

            // Collect text content based on slug
            const slug = 'struktur';
            const konten = {};

            konten.deskripsi = document.getElementById('deskripsi_struktur').value;

            // Handle image upload for struktur
            const gambarStrukturImg = document.getElementById('preview_gambar_struktur');
            let gambarPath = '';

            // Check if there's a new image to upload (data URL from file reader)
            if (gambarStrukturImg && gambarStrukturImg.src && gambarStrukturImg.src.startsWith('data:')) {
                // Convert data URL to blob and upload
                try {
                    const response = await fetch(gambarStrukturImg.src);
                    const blob = await response.blob();

                    const formData = new FormData();
                    formData.append('file', blob, 'struktur_' + Date.now() + '.png');

                    const uploadResponse = await fetch('/admin/upload/struktur', {
                        method: 'POST',
                        body: formData
                    });

                    const uploadData = await uploadResponse.json();
                    if (uploadData.success) {
                        gambarPath = uploadData.filePath;
                    } else {
                        throw new Error('Gagal upload gambar: ' + uploadData.message);
                    }
                } catch (uploadError) {
                    console.error('Upload error:', uploadError);
                    // Fallback to existing image or placeholder
                    if (gambarStrukturImg.src && !gambarStrukturImg.src.includes('placeholder')) {
                        gambarPath = gambarStrukturImg.src;
                    } else {
                        gambarPath = '/static/images/placeholder.png';
                    }
                }
            } else if (gambarStrukturImg && gambarStrukturImg.src && !gambarStrukturImg.src.includes('placeholder')) {
                // Keep existing image if it's not placeholder
                gambarPath = gambarStrukturImg.src;
            } else {
                // Use placeholder if no image
                gambarPath = '/static/images/placeholder.png';
            }

            konten.gambar_struktur = gambarPath;

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
                    window.location.reload();
                } else {
                    throw new Error(data.message || 'Gagal memperbarui halaman');
                }
            })
            .catch(error => {
                console.error('Error:', error);
                alert('Gagal memperbarui halaman: ' + error.message);
            })
            .finally(() => {
                // Re-enable submit button
                isSubmitting = false;
                if (submitBtn) {
                    submitBtn.disabled = false;
                    submitBtn.innerHTML = '<i class="fas fa-save me-2"></i>Simpan Perubahan';
                }
            });
        });
    }
});
