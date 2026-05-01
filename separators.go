package ihttp

import "sort"

// Separator used in args as Items after the URL argument.
const (
	SepHeader      = ":"
	SepHeaderEmpty = ";"

	//SepCredentials           = ":"
	//SepProxy                 = ":"
	//SepHeaderEmbed           = ":@"

	SepDataString = "="

	SepDataRawJSON = ":="

	SepFileUpload = "@"

	//SepFileUploadType        = ";type=" // in already parsed file upload path only
	//SepDataEmbedFileContents = "=@"
	//SepDataEmbedRawJSONFile  = ":=@"

	SepQueryParam = "=="

	//SepQueryEmbedFile        = "==@"
)

// SepsGroupNestedJSONItems return separators for nested JSON data type Items.
func SepsGroupNestedJSONItems() []string {
	return sortSeps([]string{
		SepDataString,
		SepDataRawJSON,
		//SepDataEmbedFileContents,
		//SepDataEmbedRawJSONFile,
	})
}

// SepsGroupDataItems return separators for data type Items.
func SepsGroupDataItems() []string {
	return sortSeps([]string{
		SepDataString,
		SepDataRawJSON,
		SepFileUpload,
		//SepDataEmbedFileContents,
		//SepDataEmbedRawJSONFile,
	})
}

// SepsGroupAllItems return separators for all types of Items.
func SepsGroupAllItems() []string {
	return sortSeps([]string{
		SepHeader,
		SepHeaderEmpty,
		//SepHeaderEmbed,
		SepQueryParam,
		//SepQueryEmbedFile,
		SepDataString,
		SepDataRawJSON,
		SepFileUpload,
		//SepDataEmbedFileContents,
		//SepDataEmbedRawJSONFile,
	})
}

// sortSeps sorts separators by length in descending order
// to ensure that longer separators are matched before shorter
// ones during tokenization.
func sortSeps(seps []string) []string {
	out := make([]string, len(seps))
	copy(out, seps)
	sort.Slice(out, func(i, j int) bool {
		return len(out[i]) > len(out[j])
	})
	return out
}
