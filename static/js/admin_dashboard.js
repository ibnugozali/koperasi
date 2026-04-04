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
    
        // --- Tambahkan Script Chart.js untuk Aktivitas Terbaru ---
        if (typeof window.AktivitasData !== 'undefined' && document.getElementById('activityChart')) {
            // Jika data hanya berisi statistik utama (Anggota, Simpanan, Pinjaman)
            const isStatistikOnly = window.AktivitasData.length === 3 &&
                window.AktivitasData.some(x => x.Jenis === 'Anggota') &&
                window.AktivitasData.some(x => x.Jenis === 'Simpanan') &&
                window.AktivitasData.some(x => x.Jenis === 'Pinjaman');

            let labels = [];
            let datasets = [];
            if (isStatistikOnly) {
                labels = ['Statistik'];
                datasets = [
                    {
                        label: 'Anggota',
                        data: [window.AktivitasData.find(x => x.Jenis === 'Anggota').Jumlah],
                        borderColor: 'blue',
                        backgroundColor: 'rgba(0,0,255,0.65)',
                        borderWidth: 1
                    },
                    {
                        label: 'Simpanan',
                        data: [window.AktivitasData.find(x => x.Jenis === 'Simpanan').Jumlah],
                        borderColor: 'green',
                        backgroundColor: 'rgba(0,128,0,0.65)',
                        borderWidth: 1
                    },
                    {
                        label: 'Pinjaman',
                        data: [window.AktivitasData.find(x => x.Jenis === 'Pinjaman').Jumlah],
                        borderColor: 'red',
                        backgroundColor: 'rgba(255,0,0,0.65)',
                        borderWidth: 1
                    }
                ];
            } else {
                // Format data untuk Chart.js seperti sebelumnya
                labels = window.AktivitasData.map(item => {
                    const tgl = new Date(item.Tanggal);
                    return tgl.toLocaleDateString('id-ID');
                });
                datasets = [
                    {
                        label: 'Simpanan',
                        data: window.AktivitasData.filter(x => x.Jenis === 'Simpanan').map(x => x.Jumlah),
                        borderColor: 'green',
                        backgroundColor: 'rgba(0,128,0,0.65)',
                        borderWidth: 1
                    },
                    {
                        label: 'Pinjaman',
                        data: window.AktivitasData.filter(x => x.Jenis === 'Pinjaman').map(x => x.Jumlah),
                        borderColor: 'red',
                        backgroundColor: 'rgba(255,0,0,0.65)',
                        borderWidth: 1
                    },
                    {
                        label: 'Anggota',
                        data: window.AktivitasData.filter(x => x.Jenis === 'Anggota').map(x => x.Jumlah),
                        borderColor: 'blue',
                        backgroundColor: 'rgba(0,0,255,0.65)',
                        borderWidth: 1
                    }
                ];
            }
            const ctx = document.getElementById('activityChart').getContext('2d');
            new Chart(ctx, {
                type: 'bar',
                data: {
                    labels: labels,
                    datasets: datasets
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    datasets: {
                        bar: {
                            borderRadius: 6,
                            barThickness: 16,
                            maxBarThickness: 18
                        }
                    },
                    plugins: {
                        legend: { position: 'top' },
                        title: { display: true, text: isStatistikOnly ? 'Statistik Koperasi' : 'Aktivitas 30 Hari Terakhir' }
                    }
                }
            });
        }
});
