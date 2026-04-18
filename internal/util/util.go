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
func sanitizeLatin1(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 {
			if r == '\n' || r == '\t' {
				return ' '
			}
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
	pdf.SetMargins(10, 10, 10)
	// Reduce auto-break trigger to give more control over footer placement
	pdf.SetAutoPageBreak(true, 10)
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
	pdf.Ln(2)

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
	pdf.CellFormat(30, 5, sanitizeLatin1(data.Date), "", 0, "L", false, 0, "")
	if data.PONo != "" {
		pdf.CellFormat(40, 5, sanitizeLatin1(data.PONo), "", 0, "L", false, 0, "")
	}
	pdf.SetX(140)
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(60, 5, sanitizeLatin1(data.InvoiceNo), "", 1, "R", false, 0, "")
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(4)

	// 3. Customer Info & Total
	startY := pdf.GetY()
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(95, 4, sanitizeLatin1(data.ReceiverName), "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.MultiCell(90, 4, sanitizeLatin1(data.ReceiverAddress), "", "L", false)

	pdf.SetY(startY)
	pdf.SetX(110)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(30, 10, "TOTAL Rp.", "", 0, "R", false, 0, "")
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(60, 10, formatRupiah(data.GrandTotal), "", 1, "R", false, 0, "")
	pdf.Ln(2)

	// 4. Table Header & Items
	drawTableHeader := func() {
		pdf.SetFont("Arial", "B", 9)
		pdf.SetFillColor(220, 220, 220)
		pdf.CellFormat(10, 6, "No", "1", 0, "C", true, 0, "")
		pdf.CellFormat(65, 6, "Deskripsi", "1", 0, "C", true, 0, "")
		pdf.CellFormat(15, 6, "Qty", "1", 0, "C", true, 0, "")
		pdf.CellFormat(20, 6, "Unit", "1", 0, "C", true, 0, "")
		pdf.CellFormat(30, 6, "Harga", "1", 0, "C", true, 0, "")
		pdf.CellFormat(15, 6, "Disc", "1", 0, "C", true, 0, "")
		pdf.CellFormat(35, 6, "Subtotal", "1", 1, "C", true, 0, "")
		pdf.SetFont("Arial", "", 9)
	}

	drawTableHeader()

	for i, item := range data.Items {
		// Safety check: A5 height is 148mm.
		// We trigger page break if Y > 100mm to leave room for the footer.
		if pdf.GetY() > 100 {
			pdf.AddPage()
			drawTableHeader()
		}
		pdf.CellFormat(10, 6, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
		pdf.CellFormat(65, 6, sanitizeLatin1(item.Item.Name), "1", 0, "L", false, 0, "")
		pdf.CellFormat(15, 6, fmt.Sprintf("%g", item.Amount), "1", 0, "C", false, 0, "")
		pdf.CellFormat(20, 6, sanitizeLatin1(item.Item.MeasureUnit), "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 6, formatRupiah(item.UnitPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(15, 6, "0", "1", 0, "C", false, 0, "")
		pdf.CellFormat(35, 6, formatRupiah(item.TotalPrice), "1", 1, "R", false, 0, "")
	}

	// 5. Footer Logic
	// If the table ends too low, force a page break BEFORE drawing the footer
	// to avoid triggering an automatic one mid-footer.
	if pdf.GetY() > 115 {
		pdf.AddPage()
	}

	pdf.Ln(2)
	finalY := pdf.GetY()

	// Keterangan
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(90, 4, "Keterangan:", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.MultiCell(90, 4, sanitizeLatin1(data.Keterangan), "", "L", false)

	// Summary Recap
	pdf.SetY(finalY)
	// Compute subtotal from line items so it exactly matches printed rows.
	var subtotal float64
	for _, it := range data.Items {
		subtotal = Round2(subtotal + it.TotalPrice)
	}
	summaryLabels := []string{"Subtotal Rp.", "Diskon Rp.", "Pajak Rp."}
	summaryValues := []string{formatRupiah(subtotal), "0", "0"}

	for idx, label := range summaryLabels {
		pdf.SetX(125)
		pdf.CellFormat(40, 5, label, "", 0, "R", false, 0, "")
		pdf.CellFormat(35, 5, summaryValues[idx], "", 1, "R", false, 0, "")
	}

	// Signature - Absolute Positioning to avoid page push
	pdf.SetY(pdf.GetY() + 5)
	pdf.SetX(130)
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(60, 4, "Penanggung Jawab,", "", 1, "C", false, 0, "")
	pdf.Ln(10)
	pdf.SetX(130)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(60, 4, fmt.Sprintf("( %s )", sanitizeLatin1(data.Penanggungjawab)), "", 1, "C", false, 0, "")
	pdf.SetX(130)
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(60, 4, sanitizeLatin1(data.Jabatan), "", 1, "C", false, 0, "")

	// Pagination
	pdf.SetY(-10) // Move closer to bottom
	pdf.SetFont("Arial", "I", 7)
	pdf.CellFormat(190, 5, fmt.Sprintf("Halaman %d", pdf.PageNo()), "", 0, "R", false, 0, "")

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
