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

    // Handle timeline functionality for sejarah page
    const addTimelineBtn = document.getElementById('add-timeline-item');
    const timelineContainer = document.getElementById('timeline-container');

    if (addTimelineBtn && timelineContainer) {
        addTimelineBtn.addEventListener('click', function() {
            const newItem = document.createElement('div');
            newItem.className = 'timeline-item-editor border p-3 mb-3 rounded';
            newItem.innerHTML = `
                <div class="row">
                    <div class="col-md-6 mb-2">
                        <label class="form-label">Judul Timeline</label>
                        <input type="text" class="form-control timeline-title" placeholder="Masukkan judul timeline">
                    </div>
                    <div class="col-md-3 mb-2 d-flex align-items-end">
                        <button type="button" class="btn btn-danger btn-sm remove-timeline-item">Hapus</button>
                    </div>
                </div>
                <div class="mb-2">
                    <label class="form-label">Deskripsi</label>
                    <textarea class="form-control timeline-text" rows="3" placeholder="Masukkan deskripsi timeline"></textarea>
                </div>
            `;
            timelineContainer.appendChild(newItem);
        });

        // Handle remove timeline item
        timelineContainer.addEventListener('click', function(e) {
            if (e.target.classList.contains('remove-timeline-item')) {
                e.target.closest('.timeline-item-editor').remove();
            }
        });
    }

    // Handle form submission with file uploads
    if (editForm) {
        editForm.addEventListener('submit', function(event) {
            event.preventDefault();

            // Collect text content based on slug
            const slug = 'sejarah';
            const konten = {};

            // Collect timeline data
            const timelineItems = [];
            const timelineEditors = document.querySelectorAll('.timeline-item-editor');
            timelineEditors.forEach(editor => {
                const title = editor.querySelector('.timeline-title').value.trim();
                const text = editor.querySelector('.timeline-text').value.trim();
                timelineItems.push({
                    title: title,
                    text: text
                });
            });
            konten.timeline = timelineItems;
            konten.komitmen_title = document.getElementById('komitmen_title_sejarah').value;
            konten.komitmen_text = document.getElementById('komitmen_text_sejarah').value;

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
                    // Reload page to show updated data and ensure persistence
                    window.location.reload();
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
});
