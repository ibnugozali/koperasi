// admin_logo.js - JavaScript untuk halaman edit logo admin

document.addEventListener('DOMContentLoaded', function() {
    const logoFileInput = document.getElementById('logoFile');
    const previewLogo = document.getElementById('previewLogo');
    const currentLogo = document.getElementById('currentLogo');
    const uploadForm = document.getElementById('uploadLogoForm');
    const resetBtn = document.getElementById('resetBtn');
    const uploadProgress = document.getElementById('uploadProgress');

    // Preview logo saat file dipilih
    logoFileInput.addEventListener('change', function(event) {
        const file = event.target.files[0];
        if (file) {
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

            // Preview gambar dengan background transparan
            const reader = new FileReader();
            reader.onload = function(e) {
                // Buat canvas untuk remove background
                const img = new Image();
                img.onload = function() {
                    const canvas = document.createElement('canvas');
                    const ctx = canvas.getContext('2d');
                    canvas.width = img.width;
                    canvas.height = img.height;

                    // Gambar gambar asli
                    ctx.drawImage(img, 0, 0);

                    // Ambil data gambar
                    const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
                    const data = imageData.data;

                    // Deteksi background (ambil warna dari sudut kiri atas sebagai background)
                    const bgColor = {
                        r: data[0],
                        g: data[1],
                        b: data[2]
                    };

                    // Tolerance untuk deteksi background
                    const tolerance = 30;

                    // Remove background
                    for (let i = 0; i < data.length; i += 4) {
                        const r = data[i];
                        const g = data[i + 1];
                        const b = data[i + 2];

                        // Jika warna mirip dengan background, buat transparan
                        if (Math.abs(r - bgColor.r) < tolerance &&
                            Math.abs(g - bgColor.g) < tolerance &&
                            Math.abs(b - bgColor.b) < tolerance) {
                            data[i + 3] = 0; // Alpha channel = 0 (transparan)
                        }
                    }

                    // Terapkan perubahan
                    ctx.putImageData(imageData, 0, 0);

                    // Set sebagai preview
                    previewLogo.src = canvas.toDataURL();
                    previewLogo.style.backgroundColor = 'transparent';
                };
                img.src = e.target.result;
            };
            reader.readAsDataURL(file);
        } else {
            // Reset preview jika tidak ada file
            previewLogo.src = '/static/images/placeholder.png';
            previewLogo.style.backgroundColor = '#f8f9fa';
        }
    });

    // Upload logo
    uploadForm.addEventListener('submit', function(event) {
        event.preventDefault();

        const formData = new FormData(this);

        // Tampilkan progress bar
        uploadProgress.style.display = 'block';
        uploadProgress.querySelector('.progress-bar').style.width = '0%';

        fetch('/admin/upload-logo', {
            method: 'POST',
            body: formData
        })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                // Update logo saat ini
                currentLogo.src = data.logoPath;
                previewLogo.src = data.logoPath;

                // Reset form
                logoFileInput.value = '';
                uploadProgress.style.display = 'none';

                // Tampilkan pesan sukses
                alert('Logo berhasil diupload!');

                // Update logo di navbar jika ada
                updateNavbarLogo(data.logoPath);
            } else {
                throw new Error(data.message || 'Gagal upload logo');
            }
        })
        .catch(error => {
            console.error('Error:', error);
            alert('Gagal upload logo: ' + error.message);
            uploadProgress.style.display = 'none';
        });
    });

    // Reset preview
    resetBtn.addEventListener('click', function() {
        logoFileInput.value = '';
        previewLogo.src = currentLogo.src;
    });

    // Fungsi untuk update logo di navbar
    function updateNavbarLogo(logoPath) {
        const navbarLogos = document.querySelectorAll('.sidebar img[alt="Logo Koperasi"]');
        navbarLogos.forEach(img => {
            img.src = logoPath;
        });
    }
});
