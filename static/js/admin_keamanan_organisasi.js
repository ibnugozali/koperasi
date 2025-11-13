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

// Handle form submission
document.getElementById('securityForm').addEventListener('submit', function(e) {
    e.preventDefault();
    // Here you would typically send the data to the server
    alert('Pengaturan keamanan organisasi berhasil disimpan!');
});

function resetSettings() {
    if (confirm('Apakah Anda yakin ingin mereset pengaturan ke default?')) {
        document.getElementById('hierarchyAccess').value = 'departmental';
        document.getElementById('positionEncryption').value = 'basic';
        document.getElementById('changeAuditPeriod').value = '180';
        document.getElementById('approvalRequired').value = 'all_changes';
        document.getElementById('enableStructureAudit').checked = true;
        document.getElementById('enableConfidentialPositions').checked = false;
        document.getElementById('enableEmergencyContacts').checked = true;
    }
}
