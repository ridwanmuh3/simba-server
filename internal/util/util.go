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
const MAX_INVOICE_ITEM = 6

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
	pdf := fpdf.New("L", "mm", "A5", "")
	pdf.SetMargins(10, 17, 10)

	// ❗ manual pagination
	pdf.SetAutoPageBreak(false, 0)

	pdf.AddPage()

	// =======================
	// PAGE CONFIG
	// =======================
	pageW, pageH := pdf.GetPageSize()
	left, _, right, bottom := pdf.GetMargins()
	usableWidth := pageW - left - right

	// =======================
	// COLUMN (RESPONSIVE)
	// =======================
	colRatio := []float64{10, 65, 15, 20, 30, 35}

	scaleColumns := func(total float64, ratios []float64) []float64 {
		var sum float64
		for _, r := range ratios {
			sum += r
		}
		out := make([]float64, len(ratios))
		for i, r := range ratios {
			out[i] = (r / sum) * total
		}
		return out
	}

	colWidths := scaleColumns(usableWidth, colRatio)

	// =======================
	// HEADER
	// =======================
	title := "Invoice Bahan Keluar"
	if strings.EqualFold(data.StockType, "IN") {
		title = "Invoice Bahan Masuk"
	}

	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(usableWidth/2, 5, sanitizeLatin1(data.CompanyName), "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(usableWidth/2, 5, title, "", 1, "R", false, 0, "")

	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(usableWidth, 4, sanitizeLatin1(data.CompanyAddress), "", 1, "L", false, 0, "")
	pdf.CellFormat(usableWidth, 4, sanitizeLatin1(data.CompanyContact), "", 1, "L", false, 0, "")
	pdf.Ln(1)

	// =======================
	// METADATA
	// =======================
	pdf.Line(left, pdf.GetY(), pageW-right, pdf.GetY())
	pdf.Ln(1)

	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(30, 4, "Tanggal", "", 0, "L", false, 0, "")
	if data.PONo != "" {
		pdf.CellFormat(40, 4, "PO No.", "", 0, "L", false, 0, "")
	}

	pdf.SetX(pageW - right - 60)
	pdf.CellFormat(60, 4, "No. Nota", "", 1, "R", false, 0, "")

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(30, 4, sanitizeLatin1(data.Date), "", 0, "L", false, 0, "")
	if data.PONo != "" {
		pdf.CellFormat(40, 4, sanitizeLatin1(data.PONo), "", 0, "L", false, 0, "")
	}

	pdf.SetX(pageW - right - 60)
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(60, 4, sanitizeLatin1(data.InvoiceNo), "", 1, "R", false, 0, "")

	pdf.Line(left, pdf.GetY(), pageW-right, pdf.GetY())
	pdf.Ln(2)

	// =======================
	// CUSTOMER + TOTAL
	// =======================
	startY := pdf.GetY()

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(usableWidth/2, 4, sanitizeLatin1(data.ReceiverName), "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "", 9)
	pdf.MultiCell(usableWidth/2, 4, sanitizeLatin1(data.ReceiverAddress), "", "L", false)

	pdf.SetY(startY)
	pdf.SetX(pageW - right - 90)

	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(30, 8, "TOTAL Rp.", "", 0, "R", false, 0, "")
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(60, 8, formatRupiah(data.GrandTotal), "", 1, "R", false, 0, "")

	pdf.Ln(1)

	// =======================
	// TABLE HEADER
	// =======================
	drawHeader := func() {
		pdf.SetX(left)
		pdf.SetFont("Arial", "B", 9)
		pdf.SetFillColor(220, 220, 220)

		headers := []string{"No", "Deskripsi", "Qty", "Unit", "Harga", "Subtotal"}

		for i, h := range headers {
			lb := 0
			if i == len(headers)-1 {
				lb = 1
			}
			pdf.CellFormat(colWidths[i], 5, h, "1", lb, "C", true, 0, "")
		}

		pdf.SetFont("Arial", "", 9)
	}

	drawHeader()

	// =======================
	// PAGINATION LIMIT
	// =======================
	footerSpace := 45.0
	itemsMaxY := pageH - bottom - footerSpace

	// =======================
	// PREPARE ROWS
	// =======================
	var rows []*model.StockResponse
	for i := range data.Items {
		rows = append(rows, &data.Items[i])
	}

	if len(rows) < MAX_INVOICE_ITEM {
		for range MAX_INVOICE_ITEM - len(rows) {
			rows = append(rows, nil)
		}
	}

	// =======================
	// RENDER ROWS
	// =======================
	for i, item := range rows {
		if pdf.GetY() > itemsMaxY {
			pdf.AddPage()
			drawHeader()
		}

		pdf.SetX(left)

		if item != nil {
			values := []string{
				fmt.Sprintf("%d", i+1),
				sanitizeLatin1(item.Item.Name),
				fmt.Sprintf("%g", item.Amount),
				sanitizeLatin1(item.Item.MeasureUnit),
				formatRupiah(item.UnitPrice),
				formatRupiah(item.TotalPrice),
			}

			align := []string{"C", "L", "C", "C", "R", "R"}

			for j := range values {
				lb := 0
				if j == len(values)-1 {
					lb = 1
				}
				pdf.CellFormat(colWidths[j], 5, values[j], "1", lb, align[j], false, 0, "")
			}

		} else {
			for j := range colWidths {
				lb := 0
				if j == len(colWidths)-1 {
					lb = 1
				}

				txt := ""
				if j == 0 {
					txt = fmt.Sprintf("%d", i+1)
				}

				pdf.CellFormat(colWidths[j], 5, txt, "1", lb, "C", false, 0, "")
			}
		}
	}

	pdf.Ln(1)
	// =======================
	// FOOTER (SIDE BY SIDE LAYOUT)
	// =======================
	startFooterY := pdf.GetY()

	// =======================
	// LEFT BLOCK (Keterangan + Rekening)
	// =======================
	pdf.SetY(startFooterY)
	pdf.SetX(left)

	leftStartX := left
	leftWidth := usableWidth / 2

	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(leftWidth, 4, "Keterangan:", "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "", 8)
	pdf.SetX(leftStartX)
	pdf.MultiCell(leftWidth, 4, sanitizeLatin1(data.Keterangan), "", "L", false)

	leftEndY := pdf.GetY()

	// No Rekening INLINE (sesuai gambar kamu)
	if strings.TrimSpace(data.BankAccount) != "" {
		pdf.SetX(leftStartX)

		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(30, 4, "No. Rekening:", "", 0, "L", false, 0, "")

		pdf.SetFont("Arial", "", 8)
		pdf.CellFormat(leftWidth-30, 4, sanitizeLatin1(data.BankAccount), "", 1, "L", false, 0, "")

		leftEndY = pdf.GetY()
	}

	// =======================
	// RIGHT BLOCK (SUMMARY)
	// =======================
	pdf.SetY(startFooterY)

	rightStartX := pageW - right - 75

	var subtotal float64
	for _, it := range data.Items {
		subtotal = Round2(subtotal + it.TotalPrice)
	}

	labels := []string{"Subtotal Rp.", "Diskon Rp.", "Pajak Rp."}
	values := []string{formatRupiah(subtotal), "0", "0"}

	pdf.SetFont("Arial", "", 8)

	for i := range labels {
		pdf.SetX(rightStartX)
		pdf.CellFormat(40, 4, labels[i], "", 0, "R", false, 0, "")
		pdf.CellFormat(35, 4, values[i], "", 1, "R", false, 0, "")
	}

	rightEndY := pdf.GetY()

	// =======================
	// SYNC HEIGHT (biar sejajar)
	// =======================
	footerEndY := leftEndY
	if rightEndY > footerEndY {
		footerEndY = rightEndY
	}

	// =======================
	// SEPARATOR LINE
	// =======================
	pdf.SetY(footerEndY + 2)
	pdf.Line(left, pdf.GetY(), pageW-right, pdf.GetY())

	// =======================
	// SIGNATURE (KANAN)
	// =======================
	pdf.Ln(2)

	pdf.SetX(pageW - right - 60)
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(60, 4, "Penanggung Jawab,", "", 1, "C", false, 0, "")

	pdf.Ln(12)

	pdf.SetX(pageW - right - 60)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(60, 4, fmt.Sprintf("( %s )", sanitizeLatin1(data.Penanggungjawab)), "", 1, "C", false, 0, "")

	pdf.SetX(pageW - right - 60)
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(60, 4, sanitizeLatin1(data.Jabatan), "", 1, "C", false, 0, "")
	// =======================
	// OUTPUT
	// =======================
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
