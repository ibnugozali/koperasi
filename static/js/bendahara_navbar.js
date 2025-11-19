// Inisialisasi - pastikan submenu yang tidak aktif tertutup, yang aktif tetap terbuka
document.addEventListener('DOMContentLoaded', function() {
    // Load logo terbaru dari localStorage jika ada
    const savedLogo = localStorage.getItem('currentLogo');
    if (savedLogo) {
        const navbarLogo = document.getElementById('navbarLogo');
        if (navbarLogo) {
            navbarLogo.src = savedLogo;
        }
    }

    var elemenCollapse = document.querySelectorAll('.collapse');
    elemenCollapse.forEach(function(element) {
        if (!element.classList.contains('show')) {
            // Tutup submenu yang tidak aktif
            element.classList.remove('show');
            element.style.display = 'none';
            var linkToggle = element.previousElementSibling;
            if (linkToggle) {
                linkToggle.setAttribute('aria-expanded', 'false');
            }
        } else {
            // Biarkan submenu yang aktif terbuka
            var linkToggle = element.previousElementSibling;
            if (linkToggle) {
                linkToggle.setAttribute('aria-expanded', 'true');
            }
        }
    });

    // Mencegah submenu tertutup saat klik link di dalamnya
    var linkSubmenu = document.querySelectorAll('.collapse a');
    linkSubmenu.forEach(function(link) {
        link.addEventListener('click', function(event) {
            event.stopPropagation();
        });
    });

    // Pastikan submenu tetap terbuka berdasarkan URL saat ini dengan animasi
    var currentURL = window.location.pathname;
    if (currentURL.includes('/admin/konfirmasi') || currentURL.includes('/admin/anggota') || currentURL.includes('/admin/login-history')) {
        var manajemenAnggota = document.getElementById('manajemenAnggota');
        if (manajemenAnggota && !manajemenAnggota.classList.contains('show')) {
            manajemenAnggota.style.display = 'block';
            manajemenAnggota.style.maxHeight = '0px';
            manajemenAnggota.style.opacity = '0';
            manajemenAnggota.style.transition = 'max-height 0.3s ease-in-out, opacity 0.3s ease-in-out';
            setTimeout(function() {
                manajemenAnggota.classList.add('show');
                manajemenAnggota.style.maxHeight = manajemenAnggota.scrollHeight + 'px';
                manajemenAnggota.style.opacity = '1';
            }, 10);
            setTimeout(function() {
                manajemenAnggota.style.maxHeight = '';
                manajemenAnggota.style.transition = '';
            }, 300);
            var toggle = manajemenAnggota.previousElementSibling;
            if (toggle) {
                toggle.setAttribute('aria-expanded', 'true');
            }
        }
    }

    if (currentURL.includes('/admin/halaman/') || currentURL.includes('/admin/halaman')) {
        var kontenWebsite = document.getElementById('kontenWebsite');
        if (kontenWebsite && !kontenWebsite.classList.contains('show')) {
            kontenWebsite.style.display = 'block';
            kontenWebsite.style.maxHeight = '0px';
            kontenWebsite.style.opacity = '0';
            kontenWebsite.style.transition = 'max-height 0.3s ease-in-out, opacity 0.3s ease-in-out';
            setTimeout(function() {
                kontenWebsite.classList.add('show');
                kontenWebsite.style.maxHeight = kontenWebsite.scrollHeight + 'px';
                kontenWebsite.style.opacity = '1';
            }, 10);
            setTimeout(function() {
                kontenWebsite.style.maxHeight = '';
                kontenWebsite.style.transition = '';
            }, 300);
            var toggle = kontenWebsite.previousElementSibling;
            if (toggle) {
                toggle.setAttribute('aria-expanded', 'true');
            }
        }
    }
    if (currentURL.includes('/admin/riwayat') || currentURL.includes('/admin/laporan') || currentURL.includes('/admin/transaksi')) {
        var transaksiLaporan = document.getElementById('transaksiLaporan');
        if (transaksiLaporan && !transaksiLaporan.classList.contains('show')) {
            transaksiLaporan.style.display = 'block';
            transaksiLaporan.style.maxHeight = '0px';
            transaksiLaporan.style.opacity = '0';
            transaksiLaporan.style.transition = 'max-height 0.3s ease-in-out, opacity 0.3s ease-in-out';
            setTimeout(function() {
                transaksiLaporan.classList.add('show');
                transaksiLaporan.style.maxHeight = transaksiLaporan.scrollHeight + 'px';
                transaksiLaporan.style.opacity = '1';
            }, 10);
            setTimeout(function() {
                transaksiLaporan.style.maxHeight = '';
                transaksiLaporan.style.transition = '';
            }, 300);
            var toggle = transaksiLaporan.previousElementSibling;
            if (toggle) {
                toggle.setAttribute('aria-expanded', 'true');
            }
        }
    }
});

// Fungsi toggle submenu dengan animasi
function toggleSubmenu(toggle, id) {
    var elemenTarget = document.getElementById(id);
    var elemenToggle = toggle;

    if (elemenTarget) {
        // Toggle submenu yang diklik - klik pertama buka, klik kedua tutup
        if (elemenTarget.classList.contains('show')) {
            // Jika submenu sudah terbuka, tutup dengan animasi
            elemenTarget.style.transition = 'max-height 0.3s ease-in-out, opacity 0.3s ease-in-out';
            elemenTarget.style.maxHeight = elemenTarget.scrollHeight + 'px';
            elemenTarget.style.opacity = '1';
            setTimeout(function() {
                elemenTarget.style.maxHeight = '0px';
                elemenTarget.style.opacity = '0';
            }, 10);
            setTimeout(function() {
                elemenTarget.classList.remove('show');
                elemenTarget.style.display = 'none';
                elemenTarget.style.maxHeight = '';
                elemenTarget.style.opacity = '';
                elemenTarget.style.transition = '';
                elemenToggle.setAttribute('aria-expanded', 'false');
            }, 300);
        } else {
            // Jika submenu tertutup, buka dan tutup semua submenu lain
            var semuaCollapse = document.querySelectorAll('.collapse');
            semuaCollapse.forEach(function(collapse) {
                if (collapse !== elemenTarget && collapse.classList.contains('show')) {
                    collapse.style.transition = 'max-height 0.3s ease-in-out, opacity 0.3s ease-in-out';
                    collapse.style.maxHeight = collapse.scrollHeight + 'px';
                    collapse.style.opacity = '1';
                    setTimeout(function() {
                        collapse.style.maxHeight = '0px';
                        collapse.style.opacity = '0';
                    }, 10);
                    setTimeout(function() {
                        collapse.classList.remove('show');
                        collapse.style.display = 'none';
                        collapse.style.maxHeight = '';
                        collapse.style.opacity = '';
                        collapse.style.transition = '';
                        var toggleLain = collapse.previousElementSibling;
                        if (toggleLain) {
                            toggleLain.setAttribute('aria-expanded', 'false');
                        }
                    }, 300);
                }
            });

            // Buka submenu yang diklik dengan animasi
            elemenTarget.style.display = 'block';
            elemenTarget.style.maxHeight = '0px';
            elemenTarget.style.opacity = '0';
            elemenTarget.style.transition = 'max-height 0.3s ease-in-out, opacity 0.3s ease-in-out';
            setTimeout(function() {
                elemenTarget.classList.add('show');
                elemenTarget.style.maxHeight = elemenTarget.scrollHeight + 'px';
                elemenTarget.style.opacity = '1';
                elemenToggle.setAttribute('aria-expanded', 'true');
            }, 10);
            setTimeout(function() {
                elemenTarget.style.maxHeight = '';
                elemenTarget.style.transition = '';
            }, 300);
        }
    }
}
