package util

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codeberg.org/go-pdf/fpdf"

	"github.com/ridwanmuh3/simba-server/internal/model"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()),
)

func stringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func GenerateRandomString(length int) string {
	return stringWithCharset(length, charset)
}

func DeleteFile(filePath string) error {
	cleanPath := strings.TrimPrefix(filePath, "/")
	imagePath := filepath.Join(".", cleanPath)

	if err := os.Remove(imagePath); err != nil {
		return err
	}

	return nil
}

func GenerateTemplateInvoicePDF(data *model.InvoiceData) (*bytes.Buffer, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// --- 1. LOGO PERUSAHAAN (Opsional) ---
	// Menambahkan gambar jika path logo disediakan (pastikan file ada di direktori)
	if data.LogoPath != "" {
		// Parameter: path, x, y, width, height, flow, options, link, linkStr
		pdf.ImageOptions(data.LogoPath, 10, 10, 30, 0, false, fpdf.ImageOptions{ReadDpi: true}, 0, "")
	}

	// --- 2. JUDUL INVOICE ---
	pdf.SetFont("Arial", "B", 18)
	// Kita mulai di Y=20 agar sejajar atau sedikit di bawah logo
	pdf.SetY(20)
	pdf.CellFormat(190, 10, "INVOICE", "0", 1, "C", false, 0, "")
	pdf.Ln(10)

	// --- 3. BLOK ATAS (2 Kolom: Perusahaan & Detail Invoice) ---
	// Menyimpan posisi Y saat ini agar kedua kolom sejajar
	startY := pdf.GetY()

	// Kolom Kiri (Perusahaan)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(100, 5, data.CompanyName, "0", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(100, 5, data.CompanyAddress, "0", 1, "L", false, 0, "")
	pdf.CellFormat(100, 5, data.CompanyContact, "0", 1, "L", false, 0, "")

	// Kolom Kanan (Detail Invoice)
	pdf.SetY(startY) // Kembali ke Y awal
	pdf.SetX(110)    // Geser ke kanan (X=110)
	pdf.CellFormat(80, 5, fmt.Sprintf("No. %s", data.InvoiceNo), "0", 1, "L", false, 0, "")
	pdf.SetX(110)
	pdf.CellFormat(80, 5, fmt.Sprintf("Tanggal: %s", data.Date), "0", 1, "L", false, 0, "")
	pdf.SetX(110)
	pdf.CellFormat(80, 5, fmt.Sprintf("PO No. %s", data.PONo), "0", 1, "L", false, 0, "")
	pdf.SetX(110)
	pdf.CellFormat(80, 5, fmt.Sprintf("Quo No. %s", data.QuoNo), "0", 1, "L", false, 0, "")
	pdf.Ln(10)

	// --- 4. DITUJUKAN KEPADA ---
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(190, 5, "Ditujukan Kepada:", "0", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(190, 5, data.ReceiverName, "0", 1, "L", false, 0, "")
	pdf.CellFormat(190, 5, data.ReceiverAddress, "0", 1, "L", false, 0, "")
	pdf.Ln(10)

	// --- 5. TABEL HEADER (Warna Biru) ---
	pdf.SetFont("Arial", "B", 10)
	// SetFillColor menggunakan format RGB (Ini adalah warna biru muda mirip di gambar)
	pdf.SetFillColor(140, 180, 240)

	// Parameter terakhir (true) mengaktifkan FillColor
	pdf.CellFormat(15, 8, "No.", "1", 0, "C", true, 0, "")
	pdf.CellFormat(65, 8, "Nama Bahan", "1", 0, "C", true, 0, "")
	pdf.CellFormat(20, 8, "Jumlah", "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 8, "Satuan", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "Harga Satuan", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 8, "Total", "1", 1, "C", true, 0, "")

	// --- 6. ISI TABEL ---
	pdf.SetFont("Arial", "", 10)
	for i, item := range data.Items {
		// Parameter terakhir (false) mematikan FillColor agar baris isi berwarna putih
		pdf.CellFormat(15, 8, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
		pdf.CellFormat(65, 8, item.Item.Name, "1", 0, "L", false, 0, "")
		pdf.CellFormat(20, 8, fmt.Sprintf("%d", item.Item.Stock), "1", 0, "C", false, 0, "")
		pdf.CellFormat(25, 8, item.Item.MeasureUnit, "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 8, fmt.Sprintf("Rp %.0f", item.UnitPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(35, 8, fmt.Sprintf("Rp %.0f", item.TotalPrice), "1", 1, "R", false, 0, "")
	}

	// --- 7. TOTAL & KETERANGAN ---
	pdf.Ln(5)

	// Total (Rata Kanan)
	pdf.SetX(125) // Menggeser kursor ke arah kolom total
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(30, 8, "Total:", "0", 0, "R", false, 0, "")
	pdf.CellFormat(35, 8, fmt.Sprintf("Rp %.0f", data.GrandTotal), "0", 1, "R", false, 0, "")
	pdf.Ln(10)

	// Keterangan (Kiri)
	pdf.CellFormat(190, 5, "Keterangan:", "0", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.MultiCell(100, 5, data.Keterangan, "0", "L", false)
	pdf.Ln(15)

	// --- 8. TANDA TANGAN (Kanan Bawah) ---
	pdf.SetX(130)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(60, 5, fmt.Sprintf("(%s)", data.Penanggungjawab), "0", 1, "C", false, 0, "")
	pdf.SetX(130)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(60, 5, data.Jabatan, "0", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}

	return &buf, nil
}
