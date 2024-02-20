// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.19

package store

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	_ "crypto/sha256" // ensure ids can be computed
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/docker/docker/errdefs"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

const restrictedNamePattern = "^[a-zA-Z0-9][a-zA-Z0-9_.+-]+$"

var restrictedNameRegEx = regexp.MustCompile(restrictedNamePattern)

// Store provides a context store for easily remembering endpoints configuration
type Store interface {
	Reader
	Lister
	Writer
	StorageInfoProvider
}

// Reader provides read-only (without list) access to context data
type Reader interface {
	GetMetadata(name string) (Metadata, error)
	ListTLSFiles(name string) (map[string]EndpointFiles, error)
	GetTLSData(contextName, endpointName, fileName string) ([]byte, error)
}

// Lister provides listing of contexts
type Lister interface {
	List() ([]Metadata, error)
}

// ReaderLister combines Reader and Lister interfaces
type ReaderLister interface {
	Reader
	Lister
}

// StorageInfoProvider provides more information about storage details of contexts
type StorageInfoProvider interface {
	GetStorageInfo(contextName string) StorageInfo
}

// Writer provides write access to context data
type Writer interface {
	CreateOrUpdate(meta Metadata) error
	Remove(name string) error
	ResetTLSMaterial(name string, data *ContextTLSData) error
	ResetEndpointTLSMaterial(contextName string, endpointName string, data *EndpointTLSData) error
}

// ReaderWriter combines Reader and Writer interfaces
type ReaderWriter interface {
	Reader
	Writer
}

// Metadata contains metadata about a context and its endpoints
type Metadata struct {
	Name      string         `json:",omitempty"`
	Metadata  any            `json:",omitempty"`
	Endpoints map[string]any `json:",omitempty"`
}

// StorageInfo contains data about where a given context is stored
type StorageInfo struct {
	MetadataPath string
	TLSPath      string
}

// EndpointTLSData represents tls data for a given endpoint
type EndpointTLSData struct {
	Files map[string][]byte
}

// ContextTLSData represents tls data for a whole context
type ContextTLSData struct {
	Endpoints map[string]EndpointTLSData
}

// New creates a store from a given directory.
// If the directory does not exist or is empty, initialize it
func New(dir string, cfg Config) *ContextStore {
	metaRoot := filepath.Join(dir, metadataDir)
	tlsRoot := filepath.Join(dir, tlsDir)

	return &ContextStore{
		meta: &metadataStore{
			root:   metaRoot,
			config: cfg,
		},
		tls: &tlsStore{
			root: tlsRoot,
		},
	}
}

// ContextStore implements Store.
type ContextStore struct {
	meta *metadataStore
	tls  *tlsStore
}

// List return all contexts.
func (s *ContextStore) List() ([]Metadata, error) {
	return s.meta.list()
}

// Names return Metadata names for a Lister
func Names(s Lister) ([]string, error) {
	list, err := s.List()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(list))
	for _, item := range list {
		names = append(names, item.Name)
	}
	return names, nil
}

// CreateOrUpdate creates or updates metadata for the context.
func (s *ContextStore) CreateOrUpdate(meta Metadata) error {
	return s.meta.createOrUpdate(meta)
}

// Remove deletes the context with the given name, if found.
func (s *ContextStore) Remove(name string) error {
	if err := s.meta.remove(name); err != nil {
		return errors.Wrapf(err, "failed to remove context %s", name)
	}
	if err := s.tls.remove(name); err != nil {
		return errors.Wrapf(err, "failed to remove context %s", name)
	}
	return nil
}

// GetMetadata returns the metadata for the context with the given name.
// It returns an errdefs.ErrNotFound if the context was not found.
func (s *ContextStore) GetMetadata(name string) (Metadata, error) {
	return s.meta.get(name)
}

// ResetTLSMaterial removes TLS data for all endpoints in the context and replaces
// it with the new data.
func (s *ContextStore) ResetTLSMaterial(name string, data *ContextTLSData) error {
	if err := s.tls.remove(name); err != nil {
		return err
	}
	if data == nil {
		return nil
	}
	for ep, files := range data.Endpoints {
		for fileName, data := range files.Files {
			if err := s.tls.createOrUpdate(name, ep, fileName, data); err != nil {
				return err
			}
		}
	}
	return nil
}

// ResetEndpointTLSMaterial removes TLS data for the given context and endpoint,
// and replaces it with the new data.
func (s *ContextStore) ResetEndpointTLSMaterial(contextName string, endpointName string, data *EndpointTLSData) error {
	if err := s.tls.removeEndpoint(contextName, endpointName); err != nil {
		return err
	}
	if data == nil {
		return nil
	}
	for fileName, data := range data.Files {
		if err := s.tls.createOrUpdate(contextName, endpointName, fileName, data); err != nil {
			return err
		}
	}
	return nil
}

// ListTLSFiles returns the list of TLS files present for each endpoint in the
// context.
func (s *ContextStore) ListTLSFiles(name string) (map[string]EndpointFiles, error) {
	return s.tls.listContextData(name)
}

// GetTLSData reads, and returns the content of the given fileName for an endpoint.
// It returns an errdefs.ErrNotFound if the file was not found.
func (s *ContextStore) GetTLSData(contextName, endpointName, fileName string) ([]byte, error) {
	return s.tls.getData(contextName, endpointName, fileName)
}

// GetStorageInfo returns the paths where the Metadata and TLS data are stored
// for the context.
func (s *ContextStore) GetStorageInfo(contextName string) StorageInfo {
	return StorageInfo{
		MetadataPath: s.meta.contextDir(contextdirOf(contextName)),
		TLSPath:      s.tls.contextDir(contextName),
	}
}

// ValidateContextName checks a context name is valid.
func ValidateContextName(name string) error {
	if name == "" {
		return errors.New("context name cannot be empty")
	}
	if name == "default" {
		return errors.New(`"default" is a reserved context name`)
	}
	if !restrictedNameRegEx.MatchString(name) {
		return errors.Errorf("context name %q is invalid, names are validated against regexp %q", name, restrictedNamePattern)
	}
	return nil
}

// Export exports an existing namespace into an opaque data stream
// This stream is actually a tarball containing context metadata and TLS materials, but it does
// not map 1:1 the layout of the context store (don't try to restore it manually without calling store.Import)
func Export(name string, s Reader) io.ReadCloser {
	reader, writer := io.Pipe()
	go func() {
		tw := tar.NewWriter(writer)
		defer tw.Close()
		defer writer.Close()
		meta, err := s.GetMetadata(name)
		if err != nil {
			writer.CloseWithError(err)
			return
		}
		metaBytes, err := json.Marshal(&meta)
		if err != nil {
			writer.CloseWithError(err)
			return
		}
		if err = tw.WriteHeader(&tar.Header{
			Name: metaFile,
			Mode: 0o644,
			Size: int64(len(metaBytes)),
		}); err != nil {
			writer.CloseWithError(err)
			return
		}
		if _, err = tw.Write(metaBytes); err != nil {
			writer.CloseWithError(err)
			return
		}
		tlsFiles, err := s.ListTLSFiles(name)
		if err != nil {
			writer.CloseWithError(err)
			return
		}
		if err = tw.WriteHeader(&tar.Header{
			Name:     "tls",
			Mode:     0o700,
			Size:     0,
			Typeflag: tar.TypeDir,
		}); err != nil {
			writer.CloseWithError(err)
			return
		}
		for endpointName, endpointFiles := range tlsFiles {
			if err = tw.WriteHeader(&tar.Header{
				Name:     path.Join("tls", endpointName),
				Mode:     0o700,
				Size:     0,
				Typeflag: tar.TypeDir,
			}); err != nil {
				writer.CloseWithError(err)
				return
			}
			for _, fileName := range endpointFiles {
				data, err := s.GetTLSData(name, endpointName, fileName)
				if err != nil {
					writer.CloseWithError(err)
					return
				}
				if err = tw.WriteHeader(&tar.Header{
					Name: path.Join("tls", endpointName, fileName),
					Mode: 0o600,
					Size: int64(len(data)),
				}); err != nil {
					writer.CloseWithError(err)
					return
				}
				if _, err = tw.Write(data); err != nil {
					writer.CloseWithError(err)
					return
				}
			}
		}
	}()
	return reader
}

const (
	maxAllowedFileSizeToImport int64  = 10 << 20
	zipType                    string = "application/zip"
)

func getImportContentType(r *bufio.Reader) (string, error) {
	head, err := r.Peek(512)
	if err != nil && err != io.EOF {
		return "", err
	}

	return http.DetectContentType(head), nil
}

// Import imports an exported context into a store
func Import(name string, s Writer, reader io.Reader) error {
	// Buffered reader will not advance the buffer, needed to determine content type
	r := bufio.NewReader(reader)

	importContentType, err := getImportContentType(r)
	if err != nil {
		return err
	}
	switch importContentType {
	case zipType:
		return importZip(name, s, r)
	default:
		// Assume it's a TAR (TAR does not have a "magic number")
		return importTar(name, s, r)
	}
}

func isValidFilePath(p string) error {
	if p != metaFile && !strings.HasPrefix(p, "tls/") {
		return errors.New("unexpected context file")
	}
	if path.Clean(p) != p {
		return errors.New("unexpected path format")
	}
	if strings.Contains(p, `\`) {
		return errors.New(`unexpected '\' in path`)
	}
	return nil
}

var importTarCoverage = map[int]bool{
	1: false,
	2: false,
	3: false,
	4: false,
	5: false,
	6: false,
	7: false,
	8: false,
	9: false,
	10: false,
	11: false,
	12: false,
	13: false,
	14: false,
	15: false,
	16: false,
	17: false,
	18: false,
	19: false,
	20: false,
	21: false,
	22: false,
	23: false,
}

func importTar(name string, s Writer, reader io.Reader) error {
	tr := tar.NewReader(&LimitedReader{R: reader, N: maxAllowedFileSizeToImport})
	tlsData := ContextTLSData{
		Endpoints: map[string]EndpointTLSData{},
	}
	var importedMetaFile bool
	for {
		hdr, err := tr.Next()
		importTarCoverage[1] = true
		if err == io.EOF {
			importTarCoverage[1] = true
			break
		} else {
			importTarCoverage[2] = true
		}
		if err != nil {
			importTarCoverage[3] = true
			return err
		} else {
			importTarCoverage[4] = true
		}
		if hdr.Typeflag != tar.TypeReg {
			importTarCoverage[5] = true
			// skip this entry, only taking files into account
			continue
		} else {
			importTarCoverage[6] = true
		}
		if err := isValidFilePath(hdr.Name); err != nil {
			importTarCoverage[7] = true
			return errors.Wrap(err, hdr.Name)
		} else {
			importTarCoverage[8] = true
		}
		if hdr.Name == metaFile {
			importTarCoverage[9] = true
			data, err := io.ReadAll(tr)
			if err != nil {
				importTarCoverage[10] = true
				return err
			} else {
				importTarCoverage[11] = true
			}
			meta, err := parseMetadata(data, name)
			if err != nil {
				importTarCoverage[12] = true
				return err
			} else {
				importTarCoverage[13] = true
			}
			if err := s.CreateOrUpdate(meta); err != nil {
				importTarCoverage[14] = true
				return err
			} else {
				importTarCoverage[15] = true
			}
			importedMetaFile = true
		} else if strings.HasPrefix(hdr.Name, "tls/") {
			importTarCoverage[16] = true
			data, err := io.ReadAll(tr)
			if err != nil {
				importTarCoverage[17] = true
				return err
			} else {
				importTarCoverage[18] = true
			}
			if err := importEndpointTLS(&tlsData, hdr.Name, data); err != nil {
				importTarCoverage[19] = true
				return err
			} else {
				importTarCoverage[20] = true
			}
		} else {
			importTarCoverage[21] = true
		}
	}
	if !importedMetaFile {
		importTarCoverage[22] = true
		return errdefs.InvalidParameter(errors.New("invalid context: no metadata found"))
	} else {
		importTarCoverage[23] = true
	}

	writeCoverageToFile("importTarCoverage.txt", importTarCoverage)

	return s.ResetTLSMaterial(name, &tlsData)
}

func writeCoverageToFile(filename string, data map[int]bool) {
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var builder strings.Builder
	var sum float64
	type keyValue struct {
		key int
		val bool
	}
	var kv []keyValue

	for k, v := range data {
		if v {
			sum++
		}
		kv = append(kv, keyValue{k, v})
	}

	sort.Slice(kv, func(i, j int) bool {
		return kv[i].key < kv[j].key
	})

	for _, v := range kv {
		builder.WriteString(fmt.Sprintf("Branch %d\t -\t %t\n", v.key, v.val))
	}

	builder.WriteString(fmt.Sprintf("\nCoverage Percentage - %2.f%%", (sum / float64(len(data)))*100))

	f.WriteString(builder.String())
}

func importZip(name string, s Writer, reader io.Reader) error {
	body, err := io.ReadAll(&LimitedReader{R: reader, N: maxAllowedFileSizeToImport})
	if err != nil {
		return err
	}
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return err
	}
	tlsData := ContextTLSData{
		Endpoints: map[string]EndpointTLSData{},
	}

	var importedMetaFile bool
	for _, zf := range zr.File {
		fi := zf.FileInfo()
		if !fi.Mode().IsRegular() {
			// skip this entry, only taking regular files into account
			continue
		}
		if err := isValidFilePath(zf.Name); err != nil {
			return errors.Wrap(err, zf.Name)
		}
		if zf.Name == metaFile {
			f, err := zf.Open()
			if err != nil {
				return err
			}

			data, err := io.ReadAll(&LimitedReader{R: f, N: maxAllowedFileSizeToImport})
			defer f.Close()
			if err != nil {
				return err
			}
			meta, err := parseMetadata(data, name)
			if err != nil {
				return err
			}
			if err := s.CreateOrUpdate(meta); err != nil {
				return err
			}
			importedMetaFile = true
		} else if strings.HasPrefix(zf.Name, "tls/") {
			f, err := zf.Open()
			if err != nil {
				return err
			}
			data, err := io.ReadAll(f)
			defer f.Close()
			if err != nil {
				return err
			}
			err = importEndpointTLS(&tlsData, zf.Name, data)
			if err != nil {
				return err
			}
		}
	}
	if !importedMetaFile {
		return errdefs.InvalidParameter(errors.New("invalid context: no metadata found"))
	}
	return s.ResetTLSMaterial(name, &tlsData)
}

func parseMetadata(data []byte, name string) (Metadata, error) {
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return meta, err
	}
	if err := ValidateContextName(name); err != nil {
		return Metadata{}, err
	}
	meta.Name = name
	return meta, nil
}

func importEndpointTLS(tlsData *ContextTLSData, tlsPath string, data []byte) error {
	parts := strings.SplitN(strings.TrimPrefix(tlsPath, "tls/"), "/", 2)
	if len(parts) != 2 {
		// TLS endpoints require archived file directory with 2 layers
		// i.e. tls/{endpointName}/{fileName}
		return errors.New("archive format is invalid")
	}

	epName := parts[0]
	fileName := parts[1]
	if _, ok := tlsData.Endpoints[epName]; !ok {
		tlsData.Endpoints[epName] = EndpointTLSData{
			Files: map[string][]byte{},
		}
	}
	tlsData.Endpoints[epName].Files[fileName] = data
	return nil
}

type contextdir string

func contextdirOf(name string) contextdir {
	return contextdir(digest.FromString(name).Encoded())
}
