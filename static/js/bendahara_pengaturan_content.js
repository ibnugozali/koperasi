// JavaScript for bendahara_pengaturan_content.html

document.addEventListener('DOMContentLoaded', function() {
    // Handle profile form submission
    const profileForm = document.getElementById('profileForm');
    if (profileForm) {
        profileForm.addEventListener('submit', function(e) {
            e.preventDefault();

            // Basic validation
            const nama = document.getElementById('namaBendahara').value.trim();
            const username = document.getElementById('usernameBendahara').value.trim();
            const email = document.getElementById('emailBendahara').value.trim();
            const telepon = document.getElementById('teleponBendahara').value.trim();

            if (!nama || !username || !email || !telepon) {
                showAlert('Semua field harus diisi!', 'danger');
                return;
            }

            // Email validation
            const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
            if (!emailRegex.test(email)) {
                showAlert('Format email tidak valid!', 'danger');
                return;
            }

            // Phone validation (basic)
            const phoneRegex = /^[\d\s\-\+\(\)]+$/;
            if (!phoneRegex.test(telepon)) {
                showAlert('Format nomor telepon tidak valid!', 'danger');
                return;
            }

            // Here you would typically send an AJAX request to update the profile
            // For now, we'll just show a success message
            showAlert('Profil berhasil diperbarui!', 'success');

            // Reset form or update display values if needed
        });
    }

    // Handle password form submission
    const passwordForm = document.getElementById('passwordForm');
    if (passwordForm) {
        passwordForm.addEventListener('submit', function(e) {
            e.preventDefault();

            const currentPassword = document.getElementById('currentPassword').value;
            const newPassword = document.getElementById('newPassword').value;
            const confirmPassword = document.getElementById('confirmPassword').value;

            // Clear previous alerts
            clearAlerts();

            if (!currentPassword || !newPassword || !confirmPassword) {
                showAlert('Semua field password harus diisi!', 'danger');
                return;
            }

            if (newPassword !== confirmPassword) {
                showAlert('Password baru dan konfirmasi tidak cocok!', 'danger');
                return;
            }

            if (newPassword.length < 8) {
                showAlert('Password minimal 8 karakter!', 'danger');
                return;
            }

            // Check if new password is different from current
            if (currentPassword === newPassword) {
                showAlert('Password baru harus berbeda dari password lama!', 'danger');
                return;
            }

            // Here you would typically send an AJAX request to change the password
            // For now, we'll just show a success message
            showAlert('Password berhasil diubah!', 'success');
            this.reset();
        });
    }

    // Function to show alerts
    function showAlert(message, type) {
        clearAlerts();

        const alertDiv = document.createElement('div');
        alertDiv.className = `alert alert-${type} alert-dismissible fade show`;
        alertDiv.innerHTML = `
            ${message}
            <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
        `;

        // Insert alert after the form
        const form = type === 'success' ? profileForm : passwordForm;
        if (form) {
            form.parentNode.insertBefore(alertDiv, form.nextSibling);
        }

        // Auto-dismiss after 5 seconds
        setTimeout(() => {
            if (alertDiv.parentNode) {
                alertDiv.remove();
            }
        }, 5000);
    }

    // Function to clear existing alerts
    function clearAlerts() {
        const alerts = document.querySelectorAll('.alert');
        alerts.forEach(alert => alert.remove());
    }

    // Add input validation feedback
    const inputs = document.querySelectorAll('input');
    inputs.forEach(input => {
        input.addEventListener('blur', function() {
            if (this.value.trim() === '' && this.hasAttribute('required')) {
                this.classList.add('is-invalid');
            } else {
                this.classList.remove('is-invalid');
            }
        });

        input.addEventListener('input', function() {
            if (this.classList.contains('is-invalid') && this.value.trim() !== '') {
                this.classList.remove('is-invalid');
            }
        });
    });
});
