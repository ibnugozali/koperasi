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
            let url = form.getAttribute('action');
            // Debug log
            console.log('Submit konfirmasi-transaksi:', form, url);
            // Validasi URL
            if (!url || url.indexOf('/bendahara/konfirmasi-transaksi/') === -1) {
                alert('Form action tidak valid: ' + url);
                return;
            }
            const formData = new FormData(form);
            fetch(url, {
                method: 'POST',
                body: formData,
                credentials: 'same-origin'
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
// AJAX submit untuk form angsuran (tanpa reload)
document.addEventListener('DOMContentLoaded', function() {
    var formAngsuran = document.querySelector('form[action="/bendahara/transaksi/angsuran"]');
    if (formAngsuran) {
        formAngsuran.addEventListener('submit', function(e) {
            e.preventDefault();
            var formData = new FormData(formAngsuran);
            fetch('/bendahara/transaksi/angsuran', {
                method: 'POST',
                body: formData
            })
            .then(res => res.json())
            .then(data => {
                if (data.message) {
                    showAngsuranNotif('success', data.message);
                    formAngsuran.reset();
                } else {
                    showAngsuranNotif('danger', data.error || 'Gagal mencatat angsuran');
                }
            })
            .catch(() => showAngsuranNotif('danger', 'Terjadi kesalahan jaringan'));
        });
    }
});

// AJAX submit untuk form simpanan (tanpa reload)
document.addEventListener('DOMContentLoaded', function() {
    var formSimpanan = document.querySelector('form[action="/bendahara/transaksi/simpanan"]');
    if (formSimpanan) {
        formSimpanan.addEventListener('submit', function(e) {
            e.preventDefault();
            var formData = new FormData(formSimpanan);
            fetch('/bendahara/transaksi/simpanan', {
                method: 'POST',
                body: formData
            })
            .then(res => res.json())
            .then(data => {
                if (data.message) {
                    showSimpananNotif('success', data.message);
                    formSimpanan.reset();
                } else {
                    showSimpananNotif('danger', data.error || 'Gagal mencatat simpanan');
                }
            })
            .catch(() => showSimpananNotif('danger', 'Terjadi kesalahan jaringan'));
        });
    }
});

// Fungsi notifikasi angsuran
function showAngsuranNotif(type, msg) {
    var notifId = 'angsuranNotif';
    var notif = document.getElementById(notifId);
    if (!notif) {
        notif = document.createElement('div');
        notif.id = notifId;
        notif.className = 'alert alert-' + type + ' mt-2';
        var parent = document.querySelector('form[action="/bendahara/transaksi/angsuran"]').parentNode;
        parent.insertBefore(notif, parent.firstChild);
    }
    notif.className = 'alert alert-' + type + ' mt-2';
    notif.textContent = msg;
    setTimeout(function() {
        if (notif) notif.remove();
    }, 3500);
}

// Fungsi notifikasi simpanan
function showSimpananNotif(type, msg) {
    var notifId = 'simpananNotif';
    var notif = document.getElementById(notifId);
    if (!notif) {
        notif = document.createElement('div');
        notif.id = notifId;
        notif.className = 'alert alert-' + type + ' mt-2';
        var parent = document.querySelector('form[action="/bendahara/transaksi/simpanan"]').parentNode;
        parent.insertBefore(notif, parent.firstChild);
    }
    notif.className = 'alert alert-' + type + ' mt-2';
    notif.textContent = msg;
    setTimeout(function() {
        if (notif) notif.remove();
    }, 3500);
}
