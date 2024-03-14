package papersappsdk

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

// ImportData import data
//
// swagger:model ImportData
type ImportData struct {

	// Client that did the import
	ImportedBy string `json:"imported_by,omitempty"`

	// original id
	OriginalID string `json:"original_id,omitempty"`

	// original type
	OriginalType string `json:"original_type,omitempty"`

	// Source from which the item is imported
	Source string `json:"source,omitempty"`
}