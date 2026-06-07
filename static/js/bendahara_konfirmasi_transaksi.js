// JS untuk konfirmasi transaksi bendahara (AJAX, feedback, dsb)
// Single DOMContentLoaded untuk menghindari duplikat event listener
document.addEventListener('DOMContentLoaded', function() {
    // --- Dropdown Jenis Simpanan dinamis ---
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

    // --- Konfirmasi/tolak transaksi via AJAX ---
    document.querySelectorAll('form[action*="/bendahara/konfirmasi-transaksi/"]:not([data-no-ajax="true"])').forEach(function(form) {
        form.addEventListener('submit', function(e) {
            e.preventDefault();
            let url = form.getAttribute('action');
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
                    location.reload();
                } else {
                    alert(data.error || 'Gagal memproses transaksi');
                }
            })
            .catch(() => alert('Terjadi kesalahan jaringan'));
        });
    });

    // --- AJAX submit untuk form simpanan (dengan proteksi double-submit) ---
    var isSubmittingSimpanan = false;
    var formSimpanan = document.querySelector('form[action="/bendahara/transaksi/simpanan"]');
    if (formSimpanan) {
        var btnSimpanan = formSimpanan.querySelector('button[type="submit"]');
        formSimpanan.addEventListener('submit', function(e) {
            e.preventDefault();
            if (isSubmittingSimpanan) return; // Cegah klik ganda
            isSubmittingSimpanan = true;
            if (btnSimpanan) {
                btnSimpanan.disabled = true;
                btnSimpanan.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Menyimpan...';
            }

            var formData = new FormData(formSimpanan);
            fetch('/bendahara/transaksi/simpanan', {
                method: 'POST',
                body: formData
            })
            .then(res => res.json())
            .then(data => {
                isSubmittingSimpanan = false;
                if (btnSimpanan) {
                    btnSimpanan.disabled = false;
                    btnSimpanan.innerHTML = '<i class="fa-solid fa-plus"></i> Simpan';
                }
                if (data.message) {
                    showSimpananNotif('success', data.message);
                    formSimpanan.reset();
                    location.reload(); // Agar Simpanan Pending / Cicilan Pending ter-update otomatis
                } else {
                    showSimpananNotif('danger', data.error || 'Gagal mencatat simpanan');
                }
            })
            .catch(function() {
                isSubmittingSimpanan = false;
                if (btnSimpanan) {
                    btnSimpanan.disabled = false;
                    btnSimpanan.innerHTML = '<i class="fa-solid fa-plus"></i> Simpan';
                }
                showSimpananNotif('danger', 'Terjadi kesalahan jaringan');
            });
        });
    }

    // --- AJAX submit untuk form angsuran (dengan proteksi double-submit) ---
    var isSubmittingAngsuran = false;
    var formAngsuran = document.querySelector('form[action="/bendahara/transaksi/angsuran"]');
    if (formAngsuran) {
        var inputAnggotaAngsuran = formAngsuran.querySelector('input[name="id_anggota"]');
        var selectPinjamanAngsuran = formAngsuran.querySelector('select[name="jenis_angsuran"]');
        var inputJumlahAngsuran = formAngsuran.querySelector('input[name="jumlah_angsuran"]');
        var pinjamanAngsuranByID = {};

        function resetPinjamanAngsuranOptions(message) {
            pinjamanAngsuranByID = {};
            if (selectPinjamanAngsuran) {
                selectPinjamanAngsuran.innerHTML = '<option value="">' + message + '</option>';
            }
            if (inputJumlahAngsuran) {
                inputJumlahAngsuran.value = '';
            }
        }

        function loadPinjamanAngsuran() {
            if (!inputAnggotaAngsuran || !selectPinjamanAngsuran) return;

            var idAnggota = inputAnggotaAngsuran.value.trim();
            if (!idAnggota) {
                resetPinjamanAngsuranOptions('Masukkan ID anggota dulu...');
                return;
            }

            resetPinjamanAngsuranOptions('Memuat pinjaman aktif...');
            fetch('/api/jenis-angsuran?id_anggota=' + encodeURIComponent(idAnggota), {
                credentials: 'same-origin'
            })
            .then(function(res) { return res.json(); })
            .then(function(data) {
                var items = data && Array.isArray(data.jenis_angsuran) ? data.jenis_angsuran : [];
                selectPinjamanAngsuran.innerHTML = '<option value="">Pilih pinjaman aktif...</option>';
                pinjamanAngsuranByID = {};

                if (items.length === 0) {
                    resetPinjamanAngsuranOptions('Tidak ada pinjaman aktif');
                    return;
                }

                items.forEach(function(item) {
                    var opt = document.createElement('option');
                    opt.value = item.key;
                    opt.textContent = item.nama;
                    opt.dataset.perkiraanAngsuran = item.perkiraan_angsuran || 0;
                    selectPinjamanAngsuran.appendChild(opt);
                    pinjamanAngsuranByID[String(item.key)] = item;
                });
            })
            .catch(function() {
                resetPinjamanAngsuranOptions('Gagal memuat pinjaman');
            });
        }

        function syncJumlahAngsuranFromPinjaman() {
            if (!selectPinjamanAngsuran || !inputJumlahAngsuran) return;
            var selected = pinjamanAngsuranByID[String(selectPinjamanAngsuran.value)];
            var perkiraan = selected ? Number(selected.perkiraan_angsuran || 0) : 0;
            inputJumlahAngsuran.value = perkiraan > 0 ? String(Math.round(perkiraan)) : '';
        }

        if (inputAnggotaAngsuran) {
            inputAnggotaAngsuran.addEventListener('change', loadPinjamanAngsuran);
            inputAnggotaAngsuran.addEventListener('blur', loadPinjamanAngsuran);
        }
        if (selectPinjamanAngsuran) {
            selectPinjamanAngsuran.addEventListener('change', syncJumlahAngsuranFromPinjaman);
        }

        var btnAngsuran = formAngsuran.querySelector('button[type="submit"]');
        formAngsuran.addEventListener('submit', function(e) {
            e.preventDefault();
            if (isSubmittingAngsuran) return; // Cegah klik ganda
            isSubmittingAngsuran = true;
            if (btnAngsuran) {
                btnAngsuran.disabled = true;
                btnAngsuran.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Menyimpan...';
            }

            var formData = new FormData(formAngsuran);
            fetch('/bendahara/transaksi/angsuran', {
                method: 'POST',
                body: formData
            })
            .then(res => res.json())
            .then(data => {
                isSubmittingAngsuran = false;
                if (btnAngsuran) {
                    btnAngsuran.disabled = false;
                    btnAngsuran.innerHTML = '<i class="fa-solid fa-plus"></i> Simpan';
                }
                if (data.message) {
                    showAngsuranNotif('success', data.message);
                    formAngsuran.reset();
                    location.reload(); // Agar Simpanan Pending / Cicilan Pending ter-update otomatis
                } else {
                    showAngsuranNotif('danger', data.error || 'Gagal mencatat angsuran');
                }
            })
            .catch(function() {
                isSubmittingAngsuran = false;
                if (btnAngsuran) {
                    btnAngsuran.disabled = false;
                    btnAngsuran.innerHTML = '<i class="fa-solid fa-plus"></i> Simpan';
                }
                showAngsuranNotif('danger', 'Terjadi kesalahan jaringan');
            });
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

// AJAX submit untuk form import potong gaji Excel
function escapeHtml(text) {
    if (!text) return '';
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(text));
    return div.innerHTML;
}

document.addEventListener('DOMContentLoaded', function() {
    var potongGajiForm = document.getElementById('potongGajiImportForm');
    var potongGajiBtn = document.getElementById('potongGajiImportBtn');
    var potongGajiAlert = document.getElementById('potongGajiAlert');

    if (potongGajiForm) {
        potongGajiForm.addEventListener('submit', function(e) {
            e.preventDefault();
            if (potongGajiBtn) {
                potongGajiBtn.disabled = true;
                potongGajiBtn.innerHTML = '<span class="spinner-border spinner-border-sm me-1"></span>Memproses...';
            }

            var formData = new FormData(potongGajiForm);
            fetch('/bendahara/konfirmasi-transaksi/import-potong-gaji', {
                method: 'POST',
                body: formData,
                credentials: 'same-origin'
            })
            .then(function(res) { return res.json(); })
            .then(function(data) {
                if (potongGajiAlert) {
                    if (data.error) {
                        var errorHtml = '<div class="alert alert-danger"><strong>Gagal:</strong> ' + escapeHtml(data.error) +
                            '<br><small>Berhasil: ' + (data.success || 0) + ', Gagal: ' + (data.failed || 0) + '</small>';
                        if (data.parseErrors && data.parseErrors.length > 0) {
                            errorHtml += '<ul class="mb-0 mt-1">';
                            data.parseErrors.forEach(function(err) {
                                errorHtml += '<li><small>' + escapeHtml(err) + '</small></li>';
                            });
                            errorHtml += '</ul>';
                        }
                        errorHtml += '</div>';
                        potongGajiAlert.innerHTML = errorHtml;
                    } else {
                        var successHtml = '<div class="alert alert-success"><strong>Berhasil!</strong> ' + escapeHtml(data.message) +
                            '<br><small>Berhasil: ' + (data.success || 0) + ', Gagal: ' + (data.failed || 0) + '</small>';
                        if (data.parseErrors && data.parseErrors.length > 0) {
                            successHtml += '<ul class="mb-0 mt-1">';
                            data.parseErrors.forEach(function(err) {
                                successHtml += '<li><small>' + escapeHtml(err) + '</small></li>';
                            });
                            successHtml += '</ul>';
                        }
                        successHtml += '</div>';
                        potongGajiAlert.innerHTML = successHtml;
                        setTimeout(function() {
                            location.reload();
                        }, 1500);
                    }
                }
            })
            .catch(function() {
                if (potongGajiAlert) {
                    potongGajiAlert.innerHTML = '<div class="alert alert-danger">Terjadi kesalahan jaringan saat upload file.</div>';
                }
            })
            .finally(function() {
                if (potongGajiBtn) {
                    potongGajiBtn.disabled = false;
                    potongGajiBtn.innerHTML = '<i class="fa-solid fa-upload me-1"></i>Import Potong Gaji';
                }
            });
        });
    }
});

