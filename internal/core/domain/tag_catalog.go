package domain

// DistinctTagsResult is the sorted distinct tag strings visible in the database (Hexxla [Tx.ListExistingTopics] semantics).
type DistinctTagsResult struct {
	Tags []string `json:"tags"`
}

// TagCountsResult holds per-tag visible cell counts, typically sorted by count descending ([Tx.TagCounts]).
type TagCountsResult struct {
	Counts []TagCountEntry `json:"counts"`
}

// TagCountEntry is one tag plus how many visible cells carry it.
type TagCountEntry struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}
