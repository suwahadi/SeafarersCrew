package service

import (
	"time"

	"github.com/google/uuid"
)

// Compliance item states from PRD 6.3 tab 9.
const (
	ItemValid    = "valid"
	ItemExpiring = "expiring"
	ItemExpired  = "expired"
	ItemMissing  = "missing"
)

// Requirement is one mandatory document or certificate type for a rank.
type Requirement struct {
	Kind      string
	TypeID    uuid.UUID
	TypeName  string
	HasExpiry bool
}

// HeldItem is a document or certificate the crew member holds.
type HeldItem struct {
	Kind        string
	TypeID      uuid.UUID
	Number      string
	ExpiredDate *time.Time
}

// ComplianceItem is the evaluated state of one requirement.
type ComplianceItem struct {
	Kind        string     `json:"kind"`
	TypeID      uuid.UUID  `json:"type_id"`
	TypeName    string     `json:"type_name"`
	Status      string     `json:"status"`
	Badge       string     `json:"badge"`
	Number      string     `json:"number,omitempty"`
	ExpiredDate *time.Time `json:"expired_date,omitempty"`
	DaysLeft    *int       `json:"days_left,omitempty"`
}

// ComplianceResult is the checklist for one rank on one date.
type ComplianceResult struct {
	CheckDate time.Time        `json:"check_date"`
	Compliant bool             `json:"compliant"`
	Items     []ComplianceItem `json:"items"`
}

// EvaluateCompliance checks every requirement against the held items on checkDate (PRD 11.3).
func EvaluateCompliance(reqs []Requirement, held []HeldItem, checkDate time.Time, t Thresholds) ComplianceResult {
	best := bestHeldByType(held)
	res := ComplianceResult{CheckDate: checkDate, Compliant: true, Items: make([]ComplianceItem, 0, len(reqs))}
	for _, r := range reqs {
		item := ComplianceItem{Kind: r.Kind, TypeID: r.TypeID, TypeName: r.TypeName}
		h, ok := best[heldKey{r.Kind, r.TypeID}]
		switch {
		case !ok:
			item.Status, item.Badge = ItemMissing, BadgeExpired
		default:
			item.Number = h.Number
			expiry := h.ExpiredDate
			if !r.HasExpiry {
				expiry = nil
			}
			item.ExpiredDate = expiry
			item.DaysLeft = DaysToExpiry(expiry, checkDate)
			item.Badge = ExpiryBadge(expiry, checkDate, t)
			item.Status = itemStatus(item.Badge)
		}
		if item.Status == ItemMissing || item.Status == ItemExpired {
			res.Compliant = false
		}
		res.Items = append(res.Items, item)
	}
	return res
}

func itemStatus(badge string) string {
	switch badge {
	case BadgeExpired:
		return ItemExpired
	case BadgeGreen:
		return ItemValid
	default:
		return ItemExpiring
	}
}

type heldKey struct {
	kind string
	id   uuid.UUID
}

func bestHeldByType(held []HeldItem) map[heldKey]HeldItem {
	best := make(map[heldKey]HeldItem, len(held))
	for _, h := range held {
		k := heldKey{h.Kind, h.TypeID}
		cur, ok := best[k]
		if !ok || laterExpiry(h.ExpiredDate, cur.ExpiredDate) {
			best[k] = h
		}
	}
	return best
}

func laterExpiry(a, b *time.Time) bool {
	if a == nil {
		return b != nil
	}
	if b == nil {
		return false
	}
	return a.After(*b)
}
