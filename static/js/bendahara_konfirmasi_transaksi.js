// JS untuk konfirmasi transaksi bendahara (AJAX, feedback, dsb)
document.addEventListener('DOMContentLoaded', function() {
    // Konfirmasi/tolak transaksi tanpa reload (optional improvement)
    document.querySelectorAll('form[action*="/bendahara/konfirmasi-transaksi/"]').forEach(function(form) {
        form.addEventListener('submit', function(e) {
            e.preventDefault();
            const url = form.action;
            const formData = new FormData(form);
            fetch(url, {
                method: 'POST',
                body: formData
            })
            .then(res => res.json())
            .then(data => {
                if (data.success || data.status === 'ok') {
                    // Berhasil, reload atau update baris
                    location.reload();
                } else {
                    alert(data.error || 'Gagal memproses transaksi');
                }
            })
            .catch(() => alert('Terjadi kesalahan jaringan'));
        });
    });
});
