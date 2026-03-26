package main

import (
	"encoding/binary"

	"github.com/hashicorp/go-memdb"
)

// queryField tracks which dimensions are active in a query,
// distinct from matchFlag which tracks match semantics.
type queryField uint

const (
	queryName       queryField = 1 << iota
	queryParent
	queryPathSuffix
	querySize
	queryChecksum
)

func (f queryField) has(flag queryField) bool {
	return f&flag == flag
}

type query struct {
	fields     queryField
	Name       string
	Parent     string
	PathSuffix string
	Size       int64
	Checksum   checksum
}

func (r *fileRecord) byName(q *query) *query {
	q.fields |= queryName
	q.Name = r.FoldedName
	return q
}

func (r *fileRecord) byParent(q *query) *query {
	q.fields |= queryParent
	q.Parent = r.FoldedParent
	return q
}

func (r *fileRecord) byPathSuffix(q *query) *query {
	q = r.byParent(q)
	q.fields |= queryPathSuffix
	q.PathSuffix = r.PathSuffix
	return q
}

func (r *fileRecord) bySize(q *query) *query {
	q.fields |= querySize
	q.Size = r.Size()
	return q
}

// If !HasChecksum, equivalent to bySize()
func (r *fileRecord) byChecksum(q *query) *query {
	q = r.bySize(q)
	if r.HasChecksum {
		q.fields |= queryChecksum
		q.Checksum = r.Checksum
	}
	return q
}

// -- Indexers --

type filePathIndexer struct{}

func (f *filePathIndexer) FromObject(raw interface{}) (bool, []byte, error) {
	r := raw.(*fileRecord)
	return true, []byte(r.FilePath + "\x00"), nil
}

func (f *filePathIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	return []byte(args[0].(string) + "\x00"), nil
}

type foldedNameIndexer struct{}

func (f *foldedNameIndexer) FromObject(raw interface{}) (bool, []byte, error) {
	r := raw.(*fileRecord)
	if r.FoldedName == "" {
		return false, nil, nil
	}
	return true, []byte(r.FoldedName + "\x00"), nil
}

func (f *foldedNameIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	return []byte(args[0].(string) + "\x00"), nil
}

type foldedParentIndexer struct{}

func (f *foldedParentIndexer) FromObject(raw interface{}) (bool, []byte, error) {
	r := raw.(*fileRecord)
	if r.FoldedParent == "" {
		return false, nil, nil
	}
	return true, []byte(r.FoldedParent + "\x00"), nil
}

func (f *foldedParentIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	return []byte(args[0].(string) + "\x00"), nil
}

type pathSuffixIndexer struct{}

func (f *pathSuffixIndexer) FromObject(raw interface{}) (bool, []byte, error) {
	r := raw.(*fileRecord)
	if r.FoldedParent == "" && r.PathSuffix == "" {
		return false, nil, nil
	}
	return true, []byte(r.FoldedParent + "\x00" + r.PathSuffix + "\x00"), nil
}

func (f *pathSuffixIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	return []byte(args[0].(string) + "\x00" + args[1].(string) + "\x00"), nil
}

type sizeIndexer struct{}

func (f *sizeIndexer) FromObject(raw interface{}) (bool, []byte, error) {
	r := raw.(*fileRecord)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(r.Size()))
	return true, buf, nil
}

func (f *sizeIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(args[0].(int64)))
	return buf, nil
}

type checksumIndexer struct{}

func (f *checksumIndexer) FromObject(raw interface{}) (bool, []byte, error) {
	r := raw.(*fileRecord)
	if !r.HasChecksum {
		return false, nil, nil
	}
	buf := make([]byte, 8+ChecksumBlockSize)
	binary.BigEndian.PutUint64(buf[:8], uint64(r.Checksum.size))
	copy(buf[8:], r.Checksum.hash[:])
	return true, buf, nil
}

func (f *checksumIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	cs := args[0].(checksum)
	buf := make([]byte, 8+ChecksumBlockSize)
	binary.BigEndian.PutUint64(buf[:8], uint64(cs.size))
	copy(buf[8:], cs.hash[:])
	return buf, nil
}

// -- Schema --

var dbSchema = &memdb.DBSchema{
	Tables: map[string]*memdb.TableSchema{
		"files": {
			Name: "files",
			Indexes: map[string]*memdb.IndexSchema{
				"id": {
					Name:    "id",
					Unique:  true,
					Indexer: &filePathIndexer{},
				},
				"name": {
					Name:         "name",
					Unique:       false,
					AllowMissing: true,
					Indexer:      &foldedNameIndexer{},
				},
				"parent": {
					Name:         "parent",
					Unique:       false,
					AllowMissing: true,
					Indexer:      &foldedParentIndexer{},
				},
				"pathsuffix": {
					Name:         "pathsuffix",
					Unique:       false,
					AllowMissing: true,
					Indexer:      &pathSuffixIndexer{},
				},
				"size": {
					Name:         "size",
					Unique:       false,
					AllowMissing: true,
					Indexer:      &sizeIndexer{},
				},
				"checksum": {
					Name:         "checksum",
					Unique:       false,
					AllowMissing: true,
					Indexer:      &checksumIndexer{},
				},
			},
		},
	},
}

// -- DB wrapper --

type db struct {
	store *memdb.MemDB
}

func newDB() *db {
	store, err := memdb.NewMemDB(dbSchema)
	if err != nil {
		panic(err)
	}
	return &db{store: store}
}

func (d *db) insert(r *fileRecord) {
	txn := d.store.Txn(true)
	txn.Insert("files", r)
	txn.Commit()
}

func (d *db) remove(r *fileRecord) {
	txn := d.store.Txn(true)
	txn.Delete("files", r)
	txn.Commit()
}

func (d *db) query(q *query) recordSet {
	txn := d.store.Txn(false)
	defer txn.Abort()

	var iter memdb.ResultIterator
	var err error

	// Pick the most selective index among active fields.
	switch {
	case q.fields.has(queryChecksum):
		iter, err = txn.Get("files", "checksum", q.Checksum)
	case q.fields.has(querySize):
		iter, err = txn.Get("files", "size", q.Size)
	case q.fields.has(queryPathSuffix):
		iter, err = txn.Get("files", "pathsuffix", q.Parent, q.PathSuffix)
	case q.fields.has(queryParent):
		iter, err = txn.Get("files", "parent", q.Parent)
	case q.fields.has(queryName):
		iter, err = txn.Get("files", "name", q.Name)
	default:
		return nil
	}

	if err != nil {
		return nil
	}

	result := make(recordSet)
	for obj := iter.Next(); obj != nil; obj = iter.Next() {
		r := obj.(*fileRecord)
		if matchesQuery(r, q) {
			result[r] = struct{}{}
		}
	}
	return result
}

func matchesQuery(r *fileRecord, q *query) bool {
	if q.fields.has(queryName) && r.FoldedName != q.Name {
		return false
	}
	if q.fields.has(queryParent) && r.FoldedParent != q.Parent {
		return false
	}
	if q.fields.has(queryPathSuffix) && r.PathSuffix != q.PathSuffix {
		return false
	}
	if q.fields.has(querySize) && r.Size() != q.Size {
		return false
	}
	if q.fields.has(queryChecksum) && r.Checksum != q.Checksum {
		return false
	}
	return true
}

func (d *db) allRecords() []*fileRecord {
	txn := d.store.Txn(false)
	defer txn.Abort()
	iter, err := txn.Get("files", "id")
	if err != nil {
		return nil
	}
	var result []*fileRecord
	for obj := iter.Next(); obj != nil; obj = iter.Next() {
		result = append(result, obj.(*fileRecord))
	}
	return result
}
