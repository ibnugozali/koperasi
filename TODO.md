# TODO: Add Status Anggota and Fakultas to Registration Form

## Tasks
- [x] Update database/koperasi.sql to add new columns status_anggota and fakultas to anggota table
- [x] Update models/anggota.go to include StatusAnggota and Fakultas fields
- [x] Update repository/anggota_repository.go CreateAnggota function to insert new fields
- [x] Update templates/utama/register.html to add form fields for status_anggota and fakultas

## Followup
- [x] Execute the updated SQL to alter the database
- [x] Test the registration form (server running on :8080, form updated with new fields)
