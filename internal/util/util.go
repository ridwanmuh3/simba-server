package util

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"codeberg.org/go-pdf/fpdf"

	"github.com/ridwanmuh3/simba-server/internal/model"
)

// HashToken returns hex-encoded SHA-256 of the plaintext token. Used so
// DB-side storage is not a bearer token equivalent: a DB leak alone cannot
// impersonate a user without also obtaining the plaintext cookie value.
func HashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateRandomString(length int) string {
	b := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))
	for i := range b {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			panic(fmt.Errorf("failed to generate random number: %w", err))
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

func DeleteFile(filePath string) error {
	cleanPath := strings.TrimPrefix(filePath, "/")
	imagePath := filepath.Join(".", cleanPath)

	absPath, err := filepath.Abs(imagePath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	uploadsDir := filepath.Join(cwd, "uploads")
	if !strings.HasPrefix(absPath, uploadsDir+string(os.PathSeparator)) {
		return fmt.Errorf("forbidden: path outside uploads directory")
	}

	return os.Remove(absPath)
}

// sanitizeLatin1 maps runes outside Windows-1252 (fpdf default) to '?'
// and strips control characters to prevent PDF stream injection.
// Preserves '\n' so MultiCell renders user line breaks; strips other controls.
func sanitizeLatin1(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Map(func(r rune) rune {
		if r == '\n' {
			return r
		}
		if r == '\t' {
			return ' '
		}
		if r < 0x20 {
			return -1
		}
		if r > 0xFF || !unicode.IsPrint(r) {
			return '?'
		}
		return r
	}, s)
}

// formatRupiah returns Indonesian-locale currency: "Rp 1.234.567".
func formatRupiah(v float64) string {
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	intPart := fmt.Sprintf("%.0f", v)
	n := len(intPart)
	if n <= 3 {
		return fmt.Sprintf("Rp %s%s", sign, intPart)
	}
	var b strings.Builder
	first := n % 3
	if first > 0 {
		b.WriteString(intPart[:first])
		if n > first {
			b.WriteByte('.')
		}
	}
	for i := first; i < n; i += 3 {
		b.WriteString(intPart[i : i+3])
		if i+3 < n {
			b.WriteByte('.')
		}
	}
	return fmt.Sprintf("Rp %s%s", sign, b.String())
}

func GenerateTemplateInvoicePDF(data *model.InvoiceData) (*bytes.Buffer, error) {
	// A5 Landscape (210mm x 148mm)
	pdf := fpdf.New("L", "mm", "A5", "")

	// Reduced top margin from 10 to 5 to pull everything up
	pdf.SetMargins(12, 5, 12)
	pdf.SetAutoPageBreak(true, 5)

	// FIXED PAGINATION LOGIC
	pdf.SetFooterFunc(func() {
		pdf.SetY(-8) // Tighter to the bottom edge
		pdf.SetFont("Arial", "I", 7)
		pdf.CellFormat(0, 10, fmt.Sprintf("Halaman %d", pdf.PageNo()), "", 0, "R", false, 0, "")
	})

	pdf.AddPage()

	invoiceTitle := "Invoice Bahan Keluar"
	if strings.EqualFold(data.StockType, "IN") {
		invoiceTitle = "Invoice Bahan Masuk"
	}

	// 1. Header
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(95, 5, sanitizeLatin1(data.CompanyName), "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(95, 5, invoiceTitle, "", 1, "R", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(190, 4, sanitizeLatin1(data.CompanyAddress), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 4, sanitizeLatin1(data.CompanyContact), "", 1, "L", false, 0, "")
	pdf.Ln(1) // Reduced from 2

	// 2. Metadata Block
	pdf.SetLineWidth(0.3)
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(1)
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(30, 4, "Tanggal", "", 0, "L", false, 0, "")
	if data.PONo != "" {
		pdf.CellFormat(40, 4, "PO No.", "", 0, "L", false, 0, "")
	}
	pdf.SetX(140)
	pdf.CellFormat(60, 4, "No. Nota", "", 1, "R", false, 0, "")
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(30, 4, sanitizeLatin1(data.Date), "", 0, "L", false, 0, "")
	if data.PONo != "" {
		pdf.CellFormat(40, 4, sanitizeLatin1(data.PONo), "", 0, "L", false, 0, "")
	}
	pdf.SetX(140)
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(60, 4, sanitizeLatin1(data.InvoiceNo), "", 1, "R", false, 0, "")
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(2) // Reduced from 4

	// 3. Customer Info & Total
	startY := pdf.GetY()
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(95, 4, sanitizeLatin1(data.ReceiverName), "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.MultiCell(90, 4, sanitizeLatin1(data.ReceiverAddress), "", "L", false)

	pdf.SetY(startY)
	pdf.SetX(110)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(30, 8, "TOTAL Rp.", "", 0, "R", false, 0, "") // Height reduced from 10 to 8
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(60, 8, formatRupiah(data.GrandTotal), "", 1, "R", false, 0, "") // Height reduced
	pdf.Ln(1)                                                                      // Reduced from 2

	// 4. Table Header & Items
	drawTableHeader := func() {
		pdf.SetFont("Arial", "B", 9)
		pdf.SetFillColor(220, 220, 220)
		// Row heights reduced from 6 to 5
		pdf.CellFormat(10, 5, "No", "1", 0, "C", true, 0, "")
		pdf.CellFormat(65, 5, "Deskripsi", "1", 0, "C", true, 0, "")
		pdf.CellFormat(15, 5, "Qty", "1", 0, "C", true, 0, "")
		pdf.CellFormat(20, 5, "Unit", "1", 0, "C", true, 0, "")
		pdf.CellFormat(30, 5, "Harga", "1", 0, "C", true, 0, "")
		pdf.CellFormat(15, 5, "Disc", "1", 0, "C", true, 0, "")
		pdf.CellFormat(35, 5, "Subtotal", "1", 1, "C", true, 0, "")
		pdf.SetFont("Arial", "", 9)
	}

	drawTableHeader()

	// Expanded Y-limit due to compacted header/rows
	const itemsMaxY = 115.0

	for i, item := range data.Items {
		if pdf.GetY() > itemsMaxY {
			pdf.AddPage()
			drawTableHeader()
		}
		// Row heights reduced from 6 to 5
		pdf.CellFormat(10, 5, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
		pdf.CellFormat(65, 5, sanitizeLatin1(item.Item.Name), "1", 0, "L", false, 0, "")
		pdf.CellFormat(15, 5, fmt.Sprintf("%g", item.Amount), "1", 0, "C", false, 0, "")
		pdf.CellFormat(20, 5, sanitizeLatin1(item.Item.MeasureUnit), "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 5, formatRupiah(item.UnitPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(15, 5, "0", "1", 0, "C", false, 0, "")
		pdf.CellFormat(35, 5, formatRupiah(item.TotalPrice), "1", 1, "R", false, 0, "")
	}

	pdf.Ln(1) // Reduced from 2
	finalY := pdf.GetY()

	// Keterangan
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(90, 4, "Keterangan:", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.MultiCell(90, 4, sanitizeLatin1(data.Keterangan), "", "L", false)
	keteranganEndY := pdf.GetY()

	// Nomor Rekening
	if strings.TrimSpace(data.BankAccount) != "" {
		pdf.SetY(keteranganEndY + 1)
		pdf.SetX(10)
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(25, 4, "No. Rekening:", "", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 8)
		pdf.CellFormat(90, 4, sanitizeLatin1(data.BankAccount), "", 1, "L", false, 0, "")
		keteranganEndY = pdf.GetY()
	}

	// Summary Recap
	pdf.SetY(finalY)
	var subtotal float64
	for _, it := range data.Items {
		subtotal = Round2(subtotal + it.TotalPrice)
	}
	summaryLabels := []string{"Subtotal Rp.", "Diskon Rp.", "Pajak Rp."}
	summaryValues := []string{formatRupiah(subtotal), "0", "0"}

	for idx, label := range summaryLabels {
		pdf.SetX(125)
		pdf.CellFormat(40, 4, label, "", 0, "R", false, 0, "") // Height reduced from 4.5 to 4
		pdf.CellFormat(35, 4, summaryValues[idx], "", 1, "R", false, 0, "")
	}
	summaryEndY := pdf.GetY()

	separatorY := summaryEndY
	if keteranganEndY > separatorY {
		separatorY = keteranganEndY
	}

	pdf.SetY(separatorY + 1)
	pdf.SetLineWidth(0.3)
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(1)

	// Signature Block
	pdf.SetY(pdf.GetY() + 1) // Reduced from 2
	pdf.SetX(150)
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(60, 4, "Penanggung Jawab,", "", 1, "C", false, 0, "")

	pdf.Ln(14) // Space for signature

	pdf.SetX(150)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(60, 4, fmt.Sprintf("( %s )", sanitizeLatin1(data.Penanggungjawab)), "", 1, "C", false, 0, "")
	pdf.SetX(150)
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(60, 4, sanitizeLatin1(data.Jabatan), "", 1, "C", false, 0, "")

	// Output buffer
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return &buf, nil
}

// Round2 rounds money-ish values to 2 decimal places to avoid float drift.
func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// Round4 rounds quantity-ish values to 4 decimal places for decimal-stock precision.
func Round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func GetTotalPrice(qty, price float64) float64 {
	return Round2(qty * price)
}

func FormatDateStringID(dateStr string) string {
	inputLayout := "2006-01-02 15:04:05"

	t, err := time.Parse(inputLayout, dateStr)
	if err != nil {
		return dateStr
	}

	targetLayout := "02 January 2006 15:04:05"
	englishDateStr := t.Format(targetLayout)

	replacer := strings.NewReplacer(
		"January", "Januari",
		"February", "Februari",
		"March", "Maret",
		"May", "Mei",
		"June", "Juni",
		"July", "Juli",
		"August", "Agustus",
		"October", "Oktober",
		"December", "Desember",
	)

	return replacer.Replace(englishDateStr)
}
