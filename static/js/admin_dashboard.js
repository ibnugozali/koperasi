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

// Data untuk grafik aktivitas
const ctx = document.getElementById('activityChart').getContext('2d');
const activityChart = new Chart(ctx, {
    type: 'line',
    data: {
        labels: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'],
        datasets: [{
            label: 'Simpanan',
            data: [1200000, 1500000, 1800000, 2200000, 2500000, 2800000, 3000000, 3200000, 3500000, 3800000, 4000000, 4200000],
            borderColor: 'rgb(25, 135, 84)',
            backgroundColor: 'rgba(25, 135, 84, 0.1)',
            tension: 0.4,
            fill: true
        }, {
            label: 'Pinjaman',
            data: [5000000, 4500000, 6000000, 5500000, 7000000, 6500000, 8000000, 7500000, 9000000, 8500000, 10000000, 9500000],
            borderColor: 'rgb(220, 53, 69)',
            backgroundColor: 'rgba(220, 53, 69, 0.1)',
            tension: 0.4,
            fill: true
        }]
    },
    options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
            legend: {
                position: 'top',
            },
            title: {
                display: true,
                text: 'Aktivitas Simpanan dan Pinjaman Bulanan'
            }
        },
        scales: {
            y: {
                beginAtZero: true,
                ticks: {
                    callback: function(value, index, values) {
                        return 'Rp ' + (value / 1000000).toFixed(0) + 'M';
                    }
                }
            },
            x: {
                ticks: {
                    maxTicksLimit: 12
                }
            }
        },
        interaction: {
            mode: 'index',
            intersect: false,
        },
        hover: {
            mode: 'index',
            intersect: false
        }
    }
});
