document.addEventListener('DOMContentLoaded', function() {
    // Initialize login history page functionality
    initializeLoginHistoryPage();
});

function initializeLoginHistoryPage() {
    // Add event listeners to delete buttons
    const deleteButtons = document.querySelectorAll('.btn-delete');
    deleteButtons.forEach(button => {
        button.addEventListener('click', handleDeleteLoginHistory);
    });

    // Add loading state management
    setupLoadingStates();
}

function handleDeleteLoginHistory(event) {
    const button = event.currentTarget;
    const id = button.getAttribute('data-id');

    // Show confirmation dialog
    if (!confirm('Apakah Anda yakin ingin menghapus riwayat login ini?')) {
        return;
    }

    // Show loading state
    setButtonLoading(button, true);

    // Perform delete request
    fetch('/admin/login-history/' + id, {
        method: 'DELETE',
        headers: {
            'Content-Type': 'application/json',
        },
    })
    .then(response => {
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        return response.json();
    })
    .then(data => {
        if (data.message) {
            showAlert('success', 'Riwayat login berhasil dihapus');
            // Remove the row from the table
            const row = button.closest('tr');
            row.remove();
            // Check if table is empty
            checkEmptyState();
        } else {
            throw new Error(data.error || 'Unknown error');
        }
    })
    .catch(error => {
        console.error('Error:', error);
        showAlert('error', 'Terjadi kesalahan saat menghapus riwayat login: ' + error.message);
    })
    .finally(() => {
        // Hide loading state
        setButtonLoading(button, false);
    });
}

function setButtonLoading(button, isLoading) {
    if (isLoading) {
        button.disabled = true;
        button.innerHTML = '<i class="fa-solid fa-spinner fa-spin me-1"></i>Menghapus...';
    } else {
        button.disabled = false;
        button.innerHTML = '<i class="fa-solid fa-trash me-1"></i>Hapus';
    }
}

function showAlert(type, message) {
    // Create alert element
    const alertDiv = document.createElement('div');
    alertDiv.className = `alert alert-${type === 'success' ? 'success' : 'danger'} alert-dismissible fade show position-fixed`;
    alertDiv.style.cssText = 'top: 20px; right: 20px; z-index: 9999; min-width: 300px;';
    alertDiv.innerHTML = `
        ${message}
        <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
    `;

    // Add to page
    document.body.appendChild(alertDiv);

    // Auto remove after 5 seconds
    setTimeout(() => {
        if (alertDiv.parentNode) {
            alertDiv.remove();
        }
    }, 5000);
}

function checkEmptyState() {
    const tableBody = document.querySelector('.login-history-table tbody');
    if (tableBody && tableBody.children.length === 0) {
        // Table is empty, reload page to show empty state
        location.reload();
    }
}

function setupLoadingStates() {
    // Add loading indicators for any future enhancements
    // This function can be expanded for additional loading states
}
