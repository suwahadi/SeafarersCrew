package admin

import (
	"context"
	"net/http"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

// MasterRoutes is the set of handlers mounted for one master.
type MasterRoutes interface {
	List(w http.ResponseWriter, r *http.Request)
	Show(w http.ResponseWriter, r *http.Request)
	Create(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	Delete(w http.ResponseWriter, r *http.Request)
	Reorder(w http.ResponseWriter, r *http.Request)
}

// NewMasterRoutes builds the handlers of every master keyed by URL segment.
func NewMasterRoutes(m service.MasterData) map[string]MasterRoutes {
	return map[string]MasterRoutes{
		"companies": NewMasterHandler[model.Company, request.Company](m.Companies, present(resource.NewCompany),
			map[string]Filter{"type": {Kind: FilterEnum, Options: request.CompanyTypes()}}),
		"vessels": NewMasterHandler[model.Vessel, request.Vessel](m.Vessels, present(resource.NewVessel), map[string]Filter{
			"status":           {Kind: FilterEnum, Options: request.VesselStatuses()},
			"vessel_type_id":   {Kind: FilterUUID},
			"principal_id":     {Kind: FilterUUID},
			"owner_company_id": {Kind: FilterUUID},
		}),
		"ranks": NewMasterHandler[model.Rank, request.Rank](m.Ranks, present(resource.NewRank), map[string]Filter{
			"department": {Kind: FilterEnum, Options: request.Departments()},
			"is_officer": {Kind: FilterBool},
		}),
		"vessel-types": NewMasterHandler[model.VesselType, request.VesselType](m.VesselTypes, present(resource.NewVesselType), nil),
		"currencies":   NewMasterHandler[model.Currency, request.Currency](m.Currencies, present(resource.NewCurrency), nil),
		"crew-statuses": NewMasterHandler[model.CrewStatus, request.CrewStatus](m.CrewStatuses, present(resource.NewCrewStatus),
			map[string]Filter{"is_system": {Kind: FilterBool}}),
		"allowance-types": NewMasterHandler[model.AllowanceType, request.AllowanceType](m.AllowanceTypes, present(resource.NewAllowanceType),
			map[string]Filter{"basis": {Kind: FilterEnum, Options: request.AllowanceBases()}}),
		"document-types": NewMasterHandler[model.DocumentType, request.DocumentType](m.DocumentTypes, present(resource.NewDocumentType),
			map[string]Filter{"has_expiry": {Kind: FilterBool}}),
		"certificate-types": NewMasterHandler[model.CertificateType, request.CertificateType](m.CertificateTypes, present(resource.NewCertificateType), map[string]Filter{
			"category":   {Kind: FilterEnum, Options: request.CertificateCategories()},
			"has_expiry": {Kind: FilterBool},
		}),
	}
}

func present[T, R any](fn func(T) R) func(T) any {
	return func(t T) any { return fn(t) }
}

// OptionsService returns cached dropdown options.
type OptionsService interface {
	Options(ctx context.Context) (model.MasterOptions, error)
}

// ComplianceMatrixService reads and replaces the compliance matrix.
type ComplianceMatrixService interface {
	Matrix(ctx context.Context) (model.ComplianceMatrix, error)
	Replace(ctx context.Context, cells []model.ComplianceCell) (model.ComplianceMatrix, error)
}

// MasterDataHandler serves the options payload and the compliance matrix.
type MasterDataHandler struct {
	options    OptionsService
	compliance ComplianceMatrixService
}

// NewMasterDataHandler returns a MasterDataHandler.
func NewMasterDataHandler(options OptionsService, compliance ComplianceMatrixService) *MasterDataHandler {
	return &MasterDataHandler{options: options, compliance: compliance}
}

// Options handles GET /options.
func (h *MasterDataHandler) Options(w http.ResponseWriter, r *http.Request) {
	o, err := h.options.Options(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewMasterOptions(o))
}

// ComplianceMatrix handles GET /compliance-matrix.
func (h *MasterDataHandler) ComplianceMatrix(w http.ResponseWriter, r *http.Request) {
	m, err := h.compliance.Matrix(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewComplianceMatrix(m))
}

// ReplaceComplianceMatrix handles PUT /compliance-matrix.
func (h *MasterDataHandler) ReplaceComplianceMatrix(w http.ResponseWriter, r *http.Request) {
	var in request.ComplianceMatrix
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	m, err := h.compliance.Replace(r.Context(), in.Model())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewComplianceMatrix(m))
}
