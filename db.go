package main

type queryGenerator func(r *fileRecord, q *query)

var queryGenerators [][]queryGenerator

func init() {
	singles := []queryGenerator{
		0: func(r *fileRecord, q *query) { r.byName(q) },
		1: func(r *fileRecord, q *query) { r.byParent(q) },
		2: func(r *fileRecord, q *query) { r.byPathSuffix(q) },
		3: func(r *fileRecord, q *query) { r.bySize(q) },
		4: func(r *fileRecord, q *query) { r.byChecksum(q) },
	}

	// Use binary decrement to generate all possible subsets based on bit position
	// We need (2<<N)-1 bits
	counter := 0
	for b := (1 << len(singles)) - 1; b > 0; b-- {
		counter++

		// queries will never include Checksum without Size
		if (b>>4)&1 == 1 && (b>>3)&1 != 1 {
			continue
		}

		// queries will never include PathSuffix without Parent
		if (b>>2)&1 == 1 && (b>>1)&1 != 1 {
			continue
		}

		var combo []queryGenerator
		for i, x := range singles {
			if (b>>i)&1 == 1 {
				combo = append(combo, x)
			}
		}
		queryGenerators = append(queryGenerators, combo)
	}
}

type query struct {
	Name       string
	Parent     string
	PathSuffix string
	Size       int64
	Checksum   checksum
}

func (r *fileRecord) byName(q *query) *query {
	q.Name = r.FoldedName
	return q
}

func (r *fileRecord) byParent(q *query) *query {
	q.Parent = r.FoldedParent
	return q
}

func (r *fileRecord) byPathSuffix(q *query) *query {
	q = r.byParent(q)
	q.PathSuffix = r.PathSuffix
	return q
}

func (r *fileRecord) bySize(q *query) *query {
	q.Size = r.Size()
	return q
}

// If !HasChecksum, equivalent to bySize()
func (r *fileRecord) byChecksum(q *query) *query {
	q = r.bySize(q)
	if r.HasChecksum {
		q.Checksum = r.Checksum
	}
	return q
}

type db struct {
	m map[query]recordSet
}

func newDB() *db {
	return &db{
		m: make(map[query]recordSet),
	}
}

func (d *db) insert(r *fileRecord) {
	var rs recordSet
	var ok bool

	for _, generatorSet := range queryGenerators {
		var q query
		for _, g := range generatorSet {
			g(r, &q)
		}

		if rs, ok = d.m[q]; !ok {
			rs = make(recordSet)
		}
		rs[r] = struct{}{}
		d.m[q] = rs
	}
}

func (d *db) remove(r *fileRecord) {
	for _, generatorSet := range queryGenerators {
		var q query
		for _, g := range generatorSet {
			g(r, &q)
		}

		if rs, ok := d.m[q]; ok {
			delete(rs, r)
		}
	}
}

func (d *db) query(q *query) recordSet {
	return d.m[*q]
}
