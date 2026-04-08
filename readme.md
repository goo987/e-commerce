# Cara menjalankan Hostel Mart
### saya menggunakan golang,chi,templ,alpine js,tailwind css,SweetAlert2,Font Awesome,Google Font,sqlite dan drivernya modernc sqlite,dan air(live reload)

#### gunakan 
`go get -u github.com/go-chi/chi/v5` 
### terlebih dahulu
#### kemudian 
`go install github.com/a-h/templ/cmd/templ@latest`
#### atau jika ingin lebih mudah gunakan 
`go mod tidy`
#### setelah sudah menjalankan perintah2 tersebut maka jalankan perintah 
`templ generate`
### dan setelah itu jalankan perintah 
`go run cmd/web/main.go`
### dan klik `Ctrl + C` 
### untuk menghentikan server
#### jika ingin melihat databasenya gunakan extension SQLTools dan SQLTools SQLite