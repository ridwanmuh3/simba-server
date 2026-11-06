package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"gorm.io/gorm"

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
	return &SettingService{db: db, settingRepo: settingRepo, log: log, validate: validate}
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

func (s *SettingService) GetCompanyProfile(ctx context.Context, dapurID uint) (*model.CompanyProfileResponse, error) {
	db := s.db.WithContext(ctx)
	name, _ := s.settingRepo.GetByKey(db, KeyCompanyName, dapurID)
	address, _ := s.settingRepo.GetByKey(db, KeyCompanyAddress, dapurID)
	contact, _ := s.settingRepo.GetByKey(db, KeyCompanyContact, dapurID)
	bank, _ := s.settingRepo.GetByKey(db, KeyBankAccount, dapurID)
	pj, _ := s.settingRepo.GetByKey(db, KeyPenanggungjawab, dapurID)
	jabatan, _ := s.settingRepo.GetByKey(db, KeyJabatan, dapurID)

	resp := &model.CompanyProfileResponse{BankAccount: DefaultBankAccount}
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
		return exception.InvalidRequestError
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
	invSeq, hasInv := repository.ReadSeq(db, KeySeqInvoice, dapurID)
	poSeq, hasPo := repository.ReadSeq(db, KeySeqPO, dapurID)

	if !hasInv || !hasPo {
		maxInv, maxPo, err := repository.MaxDocumentSequence(db, dapurID)
		if err != nil {
			return nil, err
		}
		if !hasInv {
			if err := repository.UpsertSeq(db, KeySeqInvoice, dapurID, maxInv); err != nil {
				s.log.Errorf("failed to seed invoice seq: %v", err)
				return nil, exception.InternalServerError
			}
			invSeq = maxInv
		}
		if !hasPo {
			if err := repository.UpsertSeq(db, KeySeqPO, dapurID, maxPo); err != nil {
				s.log.Errorf("failed to seed po seq: %v", err)
				return nil, exception.InternalServerError
			}
			poSeq = maxPo
		}
	}

	return &model.DocumentSequenceResponse{
		NextInvoiceNo: fmt.Sprintf("INV-%03d", invSeq+1),
		NextPONumber:  fmt.Sprintf("PO-%03d", poSeq+1),
	}, nil
}
