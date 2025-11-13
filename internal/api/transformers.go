package api

import "github.com/danielgtaylor/huma/v2"

// Transformers returns all response transformers used by the API.
// Transformers modify responses after handlers execute but before serialization.
// They are registered globally in the Huma config and run on all API responses.
//
// IMPORTANT: Order matters. Transformers execute sequentially, with each transformer's
// output becoming the next transformer's input. If you need to compose transformations,
// ensure they are ordered correctly in the returned slice.
//
// Each transformer should be defensive and check response types before operating,
// passing through responses it doesn't handle.
//
// Current transformers:
//   - toolFieldSelectTransformer: Filters tool responses based on ?detail= query parameter.
func Transformers() []huma.Transformer {
	return []huma.Transformer{
		toolFieldSelectTransformer,
	}
}
