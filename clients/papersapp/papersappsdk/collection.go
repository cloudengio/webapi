package papersappsdk

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

// Collection Collection Schema
//
// swagger:model Collection
type Collection struct {

	// custom fields
	CustomFields []*CustomField `json:"custom_fields"`

	// Unique collection id (e.g. 7f5e3fe2-bcd7-42b5-972e-c271f5449977)
	ID string `json:"id,omitempty"`

	// Collection name (e.g. Elephant Shark Shared Library)
	Name string `json:"name,omitempty"`

	// owner
	Owner *CollectionOwner `json:"owner,omitempty"`

	// Indicates if the collection is shared or personal
	Shared bool `json:"shared,omitempty"`
}

// CollectionOwner Owner of collection (can be null)
//
// swagger:model CollectionOwner
type CollectionOwner struct {

	// email
	Email string `json:"email,omitempty"`

	// User unique ID
	ID string `json:"id,omitempty"`

	// name
	Name string `json:"name,omitempty"`
}
