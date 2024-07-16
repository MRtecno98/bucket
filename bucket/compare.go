package bucket

import (
	"math"
	"strings"

	"github.com/gnames/levenshtein"
	"golang.org/x/exp/slices"
)

var lvh = levenshtein.NewLevenshtein()

func ComparisonIndex(a, b Plugin) float64 {
	var index float64 = 1

	index *= StringSimilarity(a.GetName(), b.GetName())

	if a, ok := a.(PluginMetadata); ok {
		if b, ok := b.(PluginMetadata); ok {
			// Kinda heavy comparison(too much variance)
			// index *= StringSimilarity(a.GetDescription(), b.GetDescription())
			// index *= StringSimilarity(a.GetWebsite(), b.GetWebsite())

			listA, listB := splitAuthors(a.GetAuthors()), splitAuthors(b.GetAuthors())

			if len(listA) > len(listB) {
				listA, listB = listB, listA
			}

			maxes := make([]int, 0)
			for _, authA := range listA { // Cycle every author in A
				maxindex := 0
				var max float64 = -1
				for i, authB := range listB { // Check every author in B
					if slices.Contains(maxes, i) { // Ignore previously paired authors
						continue
					}

					// The most similar one is selected
					tmax := math.Max(max, StringSimilarity(authA, authB))
					if tmax != max {
						maxindex = i
						max = tmax
					}

					if max >= 1 {
						break
					}
				}

				// This author in A is paired with the one in B and
				// its similarity index is added to the final product
				index *= max

				// After pairing these authors, remove them for subsequent pairings
				maxes = append(maxes, maxindex)
			}
		}
	}

	return index
}

// Inverse of the Levenshtein distance normalized between 0 and 1
func StringSimilarity(a, b string) float64 {
	return 1 - float64(lvh.Compare(a, b).EditDist)/float64(max(len(a), len(b)))
}

// Some people write multiple authors in a single string
// apparently, and that would be a problem for comparison
func splitAuthors(authors []string) []string {
	var res []string
	for _, author := range authors {
		spl := strings.Split(author, ",")
		for _, s := range spl {
			res = append(res, strings.TrimSpace(s))
		}
	}

	return res
}
