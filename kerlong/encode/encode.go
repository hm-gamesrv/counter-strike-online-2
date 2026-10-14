package encode

import (
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
)

// localEncoding is the client's code page, configured with the CODE_PAGE
// environment variable. It normally follows LOCALE, since the encoding is a
// property of the game client's language. A nil value means "the client speaks
// UTF-8" and conversion is a no-op.
//
// This replaces the previous cgo-based iconv converter with pure-Go code from
// golang.org/x/text, so the server cross-compiles and runs in minimal images.
var localEncoding encoding.Encoding

// supportedCodePages maps the names accepted by CODE_PAGE onto encodings.
var supportedCodePages = map[string]encoding.Encoding{
	"gbk":       simplifiedchinese.GBK,
	"cp936":     simplifiedchinese.GBK,
	"936":       simplifiedchinese.GBK,
	"gb2312":    simplifiedchinese.GBK,
	"gb18030":   simplifiedchinese.GB18030,
	"big5":      traditionalchinese.Big5,
	"cp950":     traditionalchinese.Big5,
	"950":       traditionalchinese.Big5,
	"euc-kr":    korean.EUCKR,
	"euckr":     korean.EUCKR,
	"cp949":     korean.EUCKR,
	"949":       korean.EUCKR,
	"shift_jis": japanese.ShiftJIS,
	"shift-jis": japanese.ShiftJIS,
	"sjis":      japanese.ShiftJIS,
	"cp932":     japanese.ShiftJIS,
	"932":       japanese.ShiftJIS,
}

// InitConverter configures the converter for the given code page name.
//
// It panics on an unsupported name, which is deliberate: silently serving a
// client with the wrong code page corrupts every player name and chat message,
// and that is far harder to diagnose than a startup failure.
func InitConverter(local string) bool {
	name := strings.ToLower(strings.TrimSpace(local))

	// UTF-8 clients need no conversion at all.
	if name == "" || name == "utf-8" || name == "utf8" {
		localEncoding = nil
		return true
	}

	enc, ok := supportedCodePages[name]
	if !ok {
		panic("unsupported CODE_PAGE \"" + local + "\"; supported values are: " +
			"gbk, gb18030, big5, euc-kr, shift_jis, utf-8")
	}

	localEncoding = enc
	return true
}

// Utf8ToLocal converts a UTF-8 string into the client's code page.
func Utf8ToLocal(str string) (string, error) {
	if localEncoding == nil {
		return str, nil
	}
	return localEncoding.NewEncoder().String(str)
}

// LocalToUtf8 converts a string in the client's code page into UTF-8.
func LocalToUtf8(str string) (string, error) {
	if localEncoding == nil {
		return str, nil
	}
	return localEncoding.NewDecoder().String(str)
}

// GbkToUtf8 converts GBK-encoded bytes to UTF-8.
func GbkToUtf8(str []byte) ([]byte, error) {
	return simplifiedchinese.GBK.NewDecoder().Bytes(str)
}

// Utf8ToGbk converts UTF-8 bytes to GBK.
func Utf8ToGbk(str []byte) ([]byte, error) {
	return simplifiedchinese.GBK.NewEncoder().Bytes(str)
}
