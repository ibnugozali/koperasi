// Function to convert number to words in Indonesian
function numberToWords(num) {
    if (num === 0) return 'nol';

    const units = ['', 'satu', 'dua', 'tiga', 'empat', 'lima', 'enam', 'tujuh', 'delapan', 'sembilan'];
    const teens = ['sepuluh', 'sebelas', 'dua belas', 'tiga belas', 'empat belas', 'lima belas', 'enam belas', 'tujuh belas', 'delapan belas', 'sembilan belas'];
    const tens = ['', '', 'dua puluh', 'tiga puluh', 'empat puluh', 'lima puluh', 'enam puluh', 'tujuh puluh', 'delapan puluh', 'sembilan puluh'];
    const thousands = ['', 'ribu', 'juta', 'miliar', 'triliun'];

    function convertHundreds(n) {
        let str = '';
        const h = Math.floor(n / 100);
        const t = Math.floor((n % 100) / 10);
        const u = n % 10;

        if (h > 0) {
            if (h === 1) str += 'seratus ';
            else str += units[h] + ' ratus ';
        }

        if (t > 0) {
            if (t === 1) {
                str += teens[u] + ' ';
                return str.trim();
            } else {
                str += tens[t] + ' ';
            }
        }

        if (u > 0) {
            str += units[u] + ' ';
        }

        return str.trim();
    }

    function convertToWords(n) {
        if (n === 0) return '';

        let result = '';
        let thousandIndex = 0;

        while (n > 0) {
            const chunk = n % 1000;
            if (chunk > 0) {
                const chunkWords = convertHundreds(chunk);
                if (thousandIndex === 0) {
                    result = chunkWords + ' ' + result;
                } else {
                    result = chunkWords + ' ' + thousands[thousandIndex] + ' ' + result;
                }
            }
            n = Math.floor(n / 1000);
            thousandIndex++;
        }

        return result.trim();
    }

    const words = convertToWords(num);
    return words ? words + ' rupiah' : 'nol rupiah';
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', function() {
    const unitKerjaValue = document.querySelector('script') ? document.querySelector('script').textContent.match(/unitKerjaValue = '([^']+)'/)?.[1] : '';
    const gajiContainer = document.getElementById('gaji_bulanan_container');
    const gajiInput = document.getElementById('gaji_bulanan');

    if (unitKerjaValue === '03') { // Mahasiswa
        if (gajiContainer) gajiContainer.style.display = 'none';
        if (gajiInput) {
            gajiInput.required = false;
            gajiInput.value = '';
        }
    } else { // Dosen atau Karyawan/Staff
        if (gajiContainer) gajiContainer.style.display = 'block';
        if (gajiInput) gajiInput.required = true;
    }

    // Hitung perkiraan angsuran saat input berubah
    calculateAngsuran();

    // Add event listener for jumlah_pinjaman to update terbilang
    const jumlahPinjamanInput = document.getElementById('jumlah_pinjaman');
    const terbilangInput = document.getElementById('jumlah_pinjaman_terbilang');

    if (jumlahPinjamanInput && terbilangInput) {
        jumlahPinjamanInput.addEventListener('input', function() {
            const value = parseInt(this.value) || 0;
            terbilangInput.value = numberToWords(value);
        });

        // Set initial value if there's already a value
        const initialValue = parseInt(jumlahPinjamanInput.value) || 0;
        if (initialValue > 0) {
            terbilangInput.value = numberToWords(initialValue);
        }
    }

    // Removed AJAX form submit handler to allow normal form submission and redirect

});

// Function to calculate perkiraan angsuran
function calculateAngsuran() {
    const jumlahPinjaman = parseFloat(document.getElementById('jumlah_pinjaman')?.value) || 0;
    const jangkaWaktu = parseInt(document.getElementById('jangka_waktu')?.value) || 0;
    const gajiBulanan = parseFloat(document.getElementById('gaji_bulanan')?.value) || 0;

    // Get values from template variables (fallback if not available)
    const unitKerjaValue = document.querySelector('script') ? document.querySelector('script').textContent.match(/unitKerjaValue = '([^']+)'/)?.[1] : '';
    const totalSimpanan = parseFloat(document.querySelector('script') ? document.querySelector('script').textContent.match(/totalSimpanan = ([0-9.]+)/)?.[1] : '0') || 0;

    let limitPinjaman = 0;
    let kemampuanBayar = 0;
    let jenisAnggota = '';

    if (unitKerjaValue === '03') { // Mahasiswa
        jenisAnggota = 'Mahasiswa';
        limitPinjaman = 5 * totalSimpanan;
        kemampuanBayar = limitPinjaman; // Untuk mahasiswa sama
    } else { // Dosen/Staff
        jenisAnggota = 'Dosen/Staff';
        if (gajiBulanan > 0 && jangkaWaktu > 0) {
            // Get bunga value from template/data attribute or script variable
            const bungaTerkini = parseFloat(document.getElementById('bunga_info')?.textContent.match(/[\d.]+/)?.[0]) || 2.0;
            const bungaDecimal = bungaTerkini / 100;
            // Langkah 1 - Kemampuan bayar: 0.4 × gaji × tenor
            kemampuanBayar = 0.4 * gajiBulanan * jangkaWaktu;
            // Langkah 3 - Limit Pinjaman (untuk informasi): (0.4 × gaji × tenor) × (1 - (bunga × tenor))
            limitPinjaman = kemampuanBayar * (1 - (bungaDecimal * jangkaWaktu));
        }
    }

    // Update limit info
    const limitInfo = document.getElementById('limit_info');
    if (limitInfo) {
        if (jenisAnggota === 'Mahasiswa') {
            limitInfo.innerHTML = '<strong>Kemampuan Bayar:</strong> Rp ' + kemampuanBayar.toLocaleString('id-ID') + ' (5x total simpanan)';
        } else if (jenisAnggota === 'Dosen/Staff' && kemampuanBayar > 0) {
            limitInfo.innerHTML = '<strong>Kemampuan Bayar:</strong> Rp ' + kemampuanBayar.toLocaleString('id-ID') + ' (0.4 × gaji × tenor)';
        } else {
            limitInfo.innerHTML = '<strong>Kemampuan Bayar:</strong> Masukkan gaji bulanan dan tenor untuk menghitung';
        }
    }

    // Check limit and show warning - menggunakan kemampuan bayar (Langkah 1)
    const limitWarning = document.getElementById('limit_warning');
    if (limitWarning) {
        if (jumlahPinjaman > kemampuanBayar && kemampuanBayar > 0) {
            limitWarning.textContent = 'Jumlah pinjaman melebihi limit maksimal';
            limitWarning.style.color = 'red';
        } else if (kemampuanBayar > 0) {
            limitWarning.textContent = '✓ Jumlah pinjaman dalam batas limit';
            limitWarning.style.color = 'green';
        } else {
            limitWarning.textContent = '';
        }
    }

    // Calculate perkiraan angsuran (pokok + bunga dari database)
    const perkiraanAngsuran = document.getElementById('perkiraan_angsuran');
    if (perkiraanAngsuran && jumlahPinjaman > 0 && jangkaWaktu > 0) {
        // Get bunga value from template/data attribute or script variable
        const bungaTerkini = parseFloat(document.getElementById('bunga_info')?.textContent.match(/[\d.]+/)?.[0]) || 2.0;
        const bungaDecimal = bungaTerkini / 100;
        // Rumus: (pinjaman/tenor) + ((bunga × pinjaman) / tenor)
        const pokok = jumlahPinjaman / jangkaWaktu;
        const bungaTotal = bungaDecimal * jumlahPinjaman;
        const bungaPerBulan = bungaTotal / jangkaWaktu;
        const totalAngsuran = pokok + bungaPerBulan;
        perkiraanAngsuran.value = Math.round(totalAngsuran);
    }
}

// Add event listeners for real-time calculation
document.addEventListener('DOMContentLoaded', function() {
    const jumlahPinjaman = document.getElementById('jumlah_pinjaman');
    const jangkaWaktu = document.getElementById('jangka_waktu');
    const gajiBulanan = document.getElementById('gaji_bulanan');

    if (jumlahPinjaman) jumlahPinjaman.addEventListener('input', calculateAngsuran);
    if (jangkaWaktu) jangkaWaktu.addEventListener('change', calculateAngsuran);
    if (gajiBulanan) gajiBulanan.addEventListener('input', calculateAngsuran);

    // Add event listeners for metode pencairan (transfer bank)
    const pencairanTransfer = document.getElementById('pencairan_transfer');
    const pencairanTunai = document.getElementById('pencairan_tunai');
    const nomorRekeningContainer = document.getElementById('nomor_rekening_container');
    const nomorRekeningInput = document.getElementById('nomor_rekening');

    if (pencairanTransfer) {
        pencairanTransfer.addEventListener('change', function() {
            if (this.checked) {
                nomorRekeningContainer.style.display = 'block';
                nomorRekeningInput.required = true;
            }
        });
    }

    if (pencairanTunai) {
        pencairanTunai.addEventListener('change', function() {
            if (this.checked) {
                nomorRekeningContainer.style.display = 'none';
                nomorRekeningInput.required = false;
                nomorRekeningInput.value = '';
            }
        });
    }

    // Set initial state (transfer is checked by default)
    if (pencairanTransfer && pencairanTransfer.checked && nomorRekeningContainer) {
        nomorRekeningContainer.style.display = 'block';
        nomorRekeningInput.required = true;
    }

    // Function to handle "Centang Semua" checkbox
    const cekSemua = document.getElementById('cek_semua');
    if (cekSemua) {
        cekSemua.addEventListener('change', function() {
            const checkboxes = document.querySelectorAll('.pernyataan-checkbox');
            checkboxes.forEach(checkbox => {
                checkbox.checked = this.checked;
            });
        });
    }

    // Function to update "Centang Semua" when individual checkboxes change
    document.querySelectorAll('.pernyataan-checkbox').forEach(checkbox => {
        checkbox.addEventListener('change', function() {
            const allCheckboxes = document.querySelectorAll('.pernyataan-checkbox');
            const checkedCheckboxes = document.querySelectorAll('.pernyataan-checkbox:checked');
            const masterCheckbox = document.getElementById('cek_semua');

            if (masterCheckbox) {
                masterCheckbox.checked = checkedCheckboxes.length === allCheckboxes.length;
            }
        });
    });
});
