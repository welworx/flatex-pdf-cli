package schema

// Output wraps transactions with optional metadata.
type Output struct {
	Metadata     *DocumentMetadata `json:"metadata,omitempty"`
	Transactions []*Transaction    `json:"transactions"`
}
