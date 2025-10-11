# TODO: Add Member Management Features to Admin Dashboard

## Information Gathered
- Admin dashboard currently shows pending members for confirmation and allows editing dashboard anggota content.
- Controller has AdminDashboard, ConfirmMembership, and halaman edit functions.
- Repository has functions for anggota: GetPendingAnggota, UpdateAnggotaStatus, CreateAnggota, GetAnggotaByID.
- Routes are set up for admin with AuthRequired and AdminOnly middleware.
- Models have Anggota struct with fields like IDAnggota, NamaAnggota, etc.

## Plan
- Add new controller functions in controllers/admin_controller.go: ListAllAnggota, ViewAnggota, EditAnggota, UpdateAnggota, DeleteAnggota.
- Add new routes in routes/routes.go for admin group: GET /admin/anggota, GET /admin/anggota/:id, GET /admin/anggota/edit/:id, POST /admin/anggota/update/:id, POST /admin/anggota/delete/:id.
- Update templates/admin/admin_dashboard.html to include a section for member management with links.
- Create new templates: templates/admin/admin_anggota_list.html, templates/admin/admin_anggota_view.html, templates/admin/admin_anggota_edit.html.
- Add repository functions in repository/anggota_repository.go: GetAllAnggota, DeleteAnggota.

## Dependent Files
- controllers/admin_controller.go
- routes/routes.go
- templates/admin/admin_dashboard.html
- repository/anggota_repository.go
- New files: templates/admin/admin_anggota_list.html, templates/admin/admin_anggota_view.html, templates/admin/admin_anggota_edit.html

## Followup Steps
- Test the new routes and functions by running the server and accessing as admin.
- Ensure proper authentication and authorization.
- Handle errors appropriately in templates and controllers.
- Verify database operations work correctly.

## Steps to Complete
- [x] Add repository functions: GetAllAnggota, DeleteAnggota in repository/anggota_repository.go
- [x] Add controller functions: ListAllAnggota, ViewAnggota, EditAnggota, UpdateAnggota, DeleteAnggota in controllers/admin_controller.go
- [x] Add new routes in routes/routes.go
- [x] Update admin_dashboard.html to add member management section
- [x] Create admin_anggota_list.html template
- [x] Create admin_anggota_view.html template
- [x] Create admin_anggota_edit.html template
- [x] Test the features
