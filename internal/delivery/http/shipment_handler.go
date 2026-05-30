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
	rate, _ := strconv.ParseInt(r.FormValue("rate"), 10, 64)
	buyingPrice, _ := strconv.ParseInt(r.FormValue("buying_price"), 10, 64)

	shipment := &domain.Shipment{
		ItemDescription: r.FormValue("item_description"),
		Origin:          r.FormValue("origin"),
		Destination:     r.FormValue("destination"),
		Qty:             qty,
		Rate:            rate,
		BuyingPrice:     buyingPrice,
		Remark:          r.FormValue("remark"),
	}

	// Buat penampung kosong untuk data file dokumen mentah
	var fileData []byte
	var contentType string

	// 3. Ambil data file dokumen yang di-upload (PDF/Gambar) jika ada
	file, header, err := r.FormFile("document")
	if err == nil {
		defer file.Close()
		// Baca file menjadi pecahan byte data mentah
		data, err := io.ReadAll(file)
		if err == nil {
			fileData = data
			contentType = header.Header.Get("Content-Type")
		}
	}

	// 4. Simpan seluruh data teks ke Postgres + upload file mentah ke MinIO via Usecase
	err = h.usecase.InsertShipment(r.Context(), shipment, fileData, contentType)
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

	// ---- HEADER INVOICE ----
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 10, "SMART TRUCKING LOGISTICS", "0", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 5, "Jl. Logistik Terpadu No. 20, Jakarta", "0", 1, "L", false, 0, "")
	pdf.CellFormat(0, 5, "Email: finance@smartdepo.com | Telp: (021) 889977", "0", 1, "L", false, 0, "")
	
	// Garis pembatas horizontal
	pdf.Line(10, 32, 200, 32)
	pdf.Ln(10)

	// ---- INFO NOTA ----
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, "INVOICE QUOTATION", "0", 1, "C", false, 0, "")
	pdf.Ln(5)

	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(50, 6, "Invoice ID: "+s.ID[:8], "0", 0, "L", false, 0, "") // ambil 8 digit depan UUID
	pdf.CellFormat(0, 6, "Tanggal: "+s.CreatedAt.Format("02 Jan 2006"), "0", 1, "R", false, 0, "")
	pdf.CellFormat(50, 6, "Rute Operasional: "+s.Origin+" Ke "+s.Destination, "0", 1, "L", false, 0, "")
	pdf.Ln(5)

	// ---- TABEL RINCIAN HARGA ----
	// Header Tabel PDF
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(80, 8, "Deskripsi Layanan Truk", "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 8, "QTY", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 8, "Harga Satuan (Rate)", "1", 0, "C", true, 0, "")
	pdf.CellFormat(45, 8, "Total Tagihan", "1", 1, "C", true, 0, "")

	// Isi Data Baris Tabel
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(80, 8, " "+s.ItemDescription, "1", 0, "L", false, 0, "")
	pdf.CellFormat(25, 8, strconv.Itoa(s.Qty), "1", 0, "C", false, 0, "")
	pdf.CellFormat(40, 8, "Rp "+strconv.FormatInt(s.Rate, 10), "1", 0, "R", false, 0, "")
	pdf.CellFormat(45, 8, "Rp "+strconv.FormatInt(s.Amount, 10), "1", 1, "R", false, 0, "")

	// Total Besar di bawah tabel
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(145, 8, "GRAND TOTAL  ", "1", 0, "R", false, 0, "")
	pdf.CellFormat(45, 8, "Rp "+strconv.FormatInt(s.Amount, 10), "1", 1, "R", false, 0, "")
	pdf.Ln(10)

	// Catatan kaki / Remark jika ada
	if s.Remark != "" {
		pdf.SetFont("Arial", "I", 9)
		pdf.CellFormat(0, 5, "Catatan: "+s.Remark, "0", 1, "L", false, 0, "")
	}

	// ---- TANDA TANGAN ----
	pdf.Ln(20)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 5, "Hormat Kami,", "0", 1, "R", false, 0, "")
	pdf.Ln(15)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(0, 5, "( Tim Keuangan SMART ) ", "0", 1, "R", false, 0, "")

	// 3. Set header HTTP dan langsung stream file PDF ke browser tab baru
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