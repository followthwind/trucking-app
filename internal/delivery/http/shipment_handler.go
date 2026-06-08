package http

import (
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"trucking-app/internal/domain"

	"github.com/jung-kurt/gofpdf"

	"github.com/xuri/excelize/v2"
)

type ShipmentHandler struct {
	usecase domain.ShipmentUsecase
	docRepo domain.DocumentRepository // Tambahkan ini agar handler bisa ambil file dari MinIO
}

// Update fungsi New dengan memasukkan d domain.DocumentRepository
func NewShipmentHandler(u domain.ShipmentUsecase, d domain.DocumentRepository) *ShipmentHandler {
	return &ShipmentHandler{
		usecase: u,
		docRepo: d,
	}
}

// RegisterRoutes mendaftarkan jalur URL web ke router HTTP Go
func (h *ShipmentHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", h.ShowFormAndTable)
	mux.HandleFunc("/submit", h.SubmitForm)
	mux.HandleFunc("/view", h.ViewFile)
	mux.HandleFunc("/export-excel", h.ExportExcel)
	mux.HandleFunc("/generate-invoice", h.GenerateInvoice) // <-- Tambahkan rute ini
	mux.HandleFunc("/delete", h.DeleteShipment)
}

// ShowFormAndTable menampilkan halaman utama berisi Form Input dan Tabel Quotation
func (h *ShipmentHandler) ShowFormAndTable(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Ambil semua data yang ada di database lewat usecase
	shipments, err := h.usecase.GetAllShipments(r.Context())
	if err != nil {
		http.Error(w, "Gagal mengambil data database: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Render file HTML ke browser
	tmpl := template.Must(template.ParseFiles("internal/delivery/http/index.html"))
	tmpl.Execute(w, shipments)
}

// SubmitForm memproses input data form dan file upload dari browser
func (h *ShipmentHandler) SubmitForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method tidak diizinkan", http.StatusMethodNotAllowed)
		return
	}

	// 1. Batasi ukuran maksimal upload file (5MB)
	err := r.ParseMultipartForm(5 << 20)
	if err != nil {
		http.Error(w, "File terlalu besar! Maksimal 5MB", http.StatusBadRequest)
		return
	}

	// 2. Ambil data teks biasa dari Form HTML
	qty, _ := strconv.Atoi(r.FormValue("qty"))
	loloRate, _ := strconv.ParseInt(r.FormValue("lolo_rate"), 10, 64)
	rate, _ := strconv.ParseInt(r.FormValue("rate"), 10, 64)
	buyingPrice, _ := strconv.ParseInt(r.FormValue("buying_price"), 10, 64)

	shipment := &domain.Shipment{
		ItemDescription: r.FormValue("item_description"),
		Origin:          r.FormValue("origin"),
		Destination:     r.FormValue("destination"),
		BLNumber:        r.FormValue("bl_number"),
		ContainerNumber: r.FormValue("container_number"),
		ReturnTo:        r.FormValue("return_to"),
		Qty:             qty,
		Rate:            rate,
		LoloRate:        loloRate,
		BuyingPrice:     buyingPrice,
		Remark:          r.FormValue("remark"),
	}

	// // Buat penampung kosong untuk data file dokumen mentah
	// var fileData []byte
	// var contentType string

	// // 3. Ambil data file dokumen yang di-upload (PDF/Gambar) jika ada
	// file, header, err := r.FormFile("document")
	// if err == nil {
	// 	defer file.Close()
	// 	// Baca file menjadi pecahan byte data mentah
	// 	data, err := io.ReadAll(file)
	// 	if err == nil {
	// 		fileData = data
	// 		contentType = header.Header.Get("Content-Type")
	// 	}
	// }

	// 4. Simpan seluruh data teks ke Postgres + upload file mentah ke MinIO via Usecase
	err = h.usecase.InsertShipment(r.Context(), shipment)
	if err != nil {
		http.Error(w, "Gagal memproses data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Setelah sukses, segarkan halaman browser kembali ke menu utama
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ViewFile bertugas mengambil file stream dari MinIO dan melemparnya langsung ke tab browser baru
func (h *ShipmentHandler) ViewFile(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Query().Get("file")
	if fileName == "" {
		http.Error(w, "Nama file tidak boleh kosong", http.StatusBadRequest)
		return
	}

	// Tarik object file langsung dari bucket MinIO
	object, err := h.docRepo.GetObject(r.Context(), fileName)
	if err != nil {
		http.Error(w, "File tidak ditemukan di storage: "+err.Error(), http.StatusNotFound)
		return
	}
	defer object.Close()

	// Set header agar browser tahu ini file dokumen yang mau ditampilkan (bukan didownload mentah-mentah)
	w.Header().Set("Content-Disposition", "inline; filename="+fileName)

	// Stream datanya langsung ke layar browser
	io.Copy(w, object)
}

func (h *ShipmentHandler) ExportExcel(w http.ResponseWriter, r *http.Request) {
	// 1. Ambil seluruh data dari database lewat usecase
	shipments, err := h.usecase.GetAllShipments(r.Context())
	if err != nil {
		http.Error(w, "Gagal mengambil data untuk excel: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Buat file Excel baru di memori server menggunakan excelize
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			log.Println(err)
		}
	}()

	sheetName := "Quotation Report"
	f.SetSheetName("Sheet1", sheetName)

	// 3. Buat Header Tabel di baris pertama Excel
	headers := []string{"Item Description", "Origin", "Destination", "QTY", "Rate", "Selling Price", "Buying Price", "Gross Profit", "Profit %", "Created At"}
	for colNum, headerTitle := range headers {
		// Mengubah angka index menjadi nama kolom Excel (0=A, 1=B, dst)
		cellCoordinate, _ := excelize.ColumnNumberToName(colNum + 1)
		f.SetCellValue(sheetName, cellCoordinate+"1", headerTitle)
	}

	// 4. Lakukan looping untuk mengisi baris data (dimulai dari baris ke-2)
	for index, s := range shipments {
		rowNum := index + 2

		f.SetCellValue(sheetName, "A"+strconv.Itoa(rowNum), s.ItemDescription)
		f.SetCellValue(sheetName, "B"+strconv.Itoa(rowNum), s.Origin)
		f.SetCellValue(sheetName, "C"+strconv.Itoa(rowNum), s.Destination)
		f.SetCellValue(sheetName, "D"+strconv.Itoa(rowNum), s.Qty)
		f.SetCellValue(sheetName, "E"+strconv.Itoa(rowNum), s.Rate)
		f.SetCellValue(sheetName, "F"+strconv.Itoa(rowNum), s.Amount)
		f.SetCellValue(sheetName, "G"+strconv.Itoa(rowNum), s.BuyingPrice)
		f.SetCellValue(sheetName, "H"+strconv.Itoa(rowNum), s.GrossProfit)
		f.SetCellValue(sheetName, "I"+strconv.Itoa(rowNum), strconv.Itoa(s.ProfitPercentage)+"%")
		f.SetCellValue(sheetName, "J"+strconv.Itoa(rowNum), s.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	// 5. Atur Header HTTP agar browser mengenali payload ini sebagai file Excel siap unduh
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=Trucking_Quotation_Report.xlsx")

	// Kirim file data dari memori Go langsung ke aliran download browser
	if err := f.Write(w); err != nil {
		http.Error(w, "Gagal mengirim file excel ke unduhan: "+err.Error(), http.StatusInternalServerError)
	}
}

func formatRupiah(amount int64) string {
	return "Rp " + strconv.FormatInt(amount, 10)
}

// GenerateInvoice membuat invoice PDF resmi untuk satu shipment tertentu
func (h *ShipmentHandler) GenerateInvoice(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID tidak valid", http.StatusBadRequest)
		return
	}

	// 1. Ambil data terperinci dari database
	s, err := h.usecase.GetShipmentByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Data tidak ditemukan: "+err.Error(), http.StatusNotFound)
		return
	}

	// 2. Inisialisasi dokumen PDF (Kertas A4, ukuran Milimeter)
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// ---- HEADER INVOICE (SESUAI GAMBAR BARU) ----
	// Simpan titik Y awal agar logo dan alamat sejajar secara vertikal
	headerY := pdf.GetY()

	// Logo SMRT di sebelah kiri (X: 10, Y: headerY + 2)
	pdf.ImageOptions("logos.png", 10, headerY+2, 55, 0, false, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}, 0, "")
	if pdf.Err() {
		http.Error(w, "Gagal memproses gambar logo: "+pdf.Error().Error(), http.StatusInternalServerError)
		return
	}

	// Blok Alamat Perusahaan di sebelah kanan
	// Geser X ke 110 (lebar area teks alamat 90mm, sehingga pas mentok kanan di koordinat 200)
	pdf.SetXY(110, headerY)
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(0, 0, 0)

	// Teks alamat dicetak baris demi baris menggunakan MultiCell dengan perataan kanan ("R")
	alamatPT := "Jalan Medan Blok C 03-04 KBN SBU Kawasan\nMarunda Kel. Cilincing Kec. Cilincing Kota Adm.\nJakarta Utara Prov. DKI Jakarta, 14120\nTel +62-21-22419139\nTel +62-21-22419167"
	pdf.MultiCell(90, 4.5, alamatPT, "0", "R", false)

	// Tautan website dengan warna biru muda (Cyan-ish) persis seperti di gambar
	pdf.SetX(110)
	pdf.SetTextColor(0, 162, 232)
	pdf.CellFormat(90, 5, "www.smartidlogistics.com", "0", 1, "R", false, 0, "")
	pdf.SetTextColor(0, 0, 0) // Reset warna teks ke hitam

	// Garis pembatas horizontal biru tipis di bawah header
	pdf.SetDrawColor(0, 123, 255)
	pdf.SetLineWidth(0.4)
	pdf.Line(10, pdf.GetY()+2, 200, pdf.GetY()+2)
	pdf.SetDrawColor(0, 0, 0) // Reset warna border ke default
	pdf.SetLineWidth(0.2)     // Reset ketebalan border ke default
	pdf.Ln(8)

	// ---- INFO INVOICE ----
	pdf.SetFont("Courier", "B", 16)
	pdf.SetTextColor(0, 86, 179)
	pdf.CellFormat(0, 10, "QUOTATION", "0", 1, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(2)

	// Metadata Dokumen (Sekarang Rapi di Sebelah Kiri)
	pdf.SetFont("Courier", "", 10)

	// Sisi Kiri: Informasi utama dokumen
	pdf.CellFormat(30, 6, "Invoice No", "0", 0, "L", false, 0, "")
	pdf.CellFormat(5, 6, ":", "0", 0, "L", false, 0, "")
	pdf.CellFormat(60, 6, "INV/"+s.CreatedAt.Format("20060102")+"/"+s.ID[:8], "0", 1, "L", false, 0, "")

	pdf.CellFormat(30, 6, "Date", "0", 0, "L", false, 0, "")
	pdf.CellFormat(5, 6, ":", "0", 0, "L", false, 0, "")
	pdf.CellFormat(60, 6, s.CreatedAt.Format("02 Jan 2006"), "0", 1, "L", false, 0, "")
	pdf.Ln(5)

	// ---- TABEL RINCIAN BIAYA (8 KOLOM - TANPA KOLOM TOTAL) ----
	pdf.SetFont("Courier", "B", 9)
	pdf.SetFillColor(0, 123, 255)   // WARNA BIRU UTAMA
	pdf.SetTextColor(255, 255, 255) // TEKS PUTIH

	// Distribusi ulang lebar kolom setelah kolom 'Total' dihapus (Total pas 190mm)
	wItem := 30.0
	wFrom := 25.0
	wTo := 40.0
	wQty := 12.0
	wRate := 25.0
	wAmount := 25.0
	wVat := 18.0
	wRemark := 15.0

	// Print Header Tabel Berwarna Biru
	pdf.CellFormat(wItem, 8, "Item", "1", 0, "C", true, 0, "")
	pdf.CellFormat(wFrom, 8, "From", "1", 0, "C", true, 0, "")
	pdf.CellFormat(wTo, 8, "To", "1", 0, "C", true, 0, "")
	pdf.CellFormat(wQty, 8, "QTY", "1", 0, "C", true, 0, "")
	pdf.CellFormat(wRate, 8, "Rate", "1", 0, "C", true, 0, "")
	pdf.CellFormat(wAmount, 8, "Amount", "1", 0, "C", true, 0, "")
	pdf.CellFormat(wVat, 8, "VAT (11%)", "1", 0, "C", true, 0, "")
	pdf.CellFormat(wRemark, 8, "Remark", "1", 1, "C", true, 0, "")

	// Balikkan teks ke Hitam untuk isi data
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Courier", "", 8)

	// Hitung VAT 11% & Grand Total
	vatAmount := (s.Amount * 11) / 100
	grandTotal := s.Amount + vatAmount

	// Ambil koordinat Y awal sebelum MultiCell menggambar
	startY := pdf.GetY()

	// Kolom 1: Item
	pdf.SetXY(10, startY)
	pdf.MultiCell(wItem, 5, s.ItemDescription, "0", "L", false)
	h1 := pdf.GetY() - startY

	// Kolom 2: From
	pdf.SetXY(10+wItem, startY)
	pdf.MultiCell(wFrom, 5, s.Origin, "0", "C", false)
	h2 := pdf.GetY() - startY

	// Kolom 3: To (Alamat Panjang)
	pdf.SetXY(10+wItem+wFrom, startY)
	pdf.MultiCell(wTo, 4, s.Destination, "0", "L", false)
	h3 := pdf.GetY() - startY

	// Kolom 4: QTY
	pdf.SetXY(10+wItem+wFrom+wTo, startY)
	pdf.MultiCell(wQty, 5, strconv.Itoa(s.Qty), "0", "C", false)
	h4 := pdf.GetY() - startY

	// Kolom 5: Rate
	pdf.SetXY(10+wItem+wFrom+wTo+wQty, startY)
	pdf.MultiCell(wRate, 5, formatRupiah(s.Rate), "0", "R", false)
	h5 := pdf.GetY() - startY

	// Kolom 6: Amount
	pdf.SetXY(10+wItem+wFrom+wTo+wQty+wRate, startY)
	pdf.MultiCell(wAmount, 5, formatRupiah(s.Amount), "0", "R", false)
	h6 := pdf.GetY() - startY

	// Kolom 7: VAT (11%)
	pdf.SetXY(10+wItem+wFrom+wTo+wQty+wRate+wAmount, startY)
	pdf.MultiCell(wVat, 5, formatRupiah(vatAmount), "0", "R", false)
	h7 := pdf.GetY() - startY

	// Kolom 8: Remark
	pdf.SetXY(10+wItem+wFrom+wTo+wQty+wRate+wAmount+wVat, startY)
	pdf.MultiCell(wRemark, 4, s.Remark, "0", "L", false)
	h9 := pdf.GetY() - startY

	// Cari baris tertinggi agar tinggi border kotak dinamis seimbang
	maxHeight := h1
	heights := []float64{h2, h3, h4, h5, h6, h7, h9}
	for _, h := range heights {
		if h > maxHeight {
			maxHeight = h
		}
	}
	maxHeight += 3 // Padding bawah biar rapi

	// Gambar garis border pembatas luar (Grid Box)
	currentX := 10.0
	widths := []float64{wItem, wFrom, wTo, wQty, wRate, wAmount, wVat, wRemark}
	for _, w := range widths {
		pdf.Rect(currentX, startY, w, maxHeight, "D")
		currentX += w
	}

	// Set posisi Y ke paling bawah tabel setelah membuat grid box
	pdf.SetY(startY + maxHeight)

	// ---- GRAND TOTAL ----
	pdf.SetFont("Courier", "B", 10)
	// Lebar total penampung text kiri disesuaikan (190mm - 35mm = 155mm)
	pdf.CellFormat(157, 8, "GRAND TOTAL (IDR)  ", "1", 0, "R", false, 0, "")

	// Tampilkan total akumulasi akhir di kolom terakhir selebar 35mm
	pdf.SetTextColor(0, 86, 179)
	pdf.CellFormat(33, 8, formatRupiah(grandTotal), "1", 1, "R", false, 0, "")
	pdf.SetTextColor(0, 0, 0) // Reset kembali ke hitam

	pdf.Ln(4)

	// ---- AREA TANDA TANGAN + DIGITAL SIGNATURE ----
	pdf.Ln(10)
	pdf.SetFont("Courier", "", 10)

	pdf.CellFormat(130, 5, "", "0", 0, "L", false, 0, "")
	pdf.CellFormat(60, 5, "Jakarta, "+s.CreatedAt.Format("02 Jan 2006"), "0", 1, "C", false, 0, "")

	// --- PROSES INSERT GAMBAR TANDA TANGAN ---
	sigY := pdf.GetY() + 2
	pdf.ImageOptions("tertanda.jpg", 158, sigY, 30, 0, false, gofpdf.ImageOptions{ImageType: "JPEG", ReadDpi: false}, 0, "")

	if pdf.Err() {
		http.Error(w, "Gagal memproses gambar tanda tangan: "+pdf.Error().Error(), http.StatusInternalServerError)
		return
	}

	pdf.Ln(28)

	pdf.SetFont("Courier", "B", 10)
	pdf.CellFormat(130, 5, "", "0", 0, "L", false, 0, "")
	pdf.CellFormat(60, 5, "( Wiwi Wiliyani )", "0", 1, "C", false, 0, "")

	// 3. Output stream PDF langsung ke browser tab baru
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline; filename=Invoice_"+s.ID[:8]+".pdf")
	pdf.Output(w)
}
func (h *ShipmentHandler) DeleteShipment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id != "" {
		h.usecase.DeleteShipment(r.Context(), id)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
