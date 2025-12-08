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

                    // Deteksi background dengan algoritma yang lebih baik
                    // Ambil sampel dari beberapa sudut untuk deteksi background yang lebih akurat
                    const corners = [
                        {x: 0, y: 0}, // kiri atas
                        {x: canvas.width - 1, y: 0}, // kanan atas
                        {x: 0, y: canvas.height - 1}, // kiri bawah
                        {x: canvas.width - 1, y: canvas.height - 1} // kanan bawah
                    ];

                    let bgColors = [];
                    corners.forEach(corner => {
                        const index = (corner.y * canvas.width + corner.x) * 4;
                        bgColors.push({
                            r: data[index],
                            g: data[index + 1],
                            b: data[index + 2]
                        });
                    });

                    // Hitung rata-rata warna background
                    const avgBgColor = {
                        r: Math.round(bgColors.reduce((sum, c) => sum + c.r, 0) / bgColors.length),
                        g: Math.round(bgColors.reduce((sum, c) => sum + c.g, 0) / bgColors.length),
                        b: Math.round(bgColors.reduce((sum, c) => sum + c.b, 0) / bgColors.length)
                    };

                    // Tolerance untuk deteksi background (lebih ketat)
                    const tolerance = 20;

                    // Remove background dengan algoritma yang lebih baik
                    for (let i = 0; i < data.length; i += 4) {
                        const r = data[i];
                        const g = data[i + 1];
                        const b = data[i + 2];

                        // Hitung jarak Euclidean dari warna background
                        const distance = Math.sqrt(
                            Math.pow(r - avgBgColor.r, 2) +
                            Math.pow(g - avgBgColor.g, 2) +
                            Math.pow(b - avgBgColor.b, 2)
                        );

                        // Jika warna mirip dengan background, buat transparan
                        if (distance < tolerance) {
                            data[i + 3] = 0; // Alpha channel = 0 (transparan)
                        }
                    }

                    // Terapkan perubahan
                    ctx.putImageData(imageData, 0, 0);

                    // Set sebagai preview
                    previewLogo.src = canvas.toDataURL('image/png'); // Pastikan PNG untuk transparan
                    previewLogo.style.backgroundColor = 'transparent';

                    // Simpan data canvas untuk upload
                    previewLogo.dataset.canvasData = canvas.toDataURL('image/png');
                };
                img.src = e.target.result;
            };
            reader.readAsDataURL(file);
        } else {
            // Reset preview jika tidak ada file
            previewLogo.src = '/static/images/placeholder.png';
            previewLogo.style.backgroundColor = '#f8f9fa';
            delete previewLogo.dataset.canvasData;
        }
    });

    // Upload logo
    uploadForm.addEventListener('submit', function(event) {
        event.preventDefault();

        // Cek apakah ada data canvas (gambar yang sudah diproses transparan)
        const canvasData = previewLogo.dataset.canvasData;
        if (!canvasData) {
            alert('Silakan pilih file logo terlebih dahulu.');
            return;
        }

        // Tampilkan progress bar
        uploadProgress.style.display = 'block';
        uploadProgress.querySelector('.progress-bar').style.width = '0%';

        // Kirim data canvas sebagai JSON
        fetch('/admin/upload-logo', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                logoData: canvasData
            })
        })
        .then(response => {
            console.log('Response status:', response.status); // Debug log
            return response.json();
        })
        .then(data => {
            console.log('Response data:', data); // Debug log
            if (data.success) {
                // Update logo saat ini
                currentLogo.src = data.logoPath;
                previewLogo.src = data.logoPath;

                // Reset form
                logoFileInput.value = '';
                delete previewLogo.dataset.canvasData;
                uploadProgress.style.display = 'none';

                // Tampilkan pesan sukses
                alert('Logo berhasil diupload!');

                // Update logo di navbar jika ada
                updateNavbarLogo(data.logoPath);

                // Simpan logo ke localStorage untuk persistensi
                localStorage.setItem('currentLogo', data.logoPath);
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
