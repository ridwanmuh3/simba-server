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

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const MAX_INVOICE_ITEM = 30

// HashToken returns hex-encoded SHA-256 of the plaintext token. Used so
// DB-side storage is not a bearer token equivalent: a DB leak alone cannot
// impersonate a user without also obtaining the plaintext cookie value.
func HashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

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
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(false, 0)

	// =========================================================
	// PAGE CONFIG
	// =========================================================
	pageW, pageH := pdf.GetPageSize()

	marginX := 10.0
	marginTop := 10.0

	pdf.SetMargins(marginX, marginTop, marginX)
	pdf.AddPage()

	left, top, right, bottom := pdf.GetMargins()

	usableWidth := pageW - left - right
	_ = pageH - top - bottom

	pdf.SetLineWidth(0.15)

	// =========================================================
	// TABLE CONFIG
	// =========================================================
	// Total must be equal to usableWidth (190mm)
	colWidths := []float64{
		10, // No
		72, // Nama Barang
		14, // Qty
		18, // Satuan
		36, // Harga
		40, // Jumlah
	}

	headers := []string{
		"No",
		"Nama Barang",
		"Qty",
		"Satuan",
		"Harga",
		"Jumlah",
	}

	var tableWidth float64
	for _, w := range colWidths {
		tableWidth += w
	}

	tableX := left

	// =========================================================
	// LOGO
	// =========================================================
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	logoPath := filepath.Join(
		wd,
		"internal",
		"assets",
		"sppg.png",
	)

	logoWidth := 24.0
	logoHeight := 24.0

	logoX := left
	logoY := top

	if _, err := os.Stat(logoPath); err == nil {
		pdf.Image(
			logoPath,
			logoX,
			logoY,
			logoWidth,
			logoHeight,
			false,
			"",
			0,
			"",
		)
	}

	// =========================================================
	// HEADER
	// =========================================================
	// Jarak logo dan teks dipersempit menjadi 1mm
	headerStartX := left + logoWidth + 1
	headerWidth := usableWidth - logoWidth - 1

	pdf.SetXY(headerStartX, top+4)
	pdf.SetFont("Arial", "B", 14) // Font diperbesar
	pdf.CellFormat(
		headerWidth,
		6,
		"KOPERASI KONSUMEN DEWA MAKMUR MULTI SEJAHTERA",
		"",
		1,
		"C",
		false,
		0,
		"",
	)

	pdf.SetX(headerStartX)
	pdf.SetFont("Arial", "B", 12) // Font diperbesar
	pdf.CellFormat(
		headerWidth,
		6,
		"Kp. Dukuh Desa.Margasari Kec. Ciawi",
		"",
		1,
		"C",
		false,
		0,
		"",
	)

	pdf.SetX(headerStartX)
	pdf.SetFont("Arial", "B", 12) // Font diperbesar
	pdf.CellFormat(
		headerWidth,
		6,
		"INVOICE",
		"",
		1,
		"C",
		false,
		0,
		"",
	)

	pdf.Ln(6)

	// =========================================================
	// RECEIVER & META RIGHT
	// =========================================================
	sectionStartY := pdf.GetY()

	// LEFT: RECEIVER
	receiverWidth := 100.0
	pdf.SetXY(left, sectionStartY)
	pdf.SetFont("Arial", "B", 10) // Font diperbesar
	pdf.CellFormat(receiverWidth, 5.5, "Kepada Yth :", "", 1, "L", false, 0, "")

	pdf.SetX(left)
	pdf.CellFormat(receiverWidth, 5.5, sanitizeLatin1(data.ReceiverName), "", 1, "L", false, 0, "")

	pdf.SetX(left)
	pdf.SetFont("Arial", "", 10) // Font diperbesar
	pdf.CellFormat(receiverWidth, 5.5, "Alamat", "", 1, "L", false, 0, "")

	pdf.SetX(left)
	pdf.MultiCell(receiverWidth, 5.5, sanitizeLatin1(data.ReceiverAddress), "", "L", false)

	// RIGHT: META
	metaWidth := 75.0
	rightMetaX := left + usableWidth - metaWidth
	pdf.SetXY(rightMetaX, sectionStartY)

	labelWidth := 20.0
	colonWidth := 4.0
	valueWidth := metaWidth - labelWidth - colonWidth

	metaRows := [][]string{
		{"Tanggal", data.Date},
		{"No. Invoice", data.InvoiceNo},
		{"Kebutuhan", data.PONo},
	}

	pdf.SetFont("Arial", "", 10) // Font diperbesar
	for _, row := range metaRows {
		pdf.SetX(rightMetaX)
		pdf.CellFormat(labelWidth, 5, row[0], "", 0, "L", false, 0, "")
		pdf.CellFormat(colonWidth, 5, ":", "", 0, "C", false, 0, "")
		pdf.CellFormat(valueWidth, 5, sanitizeLatin1(row[1]), "", 1, "L", false, 0, "")
	}

	pdf.SetY(sectionStartY + 20) // Penyesuaian gap sebelum tabel
	pdf.Ln(6)
	// =========================================================
	// TABLE HEADER
	// =========================================================
	headerRowHeight := 5.5        // Baris ditinggikan untuk font yang lebih besar
	pdf.SetFont("Arial", "B", 10) // Font diperbesar
	pdf.SetX(tableX)

	for i, h := range headers {
		pdf.CellFormat(
			colWidths[i],
			headerRowHeight,
			h,
			"1",
			0,
			"C",
			false,
			0,
			"",
		)
	}
	pdf.Ln(-1)

	// =========================================================
	// TABLE BODY
	// =========================================================
	bodyRowHeight := 5.5         // Baris ditinggikan
	pdf.SetFont("Arial", "", 10) // Font diperbesar

	maxRows := 30
	var itemsRows []*model.StockResponse

	for i := range data.Items {
		itemsRows = append(itemsRows, &data.Items[i])
	}

	if len(itemsRows) > maxRows {
		itemsRows = itemsRows[:maxRows]
	}

	for len(itemsRows) < maxRows {
		itemsRows = append(itemsRows, nil)
	}

	for i, item := range itemsRows {
		pdf.SetX(tableX)

		values := []string{"", "", "", "", "", ""}

		if item != nil {
			values = []string{
				fmt.Sprintf("%d", i+1),
				sanitizeLatin1(item.Item.Name),
				fmt.Sprintf("%g", item.Amount),
				sanitizeLatin1(item.Item.MeasureUnit),
				formatRupiah(item.UnitPrice),
				formatRupiah(item.TotalPrice),
			}
		} else {
			values[0] = fmt.Sprintf("%d", i+1)
		}

		aligns := []string{"C", "L", "C", "C", "R", "R"}

		for j := range values {
			pdf.CellFormat(
				colWidths[j],
				bodyRowHeight,
				values[j],
				"1",
				0,
				aligns[j],
				false,
				0,
				"",
			)
		}
		pdf.Ln(-1)
	}

	// =========================================================
	// SUMMARY / SUB TOTAL
	// =========================================================
	var subtotal float64
	for _, it := range data.Items {
		subtotal += it.TotalPrice
	}

	subtotalX := tableX + colWidths[0] + colWidths[1] + colWidths[2] + colWidths[3]

	pdf.SetX(subtotalX)
	pdf.SetFont("Arial", "B", 10) // Font diperbesar
	pdf.CellFormat(colWidths[4], 5.5, "Sub Total", "1", 0, "L", false, 0, "")
	pdf.CellFormat(colWidths[5], 5.5, formatRupiah(subtotal), "1", 1, "R", false, 0, "")

	pdf.Ln(4)

	// =========================================================
	// FOOTER & SIGNATURES
	// =========================================================
	footerStartY := pdf.GetY()

	// FOOTER (Left)
	footerLeftWidth := 120.0
	pdf.SetXY(left, footerStartY-8)
	pdf.SetFont("Arial", "", 10) // Font diperbesar

	footerLineHeight := 5.0
	pdf.CellFormat(footerLeftWidth, footerLineHeight, "Pembayaran Invoice:", "", 1, "L", false, 0, "")

	pdf.SetX(left)
	pdf.CellFormat(footerLeftWidth, footerLineHeight, "Bank BNI", "", 1, "L", false, 0, "")

	pdf.SetX(left)
	pdf.CellFormat(footerLeftWidth, footerLineHeight, fmt.Sprintf("No. Rekening : %s", sanitizeLatin1(data.BankAccount)), "", 1, "L", false, 0, "")

	pdf.SetX(left)
	pdf.CellFormat(footerLeftWidth, footerLineHeight, "Atas Nama : Koperasi Konsumen Dewa Makmur Multi Sejahtera", "", 1, "L", false, 0, "")

	pdf.Ln(2)

	// SIGNATURES
	signY := pdf.GetY()

	leftSignWidth := 50.0
	rightSignWidth := 50.0

	leftSignX := left + 10
	rightSignX := left + usableWidth - rightSignWidth - 10

	signGapHeight := 15.0

	pdf.SetFont("Arial", "", 10) // Font diperbesar

	// LEFT SIGN
	pdf.SetXY(leftSignX, signY)
	pdf.CellFormat(leftSignWidth, 5, "Penerima", "", 1, "C", false, 0, "")

	pdf.Ln(signGapHeight)
	pdf.SetX(leftSignX)
	pdf.CellFormat(leftSignWidth, 5, "(                     )", "", 0, "C", false, 0, "")

	// RIGHT SIGN
	pdf.SetY(signY)
	pdf.SetX(rightSignX)
	pdf.CellFormat(rightSignWidth, 5, "Hormat Kami", "", 1, "C", false, 0, "")

	pdf.Ln(signGapHeight)
	pdf.SetX(rightSignX)

	// Print Penanggungjawab under Hormat Kami
	pdf.CellFormat(rightSignWidth, 5, sanitizeLatin1(data.Penanggungjawab), "", 1, "C", false, 0, "")

	// =========================================================
	// OUTPUT
	// =========================================================
	var buf bytes.Buffer

	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}

	return &buf, nil
}

func GenerateTemplateInvoicePDFV1(data *model.InvoiceData) (*bytes.Buffer, error) {
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
