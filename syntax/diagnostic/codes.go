// Central public diagnostic registry. Never reuse a code for another category.
package diagnostic

const (
	OSFailure            = "BX4109"
	SecurityFailure      = "BX4110"
	IndexRange           = "BX4014"
	UserThrown           = "BX4015"
	IteratorProtocol     = "BX4016"
	InterfaceMismatch    = "BX3012"
	InheritanceOrder     = "BX4017"
	ShiftRange           = "BX4018"
	SourceRead           = "BX0001"
	UnknownCharacter     = "BX1001"
	UnclosedString       = "BX1002"
	UnclosedComment      = "BX1003"
	InvalidUTF8          = "BX1004"
	UnexpectedToken      = "BX2001"
	ExpectedExpression   = "BX2002"
	InvalidTarget        = "BX2003"
	NestingLimit         = "BX2004"
	ClassSyntax          = "BX2005"
	Delimiter            = "BX2006"
	MissingComma         = "BX2007"
	ObjectField          = "BX2008"
	SelfBinding          = "BX2009"
	DuplicateBinding     = "BX2010"
	RestPosition         = "BX2011"
	DuplicateKey         = "BX2012"
	DuplicateConstructor = "BX2013"
	UndefinedName        = "BX3001"
	MissingMember        = "BX3002"
	NativeFailure        = "BX4001"
	InvalidReceiver      = "BX4002"
	ArgumentCount        = "BX4003"
	ArgumentType         = "BX4004"
	NotCallable          = "BX4005"
	InvalidSelf          = "BX4006"
	InvalidBase          = "BX4007"
	ImportResolution     = "BX4008"
	MissingImportName    = "BX4009"
	InputFailure         = "BX4010"
	DestructureMissing   = "BX4011"
	InvalidClass         = "BX4012"
	TaskFailure          = "BX4013"
	FilesFailure         = "BX4101"
	TimeFailure          = "BX4102"
	NetworkFailure       = "BX4103"
	CollectionsFailure   = "BX4104"
	SerializationFailure = "BX4105"
	MathFailure          = "BX4106"
	ProcessFailure       = "BX4107"
	StringsFailure       = "BX4108"
)

// Codes returns a copy of the code-to-meaning registry.
func Codes() map[string]string {
	return map[string]string{
		OSFailure:            "Operating system operation failed",
		SecurityFailure:      "Security or encoding operation failed",
		IndexRange:           "Indexed assignment out of range",
		UserThrown:           "User-thrown value",
		IteratorProtocol:     "Invalid iterator protocol result",
		InterfaceMismatch:    "Missing or incompatible interface method",
		InheritanceOrder:     "Inconsistent multiple inheritance order",
		ShiftRange:           "Bitwise shift count outside 0 through 63",
		SourceRead:           "Source file read failure",
		UnknownCharacter:     "Unknown source character",
		UnclosedString:       "Unterminated string",
		UnclosedComment:      "Unterminated block comment",
		InvalidUTF8:          "Invalid UTF-8 source",
		UnexpectedToken:      "Missing or unexpected syntax token",
		ExpectedExpression:   "Missing or unexpected expression",
		InvalidTarget:        "Invalid assignment or destructuring target",
		NestingLimit:         "Syntax nesting limit exceeded",
		ClassSyntax:          "Invalid class body",
		Delimiter:            "Unmatched or misplaced delimiter",
		MissingComma:         "Missing list separator comma",
		ObjectField:          "Invalid object field syntax",
		SelfBinding:          "Invalid self binding or parameter",
		DuplicateBinding:     "Duplicate destructuring binding",
		RestPosition:         "Rest binding must be last",
		DuplicateKey:         "Duplicate destructuring key",
		DuplicateConstructor: "Duplicate class constructor",
		UndefinedName:        "Undefined variable or function name",
		MissingMember:        "Undefined object member",
		NativeFailure:        "Unclassified native operation failure",
		InvalidReceiver:      "Operation unavailable for receiver type",
		ArgumentCount:        "Incorrect argument count",
		ArgumentType:         "Incorrect argument or destructuring input type",
		NotCallable:          "Attempt to call a non-callable value",
		InvalidSelf:          "Invalid runtime self usage",
		InvalidBase:          "Class base is not a class",
		ImportResolution:     "Import path or package could not be resolved",
		MissingImportName:    "Requested name absent from imported module",
		InputFailure:         "Standard input read failure",
		DestructureMissing:   "Required destructuring element or field absent",
		InvalidClass:         "Invalid runtime class definition",
		TaskFailure:          "Unexpected task worker failure",
		FilesFailure:         "File or filesystem operation failed",
		TimeFailure:          "Time operation or range failed",
		NetworkFailure:       "Network request or response failed",
		CollectionsFailure:   "Collection operation or range failed",
		SerializationFailure: "Serialization or deserialization failed",
		MathFailure:          "Math domain or result failure",
		ProcessFailure:       "Process or environment operation failed",
		StringsFailure:       "Standard string operation failed",
	}
}
