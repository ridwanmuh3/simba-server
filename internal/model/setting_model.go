package model

type CompanyProfileRequest struct {
	CompanyName       string `json:"company_name" validate:"required"`
	CompanyAddress    string `json:"company_address" validate:"required"`
	CompanyContact    string `json:"company_contact" validate:"required"`
	BankAccount       string `json:"bank_account" validate:"omitempty,max=200"`
	Penanggungjawab   string `json:"penanggungjawab" validate:"omitempty,max=100"`
	Jabatan           string `json:"jabatan" validate:"omitempty,max=100"`
}

type CompanyProfileResponse struct {
	CompanyName     string `json:"company_name"`
	CompanyAddress  string `json:"company_address"`
	CompanyContact  string `json:"company_contact"`
	BankAccount     string `json:"bank_account"`
	Penanggungjawab string `json:"penanggungjawab"`
	Jabatan         string `json:"jabatan"`
}

type DocumentSequenceResponse struct {
	NextInvoiceNo string `json:"next_invoice_no"`
	NextPONumber  string `json:"next_po_no"`
}
