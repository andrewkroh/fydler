package main

import (
	"github.com/andrewkroh/fydler/internal/analysis/conflict"
	"github.com/andrewkroh/fydler/internal/analysis/duplicate"
	"github.com/andrewkroh/fydler/internal/analysis/dynamicfield"
	"github.com/andrewkroh/fydler/internal/analysis/invalidattribute"
	"github.com/andrewkroh/fydler/internal/analysis/missingtype"
	"github.com/andrewkroh/fydler/internal/analysis/nesting"
	"github.com/andrewkroh/fydler/internal/analysis/unknownattribute"
	"github.com/andrewkroh/fydler/internal/analysis/useecs"
	"github.com/andrewkroh/fydler/internal/fydler"
)

func main() {
	fydler.Main(
		conflict.Analyzer,
		duplicate.Analyzer,
		dynamicfield.Analyzer,
		invalidattribute.Analyzer,
		missingtype.Analyzer,
		nesting.Analyzer,
		unknownattribute.Analyzer,
		useecs.Analyzer,
	)
}
