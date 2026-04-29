package domain

// Retrieval hints steer LLM tool chaining: ANN/lexical return entry points; context pack expands on the lattice.

const (
	// RetrievalHintLexicalOrANN tells models that hits are partial and how to widen context.
	RetrievalHintLexicalOrANN = "These hits are ranked matches only (top-K). They may omit neighbouring turns, contradictions resolved by seams, or facts outside this slice. If the user question is not fully answered, call mosaic_hexxla_load_context_pack with one or more seed coordinates {q,r} from the hits above (max_ring + max_tokens as needed) to assemble hex-neighbourhood context under budget."

	// RetrievalHintAfterContextPack reminds when to use retrieval vs expansion (optional on pack responses).
	RetrievalHintAfterContextPack = "This pack is lattice-expanded from seeds (nearby cells, optional seams/supersession). For global semantic discovery across the DB, use mosaic_hexxla_search_embedding; for tag/text filters use mosaic_hexxla_query_cells or mosaic_hexxla_search_cells, then re-load a context pack from new coords if needed."
)
