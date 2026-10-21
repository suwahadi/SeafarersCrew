package request

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// Limits of a menu.
const (
	MaxMenuItems    = 30
	MaxMenuChildren = 20
)

// MenuChild is a second-level menu item.
type MenuChild struct {
	Label        string            `json:"label"`
	PageID       *uuid.UUID        `json:"page_id"`
	URL          string            `json:"url"`
	OpenInNewTab bool              `json:"open_in_new_tab"`
	Children     []json.RawMessage `json:"children"`
}

// MenuItem is a top-level menu item.
type MenuItem struct {
	Label        string      `json:"label"`
	PageID       *uuid.UUID  `json:"page_id"`
	URL          string      `json:"url"`
	OpenInNewTab bool        `json:"open_in_new_tab"`
	Children     []MenuChild `json:"children"`
}

// Menu is the body of PUT /menus/{header|footer}.
type Menu struct {
	Items []MenuItem `json:"items"`
}

// Validate implements Validator.
func (r *Menu) Validate() map[string]string {
	f := FieldErrors{}
	if r.Items == nil {
		f.set("items", "The items are required; send an empty list to clear the menu.")
		return f
	}
	if len(r.Items) > MaxMenuItems {
		f.set("items", fmt.Sprintf("A menu may have at most %d items.", MaxMenuItems))
		return f
	}
	for i := range r.Items {
		it := &r.Items[i]
		prefix := fmt.Sprintf("items.%d.", i)
		validateMenuLink(f, prefix, &it.Label, it.PageID, &it.URL)
		if len(it.Children) > MaxMenuChildren {
			f.set(prefix+"children", fmt.Sprintf("An item may have at most %d children.", MaxMenuChildren))
			continue
		}
		for j := range it.Children {
			c := &it.Children[j]
			cp := fmt.Sprintf("%schildren.%d.", prefix, j)
			validateMenuLink(f, cp, &c.Label, c.PageID, &c.URL)
			if len(c.Children) > 0 {
				f.set(cp+"children", "Menus support two levels only.")
			}
		}
	}
	return f
}

func validateMenuLink(f FieldErrors, prefix string, label *string, pageID *uuid.UUID, url *string) {
	*label, *url = strings.TrimSpace(*label), strings.TrimSpace(*url)
	f.Required(prefix+"label", *label, 80)
	switch {
	case pageID != nil && *url != "":
		f.set(prefix+"url", "Choose either a page or a custom link, not both.")
	case pageID == nil && *url == "":
		f.set(prefix+"page_id", "Choose a page or enter a custom link.")
	case pageID == nil:
		f.Link(prefix+"url", *url)
	}
}

// Model converts the request to menu items.
func (r *Menu) Model() []model.MenuItem {
	out := make([]model.MenuItem, len(r.Items))
	for i, it := range r.Items {
		item := model.MenuItem{Label: it.Label, PageID: it.PageID, OpenInNewTab: it.OpenInNewTab}
		if it.PageID == nil {
			item.URL = it.URL
		}
		for _, c := range it.Children {
			child := model.MenuItem{Label: c.Label, PageID: c.PageID, OpenInNewTab: c.OpenInNewTab}
			if c.PageID == nil {
				child.URL = c.URL
			}
			item.Children = append(item.Children, child)
		}
		out[i] = item
	}
	return out
}
