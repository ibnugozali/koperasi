// JS untuk konfirmasi transaksi bendahara (AJAX, feedback, dsb)
document.addEventListener('DOMContentLoaded', function() {
    // Dropdown Jenis Simpanan dinamis
    const select = document.getElementById('jenisSimpananSelect');
    if (select) {
        fetch('/api/jenis-simpanan')
            .then(res => res.json())
            .then(data => {
                console.log('Jenis simpanan API:', data);
                select.innerHTML = '';
                if (data && Array.isArray(data.jenis_simpanan) && data.jenis_simpanan.length > 0) {
                    data.jenis_simpanan.forEach(function(item) {
                        const opt = document.createElement('option');
                        opt.value = item.key;
                        opt.textContent = item.nama;
                        select.appendChild(opt);
                    });
                } else {
                    select.innerHTML = '<option value="pokok">Pokok</option><option value="wajib">Wajib</option><option value="sukarela">Sukarela</option><option value="hari_raya">Hari Raya</option>';
                }
            })
            .catch((err) => {
                console.error('Gagal load jenis simpanan:', err);
                select.innerHTML = '<option value="pokok">Pokok</option><option value="wajib">Wajib</option><option value="sukarela">Sukarela</option><option value="hari_raya">Hari Raya</option>';
            });
    }
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
