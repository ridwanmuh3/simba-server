package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SettingService struct {
	db          *gorm.DB
	settingRepo SettingRepository
	log         *zap.SugaredLogger
	validate    *validator.Validate
}

type SettingRepository interface {
	GetByKey(db *gorm.DB, key string) (*entity.AppSetting, error)
	Save(db *gorm.DB, setting *entity.AppSetting) error
	IncrementSequence(db *gorm.DB, key string) (int, error)
}

var _ SettingRepository = (*repository.SettingRepository)(nil)

func NewSettingService(db *gorm.DB, settingRepo SettingRepository, log *zap.SugaredLogger, validate *validator.Validate) *SettingService {
	return &SettingService{
		db:          db,
		settingRepo: settingRepo,
		log:         log,
		validate:    validate,
	}
}

const (
	KeyCompanyName      = "company_name"
	KeyCompanyAddress   = "company_address"
	KeyCompanyContact   = "company_contact"
	KeyBankAccount      = "bank_account"
	KeyPenanggungjawab  = "penanggungjawab"
	KeyJabatan          = "jabatan"
	KeySeqInvoice       = "seq_invoice"
	KeySeqQuotation     = "seq_quotation"
)

const DefaultBankAccount = "BNI 2048441550 A.N Koperasi Konsumen Dewa Makmur Multi Sejahtera"

func parseDocumentSequence(value, prefix string) (int, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, false
	}

	parts := strings.Split(trimmed, "-")
	if len(parts) < 2 {
		return 0, false
	}

	if !strings.EqualFold(parts[0], prefix) {
		return 0, false
	}

	seq, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, false
	}

	return seq, true
}

func (s *SettingService) getDocumentSequenceMax(ctx context.Context) (int, int, error) {
	db := s.db.WithContext(ctx)

	var invoices []entity.Invoice
	if err := db.Model(new(entity.Invoice)).Select("invoice_number, quo_number").Find(&invoices).Error; err != nil {
		s.log.Errorf("failed to scan invoice history for sequence: %v", err)
		return 0, 0, exception.InternalServerError
	}

	maxInvoice := 0
	maxQuotation := 0
	for _, invoice := range invoices {
		if seq, ok := parseDocumentSequence(invoice.InvoiceNumber, "INV"); ok && seq > maxInvoice {
			maxInvoice = seq
		}
		if seq, ok := parseDocumentSequence(invoice.QuoNumber, "QUO"); ok && seq > maxQuotation {
			maxQuotation = seq
		}
	}

	return maxInvoice, maxQuotation, nil
}

func (s *SettingService) GetCompanyProfile(ctx context.Context) (*model.CompanyProfileResponse, error) {
	db := s.db.WithContext(ctx)

	name, _ := s.settingRepo.GetByKey(db, KeyCompanyName)
	address, _ := s.settingRepo.GetByKey(db, KeyCompanyAddress)
	contact, _ := s.settingRepo.GetByKey(db, KeyCompanyContact)
	bank, _ := s.settingRepo.GetByKey(db, KeyBankAccount)
	pj, _ := s.settingRepo.GetByKey(db, KeyPenanggungjawab)
	jabatan, _ := s.settingRepo.GetByKey(db, KeyJabatan)

	resp := &model.CompanyProfileResponse{
		BankAccount: DefaultBankAccount,
	}
	if name != nil {
		resp.CompanyName = name.Value
	}
	if address != nil {
		resp.CompanyAddress = address.Value
	}
	if contact != nil {
		resp.CompanyContact = contact.Value
	}
	if bank != nil && strings.TrimSpace(bank.Value) != "" {
		resp.BankAccount = bank.Value
	}
	if pj != nil {
		resp.Penanggungjawab = pj.Value
	}
	if jabatan != nil {
		resp.Jabatan = jabatan.Value
	}

	return resp, nil
}

func (s *SettingService) UpdateCompanyProfile(ctx context.Context, req *model.CompanyProfileRequest) error {
	db := s.db.WithContext(ctx)

	if err := s.validate.Struct(req); err != nil {
		s.log.Warnf("Invalid request body: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if err := s.settingRepo.Save(db, &entity.AppSetting{Key: KeyCompanyName, Value: req.CompanyName}); err != nil {
		return err
	}
	if err := s.settingRepo.Save(db, &entity.AppSetting{Key: KeyCompanyAddress, Value: req.CompanyAddress}); err != nil {
		return err
	}
	if err := s.settingRepo.Save(db, &entity.AppSetting{Key: KeyCompanyContact, Value: req.CompanyContact}); err != nil {
		return err
	}
	if err := s.settingRepo.Save(db, &entity.AppSetting{Key: KeyBankAccount, Value: req.BankAccount}); err != nil {
		return err
	}
	if err := s.settingRepo.Save(db, &entity.AppSetting{Key: KeyPenanggungjawab, Value: req.Penanggungjawab}); err != nil {
		return err
	}
	if err := s.settingRepo.Save(db, &entity.AppSetting{Key: KeyJabatan, Value: req.Jabatan}); err != nil {
		return err
	}

	return nil
}

func (s *SettingService) GetNextDocumentNumbers(ctx context.Context) (*model.DocumentSequenceResponse, error) {
	invSeq, quoSeq, err := s.getDocumentSequenceMax(ctx)
	if err != nil {
		return nil, err
	}

	invNo := fmt.Sprintf("INV-%03d", invSeq+1)
	quoNo := fmt.Sprintf("QUO-%03d", quoSeq+1)

	return &model.DocumentSequenceResponse{
		NextInvoiceNo:   invNo,
		NextQuotationNo: quoNo,
	}, nil
}

func (s *SettingService) ConsumeDocumentNumbers(ctx context.Context) error {
	db := s.db.WithContext(ctx)

	invSeq, quoSeq, err := s.getDocumentSequenceMax(ctx)
	if err != nil {
		return err
	}

	if err := s.settingRepo.Save(db, &entity.AppSetting{Key: KeySeqInvoice, Value: fmt.Sprintf("%d", invSeq)}); err != nil {
		s.log.Errorf("failed to sync invoice seq: %v", err)
		return exception.InternalServerError
	}

	if err := s.settingRepo.Save(db, &entity.AppSetting{Key: KeySeqQuotation, Value: fmt.Sprintf("%d", quoSeq)}); err != nil {
		s.log.Errorf("failed to sync quo seq: %v", err)
		return exception.InternalServerError
	}

	return nil
}
