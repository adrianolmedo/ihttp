package ihttp

// Separator used in args as Items after the URL argument.
const (
	SepHeader      = ":"
	SepHeaderEmpty = ";"

	//SepCredentials           = ":"
	//SepProxy                 = ":"
	//SepHeaderEmbed           = ":@"

	SepDataString = "="

	SepDataRawJSON = ":="

	SepFileUpload = "@" // NOTE: Enabled to satisfy internal Parser tests

	//SepFileUploadType        = ";type=" // in already parsed file upload path only
	//SepDataEmbedFileContents = "=@"
	//SepDataEmbedRawJSONFile  = ":=@"

	SepQueryParam = "=="

	//SepQueryEmbedFile        = "==@"
)

// SepsGroupDataItems return separators for data type Items.
func SepsGroupDataItems() []string {
	return []string{
		SepDataString,
		SepDataRawJSON,
		SepFileUpload,
		//SepDataEmbedFileContents,
		//SepDataEmbedRawJSONFile,
	}
}

// SepsGroupAllItems return separators for all types of Items.
func SepsGroupAllItems() []string {
	return []string{
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
	}
}
