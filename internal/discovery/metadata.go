package discovery

import (
	"path/filepath"
	"time"
)

func FromWindowsMetadata(path string, attributes, tag uint32, size int64, modified time.Time) (File, bool, error) {
	const directory = 0x10
	const reparse = 0x400
	const nameSurrogate = 0x20000000
	const unavailable = 0x1000 | 0x40000 | 0x400000 // OFFLINE, RECALL_ON_OPEN, RECALL_ON_DATA_ACCESS
	if attributes&directory != 0 || attributes&reparse != 0 && tag&nameSurrogate != 0 {
		return File{}, false, nil
	}
	// Support cloud placeholders and deduplicated data, not unknown special files.
	if attributes&reparse != 0 && tag&0xffff0fff != 0x9000001a && tag != 0x80000013 {
		return File{}, false, nil
	}
	f := File{Path: path, Name: filepath.Base(path), Size: size, ModTime: modified}
	if attributes&unavailable != 0 {
		f.Problem = DownloadRequired
	}
	return f, true, nil
}
