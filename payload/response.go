package payload

// LoginResponse is the data returned by menmos after a login call.
type LoginResponse struct {
	Token string `json:"token,omitempty"`
}

// MessageResponse is the simplest response returned by menmos.
type MessageResponse struct {
	Message string `json:"message,omitempty"`
}

// A BlobMeta contains the metadata of a single blob.
type BlobMeta struct {
	Fields map[string]string `json:"fields"` // TODO: Support multi field types
	Tags   []string          `json:"tags"`
}

func NewBlobMeta() BlobMeta {
	return BlobMeta{Fields: make(map[string]string), Tags: []string{}}
}

// Hit represents a single query result.
type Hit struct {
	ID       string   `json:"id,omitempty"`
	Metadata BlobMeta `json:"meta,omitempty"`
	URL      string   `json:"url,omitempty"`
}

// FacetResponse is the facet-related part of a query response.
type FacetResponse struct {
	Tags map[string]uint64            `json:"tags,omitempty"`
	Meta map[string]map[string]uint64 `json:"meta,omitempty"`
}

// QueryResponse is the data returned with a query.
type QueryResponse struct {
	Count  uint32         `json:"count,omitempty"`
	Total  uint32         `json:"total,omitempty"`
	Hits   []Hit          `json:"hits,omitempty"`
	Facets *FacetResponse `json:"facets,omitempty"`
}

// PushResponse is the data returned on blob creation.
type PushResponse struct {
	ID string `json:"id,omitempty"`
}

type GetMetadataResponse struct {
	Metadata *BlobMeta `json:"meta"`
}

// StorageNodeInfo is the payload returned by ListStorageNodes.
type StorageNodeInfo struct {
	ID             string `json:"id,omitempty"`
	Port           uint16 `json:"port,omitempty"`
	Size           uint64 `json:"size,omitempty"`
	AvailableSpace uint64 `json:"available_space,omitempty"`
}

type ListStorageNodesResponse struct {
	StorageNodes []StorageNodeInfo `json:"storage_nodes,omitempty"`
}
