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

	if strings.Compare(a.GetName(), b.GetName()) == 0 {
		return 1
	}

	index *= StringSimilarity(a.GetName(), b.GetName())

	// Too heavy network wise
	// verA, verB := ExtractVersions(a), ExtractVersions(b)
	// index *= MatchingComparison(verA, verB)

	if a, ok := a.(PluginMetadata); ok {
		if b, ok := b.(PluginMetadata); ok {
			// Kinda heavy comparison(too much variance)
			// index *= StringSimilarity(a.GetDescription(), b.GetDescription())
			// index *= StringSimilarity(a.GetWebsite(), b.GetWebsite())

			index *= MatchingComparison(splitAuthors(a.GetAuthors()), splitAuthors(b.GetAuthors()))
		}
	}

	return index
}

func MatchingComparison(a, b []string) float64 {
	var index float64 = 1

	if len(a) > len(b) {
		a, b = b, a
	}

	// The strings are compared in pairs, the most similar one is selected

	maxes := make([]int, 0)
	for _, authA := range a { // Cycle every strings in A
		/* if strings.Contains(authA, "https://") || strings.Contains(authA, "http://") {
			continue
		} */

		maxindex := 0
		var max float64 = -1
		for i, authB := range b { // Check every strings in B
			if slices.Contains(maxes, i) { // Ignore previously paired strings
				continue
			}

			// The most similar one is selected
			tmax := math.Max(max, LevenshteinIndex(authA, authB))
			if tmax != max {
				maxindex = i
				max = tmax
			}

			if max >= 1 {
				break
			}
		}

		// This string in A is paired with the one in B and
		// its similarity index is averaged to the final product
		index = (2*max + index) / 3

		// After pairing these strings, remove them for subsequent pairings
		maxes = append(maxes, maxindex)
	}

	// coeff := (float64(len(a)) / float64(len(b)))
	// index = (6*index + coeff) / 7

	return index
}

func ExtractVersions(p Plugin) []string {
	if p, ok := p.(Versionable); ok {
		return []string{p.GetVersion()}
	}

	if p, ok := p.(RemotePlugin); ok {
		ver, err := GetVersionNames(p)
		if err != nil {
			return []string{}
		}

		return ver
	}

	return []string{}
}

// LevenshteinIndex computes the inverse of the Levenshtein distance normalized between 0 and 1
func LevenshteinIndex(a, b string) float64 {
	tot := float64(lvh.Compare(a, b).EditDist)
	uncased := float64(lvh.Compare(strings.ToLower(a), strings.ToLower(b)).EditDist)

	cased := tot - uncased

	return 1 - float64(uncased+0.8*cased)/float64(max(len(a), len(b)))
}

func StringSimilarity(a, b string) float64 {
	if math.Abs(float64(len(a))-float64(len(b)))/math.Abs(float64(len(a))) > 0.7 {
		return ShiftSimilarity(a, b)
	}

	return LevenshteinIndex(a, b)
}

func ShiftSimilarity(a, b string) float64 {
	if len(a) < len(b) {
		a, b = b, a
	}

	var index float64 = 0
	for i := 0; i < len(a)-len(b)+1; i++ {
		index = max(index, LevenshteinIndex(a[i:i+len(b)], b))
	}

	index = (2*index + LevenshteinIndex(a, b)) / 3

	return index
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
