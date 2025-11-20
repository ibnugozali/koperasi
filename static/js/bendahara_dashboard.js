// bendahara_dashboard.js - JavaScript untuk dashboard bendahara

document.addEventListener('DOMContentLoaded', function() {
    // Ambil data aktivitas dari script tag
    const aktivitasDataScript = document.getElementById('aktivitas-data');
    let aktivitasData = [];

    if (aktivitasDataScript) {
        try {
            aktivitasData = JSON.parse(aktivitasDataScript.textContent);
        } catch (e) {
            console.error('Error parsing aktivitas data:', e);
        }
    }

    // Fungsi untuk memformat tanggal
    function formatDate(dateString) {
        const date = new Date(dateString);
        return date.toLocaleDateString('id-ID', {
            day: '2-digit',
            month: 'short',
            year: 'numeric'
        });
    }

    // Fungsi untuk memformat angka sebagai mata uang
    function formatCurrency(amount) {
        return new Intl.NumberFormat('id-ID', {
            style: 'currency',
            currency: 'IDR',
            minimumFractionDigits: 0
        }).format(amount);
    }

    // Siapkan data untuk Chart.js
    const labels = [];
    const simpananData = [];
    const pinjamanData = [];
    const angsuranData = [];

    // Kelompokkan data berdasarkan tanggal
    const groupedData = {};

    aktivitasData.forEach(item => {
        const date = formatDate(item.Tanggal);
        if (!groupedData[date]) {
            groupedData[date] = { simpanan: 0, pinjaman: 0, angsuran: 0 };
        }

        if (item.Jenis === 'Simpanan') {
            groupedData[date].simpanan += item.Jumlah;
        } else if (item.Jenis === 'Pinjaman') {
            groupedData[date].pinjaman += item.Jumlah;
        } else if (item.Jenis === 'Angsuran') {
            groupedData[date].angsuran += item.Jumlah;
        }
    });

    // Urutkan tanggal
    const sortedDates = Object.keys(groupedData).sort((a, b) => new Date(a) - new Date(b));

    // Isi data untuk chart
    sortedDates.forEach(date => {
        labels.push(date);
        simpananData.push(groupedData[date].simpanan);
        pinjamanData.push(groupedData[date].pinjaman);
        angsuranData.push(groupedData[date].angsuran);
    });

    // Jika tidak ada data, tampilkan pesan
    if (labels.length === 0) {
        labels.push('Tidak ada data');
        simpananData.push(0);
        pinjamanData.push(0);
        angsuranData.push(0);
    }

    // Buat chart menggunakan Chart.js
    const ctx = document.getElementById('activityChart').getContext('2d');
    const activityChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels,
            datasets: [
                {
                    label: 'Simpanan',
                    data: simpananData,
                    borderColor: 'rgba(40, 167, 69, 1)',
                    backgroundColor: 'rgba(40, 167, 69, 0.1)',
                    tension: 0.4,
                    fill: true
                },
                {
                    label: 'Pinjaman',
                    data: pinjamanData,
                    borderColor: 'rgba(220, 53, 69, 1)',
                    backgroundColor: 'rgba(220, 53, 69, 0.1)',
                    tension: 0.4,
                    fill: true
                },
                {
                    label: 'Angsuran',
                    data: angsuranData,
                    borderColor: 'rgba(255, 193, 7, 1)',
                    backgroundColor: 'rgba(255, 193, 7, 0.1)',
                    tension: 0.4,
                    fill: true
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    position: 'top',
                },
                tooltip: {
                    callbacks: {
                        label: function(context) {
                            let label = context.dataset.label || '';
                            if (label) {
                                label += ': ';
                            }
                            label += formatCurrency(context.parsed.y);
                            return label;
                        }
                    }
                }
            },
            scales: {
                y: {
                    beginAtZero: true,
                    ticks: {
                        callback: function(value) {
                            return formatCurrency(value);
                        }
                    }
                },
                x: {
                    display: true,
                    title: {
                        display: true,
                        text: 'Tanggal'
                    }
                }
            }
        }
    });

    console.log('Dashboard Bendahara loaded successfully');
});
