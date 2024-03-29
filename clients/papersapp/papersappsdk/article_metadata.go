package papersappsdk

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

// ArticleMetadata Article Metadata Schema
//
// swagger:model ArticleMetadata
type ArticleMetadata struct {

	// Abstract text
	Abstract string `json:"abstract,omitempty"`

	// Authors names
	Authors []string `json:"authors"`

	// Chapter title (books only)
	Chapter string `json:"chapter,omitempty"`

	// eisbn
	Eisbn string `json:"eisbn,omitempty"`

	// eissn
	Eissn string `json:"eissn,omitempty"`

	// isbn
	Isbn string `json:"isbn,omitempty"`

	// Journal ISSN (e.g. 1523-4681)
	Issn string `json:"issn,omitempty"`

	// Issue number (e.g. 12)
	Issue string `json:"issue,omitempty"`

	// Journal name (e.g. Journal of Bone and Mineral Research)
	Journal string `json:"journal,omitempty"`

	// Abbreviation of journal title (e.g. J Bone Miner Res)
	JournalAbbrev string `json:"journal_abbrev,omitempty"`

	// pagination
	Pagination string `json:"pagination,omitempty"`

	// Article or book title (e.g. Parathyroid hormone: past and present)
	Title string `json:"title,omitempty"`

	// Publisher URL (e.g. https://asbmr.onlinelibrary.wiley.com/doi/10.1002/jbmr.178)
	URL string `json:"url,omitempty"`

	// Volume number (e.g. 1)
	Volume string `json:"volume,omitempty"`

	// Publication year (e.g. 1523-4681)
	// NOTE: swagger thinks this is a float, most likely because of the example
	// above.
	Year int `json:"year,omitempty"`
}
