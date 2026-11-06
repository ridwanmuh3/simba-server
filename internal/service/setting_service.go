package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ridwanmuh3/simba-server/internal/entity"
	"github.com/ridwanmuh3/simba-server/internal/exception"
	"github.com/ridwanmuh3/simba-server/internal/model"
	"github.com/ridwanmuh3/simba-server/internal/repository"
)

type SettingService struct {
	db          *gorm.DB
	settingRepo SettingRepository
	log         *zap.SugaredLogger
	validate    *validator.Validate
}

type SettingRepository interface {
	GetByKey(db *gorm.DB, key string, dapurID uint) (*entity.AppSetting, error)
	Save(db *gorm.DB, setting *entity.AppSetting) error
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
	KeyCompanyName     = "company_name"
	KeyCompanyAddress  = "company_address"
	KeyCompanyContact  = "company_contact"
	KeyBankAccount     = "bank_account"
	KeyPenanggungjawab = "penanggungjawab"
	KeyJabatan         = "jabatan"
	KeySeqInvoice      = "seq_invoice"
	KeySeqPO           = "seq_po"
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

func (s *SettingService) getDocumentSequenceMax(ctx context.Context, dapurID uint) (int, int, error) {
	return getDocumentSequenceMaxFrom(s.db.WithContext(ctx), dapurID)
}

func getDocumentSequenceMaxFrom(db *gorm.DB, dapurID uint) (int, int, error) {
	var invoices []entity.Invoice
	if err := db.Model(new(entity.Invoice)).
		Select("invoice_number, po_number").
		Where("dapur_id = ?", dapurID).
		Find(&invoices).Error; err != nil {
		return 0, 0, exception.InternalServerError
	}

	maxInvoice := 0
	maxPo := 0
	for _, invoice := range invoices {
		if seq, ok := parseDocumentSequence(invoice.InvoiceNumber, "INV"); ok && seq > maxInvoice {
			maxInvoice = seq
		}
		if seq, ok := parseDocumentSequence(invoice.PONumber, "PO"); ok && seq > maxPo {
			maxPo = seq
		}
	}

	return maxInvoice, maxPo, nil
}

func readSeqFrom(db *gorm.DB, key string, dapurID uint) (int, bool) {
	var setting entity.AppSetting
	if err := db.Where("key = ? AND dapur_id = ?", key, dapurID).First(&setting).Error; err != nil {
		return 0, false
	}
	v, err := strconv.Atoi(strings.TrimSpace(setting.Value))
	if err != nil {
		return 0, false
	}
	return v, true
}

func upsertSeqTo(db *gorm.DB, key string, dapurID uint, value int) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}, {Name: "dapur_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&entity.AppSetting{Key: key, Value: fmt.Sprintf("%d", value), DapurID: dapurID}).Error
}

func (s *SettingService) GetCompanyProfile(ctx context.Context, dapurID uint) (*model.CompanyProfileResponse, error) {
	db := s.db.WithContext(ctx)

	name, _ := s.settingRepo.GetByKey(db, KeyCompanyName, dapurID)
	address, _ := s.settingRepo.GetByKey(db, KeyCompanyAddress, dapurID)
	contact, _ := s.settingRepo.GetByKey(db, KeyCompanyContact, dapurID)
	bank, _ := s.settingRepo.GetByKey(db, KeyBankAccount, dapurID)
	pj, _ := s.settingRepo.GetByKey(db, KeyPenanggungjawab, dapurID)
	jabatan, _ := s.settingRepo.GetByKey(db, KeyJabatan, dapurID)

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

func (s *SettingService) UpdateCompanyProfile(ctx context.Context, req *model.CompanyProfileRequest, dapurID uint) error {
	db := s.db.WithContext(ctx)

	if err := s.validate.Struct(req); err != nil {
		s.log.Warnf("Invalid request body: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	saves := []entity.AppSetting{
		{Key: KeyCompanyName, Value: req.CompanyName, DapurID: dapurID},
		{Key: KeyCompanyAddress, Value: req.CompanyAddress, DapurID: dapurID},
		{Key: KeyCompanyContact, Value: req.CompanyContact, DapurID: dapurID},
		{Key: KeyBankAccount, Value: req.BankAccount, DapurID: dapurID},
		{Key: KeyPenanggungjawab, Value: req.Penanggungjawab, DapurID: dapurID},
		{Key: KeyJabatan, Value: req.Jabatan, DapurID: dapurID},
	}

	for i := range saves {
		if err := s.settingRepo.Save(db, &saves[i]); err != nil {
			return err
		}
	}

	return nil
}

func (s *SettingService) GetNextDocumentNumbers(ctx context.Context, dapurID uint) (*model.DocumentSequenceResponse, error) {
	db := s.db.WithContext(ctx)

	invSeq, hasInv := readSeqFrom(db, KeySeqInvoice, dapurID)
	poSeq, hasPo := readSeqFrom(db, KeySeqPO, dapurID)

	if !hasInv || !hasPo {
		maxInv, maxPo, err := getDocumentSequenceMaxFrom(db, dapurID)
		if err != nil {
			return nil, err
		}
		if !hasInv {
			if err := upsertSeqTo(db, KeySeqInvoice, dapurID, maxInv); err != nil {
				s.log.Errorf("failed to seed invoice seq: %v", err)
				return nil, exception.InternalServerError
			}
			invSeq = maxInv
		}
		if !hasPo {
			if err := upsertSeqTo(db, KeySeqPO, dapurID, maxPo); err != nil {
				s.log.Errorf("failed to seed po seq: %v", err)
				return nil, exception.InternalServerError
			}
			poSeq = maxPo
		}
	}

	invNo := fmt.Sprintf("INV-%03d", invSeq+1)
	poNo := fmt.Sprintf("PO-%03d", poSeq+1)

	return &model.DocumentSequenceResponse{
		NextInvoiceNo: invNo,
		NextPONumber:  poNo,
	}, nil
}
