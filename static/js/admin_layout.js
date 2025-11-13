function showSection(section) {
    alert('Tampilkan bagian: ' + section);
}

function logout() {
    if (confirm('Yakin ingin keluar?')) {
        window.location.href = '/logout';
    }
}

function toggleSidebar() {
    const sidebar = document.getElementById('sidebar');
    const mainContent = document.querySelector('.main-content');
    const navbar = document.querySelector('.navbar');
    const footer = document.querySelector('footer');
    sidebar.classList.toggle('show');
    if (window.innerWidth <= 768) {
        if (sidebar.classList.contains('show')) {
            mainContent.style.marginLeft = '240px';
            if (navbar) navbar.style.marginLeft = '240px';
            footer.style.marginLeft = '240px';
        } else {
            mainContent.style.marginLeft = '0';
            if (navbar) navbar.style.marginLeft = '0';
            footer.style.marginLeft = '0';
        }
    }
}

// Initialize hamburger menu functionality and page animations
document.addEventListener('DOMContentLoaded', function() {
    // Fade in body and sidebar after load
    document.body.classList.add('loaded');
    const sidebar = document.getElementById('sidebar');
    if (sidebar) {
        sidebar.classList.add('loaded');
    }

    const hamburger = document.querySelector('.hamburger');
    if (hamburger) {
        hamburger.addEventListener('click', toggleSidebar);
    }

    // Add fade-in animation to main content
    const mainContent = document.querySelector('.main-content');
    if (mainContent) {
        mainContent.classList.add('fade-in');
    }

    // Add click animations for navigation links
    const navLinks = document.querySelectorAll('.sidebar a:not(.dropdown-toggle)');
    navLinks.forEach(link => {
        link.addEventListener('click', function() {
            // Add loading state to main content
            const mainContent = document.querySelector('.main-content');
            if (mainContent) {
                mainContent.classList.add('loading');
                // Remove loading state after animation
                setTimeout(() => {
                    mainContent.classList.remove('loading');
                }, 400);
            }
        });
    });
});

// Close sidebar when clicking outside on mobile
document.addEventListener('click', function(event) {
    const sidebar = document.getElementById('sidebar');
    const hamburger = document.querySelector('.hamburger');
    if (window.innerWidth <= 768 && !sidebar.contains(event.target) && !hamburger.contains(event.target)) {
        sidebar.classList.remove('show');
        // Reset margins when closing
        const mainContent = document.querySelector('.main-content');
        const navbar = document.querySelector('.navbar');
        const footer = document.querySelector('footer');
        mainContent.style.marginLeft = '0';
        if (navbar) navbar.style.marginLeft = '0';
        footer.style.marginLeft = '0';
    }
});
