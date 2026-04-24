document.addEventListener('DOMContentLoaded', function() {
    const fileInput = document.getElementById('backgroundFile');
    const previewImg = document.getElementById('previewBackgroundImg');
    const currentImg = document.getElementById('currentBackgroundImg');
    const form = document.getElementById('uploadBackgroundForm');
    const progress = document.getElementById('uploadBackgroundProgress');

    fileInput.addEventListener('change', function(event) {
        const file = event.target.files[0];
        if (!file) {
            previewImg.src = '/static/images/placeholder.png';
            return;
        }

        const allowedTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/gif'];
        if (!allowedTypes.includes(file.type)) {
            alert('Format file tidak didukung. Gunakan JPG, PNG, atau GIF.');
            fileInput.value = '';
            return;
        }
        if (file.size > 5 * 1024 * 1024) {
            alert('Ukuran file terlalu besar. Maksimal 5MB.');
            fileInput.value = '';
            return;
        }

        const reader = new FileReader();
        reader.onload = function(e) {
            previewImg.src = e.target.result;
        };
        reader.readAsDataURL(file);
    });

    form.addEventListener('submit', async function(event) {
        event.preventDefault();
        const file = fileInput.files[0];
        if (!file) {
            alert('Silakan pilih file background terlebih dahulu.');
            return;
        }

        const formData = new FormData();
        formData.append('backgroundFile', file);

        progress.style.display = 'block';
        progress.querySelector('.progress-bar').style.width = '40%';

        try {
            const response = await fetch('/admin/upload-background', {
                method: 'POST',
                body: formData
            });
            const data = await response.json();

            if (!response.ok || !data.success) {
                throw new Error(data.message || 'Gagal upload background');
            }

            const imageUrl = data.backgroundPath + '?t=' + new Date().getTime();
            currentImg.src = imageUrl;
            previewImg.src = imageUrl;
            progress.querySelector('.progress-bar').style.width = '100%';
            fileInput.value = '';

            alert('Background berhasil diupload!');
        } catch (error) {
            alert('Gagal upload background: ' + error.message);
        } finally {
            setTimeout(function() {
                progress.style.display = 'none';
                progress.querySelector('.progress-bar').style.width = '0%';
            }, 500);
        }
    });
});
