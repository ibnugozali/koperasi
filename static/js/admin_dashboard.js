document.addEventListener('DOMContentLoaded', function() {
    // Data aktivitas dari server (diasumsikan dikirim melalui template)
    const activityData = JSON.parse('{{ .AktivitasData }}');

    // Mengelompokkan data berdasarkan tanggal
    const groupedData = activityData.reduce((acc, item) => {
        const date = new Date(item.Tanggal).toISOString().split('T')[0]; // Format YYYY-MM-DD
        if (!acc[date]) {
            acc[date] = { Simpanan: 0, Pinjaman: 0, Angsuran: 0 };
        }
        acc[date][item.Jenis] += item.Jumlah;
        return acc;
    }, {});

    // Mengubah ke format yang sesuai untuk Chart.js
    const labels = Object.keys(groupedData).sort();
    const simpananData = labels.map(date => groupedData[date].Simpanan);
    const pinjamanData = labels.map(date => groupedData[date].Pinjaman);
    const angsuranData = labels.map(date => groupedData[date].Angsuran);

    // Membuat grafik menggunakan Chart.js
    const ctx = document.getElementById('activityChart').getContext('2d');
    new Chart(ctx, {
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
                            return context.dataset.label + ': Rp ' + context.parsed.y.toLocaleString('id-ID');
                        }
                    }
                }
            },
            scales: {
                x: {
                    display: true,
                    title: {
                        display: true,
                        text: 'Tanggal'
                    }
                },
                y: {
                    display: true,
                    title: {
                        display: true,
                        text: 'Jumlah (Rp)'
                    },
                    ticks: {
                        callback: function(value) {
                            return 'Rp ' + value.toLocaleString('id-ID');
                        }
                    }
                }
            }
        }
    });
});
