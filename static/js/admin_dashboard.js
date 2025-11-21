document.addEventListener("DOMContentLoaded", function() {
    const sidebar = document.querySelector('.sidebar');
    const hamburger = document.querySelector('.hamburger');
    const mainContent = document.querySelector('.main-content');
    const navbar = document.querySelector('.navbar');
    const footer = document.querySelector('footer');

    function toggleSidebar() {
        if (window.innerWidth <= 768) {
            // MOBILE
            sidebar.classList.toggle('show');
            footer.style.marginLeft = sidebar.classList.contains('show') ? '240px' : '0';
        } else {
            // DESKTOP
            if (sidebar.style.transform === 'translateX(-100%)') {
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

    if (hamburger) {
        hamburger.addEventListener("click", toggleSidebar);
    }

    // Close sidebar when clicking outside
    document.addEventListener("click", function(event) {
        if (window.innerWidth <= 768) {
            if (!sidebar.contains(event.target) && !hamburger.contains(event.target)) {
                sidebar.classList.remove('show');
                footer.style.marginLeft = '0';
            }
        }
    });
});
