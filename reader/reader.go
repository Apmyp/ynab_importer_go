// Package reader handles reading text messages from folders
package reader

// Reader reads text messages from a folder
type Reader struct {
	path string
}

// NewReader creates a new Reader instance
func NewReader() *Reader {
	return &Reader{}
}

// SetPath sets the folder path for reading messages
func (r *Reader) SetPath(path string) {
	r.path = path
}

// GetPath returns the current folder path
func (r *Reader) GetPath() string {
	return r.path
}
