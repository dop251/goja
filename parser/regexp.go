package parser

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	WhitespaceChars = " \f\n\r\t\v\u00a0\u1680\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000\ufeff"
	Re2Dot          = "[^\r\n\u2028\u2029]"
)

type regexpParseError struct {
	offset int
	err    string
}

type RegexpErrorIncompatible struct {
	regexpParseError
}
type RegexpSyntaxError struct {
	regexpParseError
}

func (s regexpParseError) Error() string {
	return s.err
}

type _RegExp_parser struct {
	str    string
	length int

	chr       rune // The current character
	chrOffset int  // The offset of current character
	offset    int  // The offset after current character (may be greater than 1)

	err error

	goRegexp   strings.Builder
	passOffset int // In continuous-pass mode: end of unparsed prefix. -1 in mixed mode.

	// consumedEnd tracks position in self.str up to which content has been written (ce_boundary)
	consumedEnd int

	// manuallyInserted counts bytes written to goRegexp that don't correspond to positions in self.str
	mi int

	// consumedEndAtMixEntry tracks consumedEnd at MIXED entry (before stopPassing writes str content).
	// Used by stopPassing PURE to correctly calculate oldMi when goRegexp contains both manual inserts and str content.
	consumedEndAtMixEntry int

	// consumedEndAfterStopPass captures consumedEnd right after stopPassing MIXED/PURE finishes writing, before any subsequent passString calls modify it.
	// This allows ce_boundary to correctly distinguish between self.str bytes written by stopPassing vs. passString.
	consumedEndAfterStopPass int

	dotAll  bool // Enable dotAll mode
	unicode bool
}

// TransformRegExp transforms a JavaScript pattern into  a Go "regexp" pattern.
//
// re2 (Go) cannot do backtracking, so the presence of a lookahead (?=) (?!) or
// backreference (\1, \2, ...) will cause an error.
//
// re2 (Go) has a different definition for \s: [\t\n\f\r ].
// The JavaScript definition, on the other hand, also includes \v, Unicode "Separator, Space", etc.
//
// If the pattern is valid, but incompatible (contains a lookahead or backreference),
// then this function returns an empty string an error of type RegexpErrorIncompatible.
//
// If the pattern is invalid (not valid even in JavaScript), then this function
// returns an empty string and a generic error.
func TransformRegExp(pattern string, dotAll, unicode bool) (transformed string, err error) {

	if pattern == "" {
		return "", nil
	}

	parser := _RegExp_parser{
		str:     pattern,
		length:  len(pattern),
		dotAll:  dotAll,
		unicode: unicode,
	}
	err = parser.parse()
	if err != nil {
		return "", err
	}

	return parser.ResultString(), nil
}

func (self *_RegExp_parser) ResultString() string {
	//		fmt.Fprintf(os.Stderr, "[ResultString] passOffset=%d goRegexp=%q consumedEnd=%d\n", self.passOffset, self.goRegexp.String(), self.consumedEnd)
	if self.passOffset != -1 {
		return self.str[:self.passOffset]
	}
	return self.goRegexp.String()
}

func (self *_RegExp_parser) parse() (err error) {
	self.read() // Pull in the first character
	self.scan()
	return self.err
}

func (self *_RegExp_parser) read() {
	if self.offset < self.length {
		self.chrOffset = self.offset
		chr, width := rune(self.str[self.offset]), 1
		if chr >= utf8.RuneSelf { // !ASCII
			chr, width = utf8.DecodeRuneInString(self.str[self.offset:])
			if chr == utf8.RuneError && width == 1 {
				self.error(true, "Invalid UTF-8 character")
				return
			}
		}
		self.offset += width
		self.chr = chr
	} else {
		self.chrOffset = self.length
		self.chr = -1 // EOF
	}
}

func (self *_RegExp_parser) stopPassing() {
	if self.consumedEnd > 0 || self.goRegexp.Len() > 0 {
		// fmt.Fprintf(os.Stderr, "[stopPassing MIXED] ce=%d po=%d glen=%d mi_before=%d\n", self.consumedEnd, self.passOffset, self.goRegexp.Len(), self.mi)

		// Capture original goRegep length and consumedEnd BEFORE any writes in this call.
		glenBefore := self.goRegexp.Len()
		oldCeAtMixEntry := self.consumedEnd // save before stopPassing writes str content
		// fmt.Fprintf(os.Stderr, "  [MIXED] glen_before=%d old_ce_entry=%d ce_after_stop_prev=%d\n", glenBefore, oldCeAtMixEntry, self.consumedEndAfterStopPass)

		if self.consumedEnd < self.passOffset {
			self.goRegexp.WriteString(self.str[self.consumedEnd:self.passOffset])
		}

		newGlen := self.goRegexp.Len()

		self.consumedEnd = self.passOffset // advance ce_boundary to end of written range in self.str

		// consumedEnd at MIXED entry (before stopPassing writes str content).
		self.consumedEndAtMixEntry = oldCeAtMixEntry // save for PURE to use in oldMi calculation
		// fmt.Fprintf(os.Stderr, "  [MIXED] ce_entry=%d glen_before=%d\n", self.consumedEndAtMixEntry, glenBefore)

		// consumedEnd after stopPassing writes str content (before passString calls since).
		self.consumedEndAfterStopPass = oldCeAtMixEntry + max(self.passOffset-oldCeAtMixEntry, 0) // = passOffset if > oldCe
		// fmt.Fprintf(os.Stderr, "  [MIXED] ce_after_stop=%d\n", self.consumedEndAfterStopPass)

		// Calculate mi: manual bytes = goRegexp content not mapped to self.str positions.
		oldMi := glenBefore - max(oldCeAtMixEntry, 0) // manual bytes from previous passString calls

		if newGlen > self.consumedEnd {
			self.mi = newGlen - self.consumedEnd // new manual inserts added during this stopPassing
		} else if oldCeAtMixEntry > 0 {
			self.mi = max(oldMi, 0) // preserve existing manual bytes from previous calls
		} else {
			self.mi = glenBefore // entering mixed mode: all goRegep content is manual inserts
		}

		// consumedEndAfterStopPass tracks how many str bytes are currently written to goRegexp.
		// Formula: alreadyWritten (goRegexp length) - mi (manual inserts like "\x{") = str bytes in buffer.
		// This ensures oldMi correctly counts all hex sequences across ALL previous passString calls.
		self.consumedEndAfterStopPass = self.goRegexp.Len() - self.mi

		// fmt.Fprintf(os.Stderr, "  [MIXED_ADJ] ce_after_stop=%d mi=%d\n", self.consumedEndAfterStopPass, self.mi)
		// fmt.Fprintf(os.Stderr, "[stopPassing MIXED DONE] ce=%d glen=%d mi=%d\n", self.consumedEnd, newGlen, self.mi)
		return
	}
	alreadyWritten := self.goRegexp.Len() // goRegexp length before stopPassing writes

	// fmt.Fprintf(os.Stderr, "[stopPassing PURE] alr=%d po=%d ce_before=%d mi_before=%d\n", alreadyWritten, self.passOffset, self.consumedEnd, self.mi)

	// Track manual inserts and gap fill for mi preservation.
	oldMi := self.mi
	gapFillLen := 0
	if alreadyWritten < self.passOffset {
		self.goRegexp.WriteString(self.str[alreadyWritten:self.passOffset])
		gapFillLen = self.passOffset - alreadyWritten
		// fmt.Fprintf(os.Stderr, "[stopPassing PURE GAP_FILL] wrote str[%d:%d] len=%d\n", alreadyWritten, self.passOffset, gapFillLen)
	}

	newGlen := self.goRegexp.Len() // length after stopPassing wrote str content

	// consumedEnd tracks how far through self.str has been written to goRegexp.
	// StopPassing writes gapFillLen characters from self.str starting at position (alreadyWritten-oldMi).
	self.consumedEnd = max(self.consumedEnd, alreadyWritten-oldMi+gapFillLen)

	// fmt.Fprintf(os.Stderr, "[stopPassing PURE PREADJ] ce=%d oldMi=%d mi_before=%d\n", self.consumedEnd, oldMi, self.mi)

	// Preserve old mi if only str content was written (no new hex escapes).
	if gapFillLen > 0 && gapFillLen == newGlen-alreadyWritten {
		self.mi = oldMi // only str content added, keep existing manual byte count
	} else {
		self.mi = max(0, newGlen-self.consumedEnd)
	}

	// fmt.Fprintf(os.Stderr, "[stopPassing PURE DONE] ce=%d glen=%d mi=%d consumedEnd-mi=%d\n", self.consumedEnd, newGlen, self.mi, self.consumedEnd-self.mi)
	self.passOffset = -1
}
func (self *_RegExp_parser) write(p []byte) {
	if self.passOffset != -1 {
		self.stopPassing()
	}
	self.goRegexp.Write(p)
}

func (self *_RegExp_parser) writeByte(b byte) {
	if self.passOffset != -1 {
		self.stopPassing()
	}
	self.goRegexp.WriteByte(b)
}

func (self *_RegExp_parser) writeString(s string) {
	if self.passOffset != -1 {
		self.stopPassing()
	}
	self.goRegexp.WriteString(s)
}

func (self *_RegExp_parser) scan() {
	for self.chr != -1 {
		switch self.chr {
		case '\\':
			self.read()
			self.scanEscape(false)
		case '(':
			self.pass()
			self.scanGroup()
		case '[':
			self.scanBracket()
		case ')':
			self.error(true, "Unmatched ')'")
			return
		case '.':
			if self.dotAll {
				self.pass()
				break
			}
			self.writeString(Re2Dot)
			self.read()
		default:
			self.pass()
		}
	}
}

// (...)
func (self *_RegExp_parser) scanGroup() {
	str := self.str[self.chrOffset:]
	if len(str) > 1 { // A possibility of (?= or (?!
		if str[0] == '?' {
			ch := str[1]
			switch {
			case ch == '=' || ch == '!':
				self.error(false, "re2: Invalid (%s) <lookahead>", self.str[self.chrOffset:self.chrOffset+2])
				return
			case ch == '<':
				if len(str) > 2 && (str[2] == '=' || str[2] == '!') {
					self.error(false, "re2: Invalid (%s) <lookbehind>", self.str[self.chrOffset:self.chrOffset+2])
					return
				}
				self.pass()         // ?
				self.writeByte('P') // older Go versions compatibility
				self.pass()         // <
				self.scanGroupName()
			case ch != ':':
				self.error(true, "Invalid group")
				return
			}
		}
	}
	for self.chr != -1 && self.chr != ')' {
		switch self.chr {
		case '\\':
			self.read()
			self.scanEscape(false)
		case '(':
			self.pass()
			self.scanGroup()
		case '[':
			self.scanBracket()
		case '.':
			if self.dotAll {
				self.pass()
				break
			}
			self.writeString(Re2Dot)
			self.read()
		default:
			self.pass()
			continue
		}
	}
	if self.chr != ')' {
		self.error(true, "Unterminated group")
		return
	}
	self.pass()
}

func (self *_RegExp_parser) scanGroupName() {
	supported := true
	if !(self.chr >= 'a' && self.chr <= 'z' || self.chr >= 'A' && self.chr <= 'Z' && self.chr == '_') {
		supported = false
	}
	for self.chr != -1 && self.chr != '>' {
		if !(self.chr >= 'a' && self.chr <= 'z' || self.chr >= 'A' && self.chr <= 'Z' && self.chr == '_' || self.chr >= '0' && self.chr <= '9') {
			supported = false
		}
		self.pass()
	}
	if !supported {
		self.error(false, "Unsupported group name")
		return
	}
	self.pass()
}

// [...]
func (self *_RegExp_parser) scanBracket() {
	str := self.str[self.chrOffset:]
	if strings.HasPrefix(str, "[]") {
		// [] -- Empty character class
		self.writeString("[^\u0000-\U0001FFFF]")
		self.offset += 1
		self.read()
		return
	}

	if strings.HasPrefix(str, "[^]") {
		self.writeString("[\u0000-\U0001FFFF]")
		self.offset += 2
		self.read()
		return
	}

	self.pass()
	for self.chr != -1 {
		if self.chr == ']' {
			break
		} else if self.chr == '\\' {
			self.read()
			self.scanEscape(true)
			continue
		}
		self.pass()
	}
	if self.chr != ']' {
		self.error(true, "Unterminated character class")
		return
	}
	self.pass()
}

// \...
func (self *_RegExp_parser) scanEscape(inClass bool) {
	offset := self.chrOffset
	//	fmt.Fprintf(os.Stderr, "[scanEscape] inClass=%v offset=%d chrOffset=%d goRegexp=%q passOffset=%d self.str=%q\n", inClass, offset, self.chrOffset, self.goRegexp.String(), self.passOffset, self.str)

	var length, base uint32
	switch self.chr {

	case '0', '1', '2', '3', '4', '5', '6', '7':
		var value int64
		size := 0
		for {
			digit := int64(digitValue(self.chr))
			if digit >= 8 {
				// Not a valid digit
				break
			}
			value = value*8 + digit
			self.read()
			size += 1
		}
		if size == 1 { // The number of characters read
			if value != 0 {
				// An invalid backreference
				self.error(false, "re2: Invalid \\%d <backreference>", value)
				return
			}
			self.passString(offset-1, self.chrOffset)
			return
		}
		tmp := []byte{'\\', 'x', '0', 0}
		if value >= 16 {
			tmp = tmp[0:2]
		} else {
			tmp = tmp[0:3]
		}
		tmp = strconv.AppendInt(tmp, value, 16)
		self.write(tmp)
		return

	case '8', '9':
		self.read()
		self.error(false, "re2: Invalid \\%s <backreference>", self.str[offset:self.chrOffset])
		return

	case 'x':
		self.read()
		length, base = 2, 16

	case 'u':
		self.read()
		if self.chr == '{' && self.unicode {
			self.read()
			length, base = 0, 16
		} else {
			length, base = 4, 16
		}

	case 'b':
		if inClass {
			self.write([]byte{'\\', 'x', '0', '8'})
			self.read()
			return
		}
		fallthrough

	case 'B':
		fallthrough

	case 'd', 'D', 'w', 'W':
		// This is slightly broken, because ECMAScript
		// includes \v in \s, \S, while re2 does not
		fallthrough

	case '\\':
		fallthrough

	case 'f', 'n', 'r', 't', 'v':
		self.passString(offset-1, self.offset)
		self.read()
		return

	case 'c':
		self.read()
		var value int64
		if 'a' <= self.chr && self.chr <= 'z' {
			value = int64(self.chr - 'a' + 1)
		} else if 'A' <= self.chr && self.chr <= 'Z' {
			value = int64(self.chr - 'A' + 1)
		} else {
			self.writeByte('c')
			return
		}
		tmp := []byte{'\\', 'x', '0', 0}
		if value >= 16 {
			tmp = tmp[0:2]
		} else {
			tmp = tmp[0:3]
		}
		tmp = strconv.AppendInt(tmp, value, 16)
		self.write(tmp)
		self.read()
		return
	case 's':
		if inClass {
			self.writeString(WhitespaceChars)
		} else {
			self.writeString("[" + WhitespaceChars + "]")
		}
		self.read()
		return
	case 'S':
		if inClass {
			self.error(false, "S in class")
			return
		} else {
			self.writeString("[^" + WhitespaceChars + "]")
		}
		self.read()
		return
	case 'k':
		// The rules are too complicated to implement here, so we pass it on to regexp2
		self.error(false, "named group back-reference")
		return
	default:
		// $ is an identifier character, so we have to have
		// a special case for it here
		if self.chr == '$' || self.chr < utf8.RuneSelf && !isIdentifierPart(self.chr) {
			// A non-identifier character needs escaping
			self.passString(offset-1, self.offset)
			self.read()
			return
		}
		// For \p{...} Unicode property escapes, preserve the backslash since Go regexp
		// requires it for \p{} support. Other unrecognized identifier chars (like \a in \abc)
		// have their backslash stripped to match ECMAScript semantics where
		// /\z/.test("z") === true and /\a/.test("a") === true.
		if self.chr == 'p' && self.offset < self.length && self.str[self.offset] == '{' {
			savedOffset := self.offset
			self.passString(offset-1, savedOffset+1) // writes "\p{" (including the opening brace)
			self.read()                              // advances past '{', self.chr = first char inside
			// Validate property name: only [a-zA-Z0-9_] allowed between braces.
			if self.chr == '}' {
				self.error(false, "Invalid Unicode property name")
				return
			}
			for self.chr != '}' && self.chr != -1 {
				valid := (self.chr >= 'a' && self.chr <= 'z') || (self.chr >= 'A' && self.chr <= 'Z') || (self.chr >= '0' && self.chr <= '9') || self.chr == '_'
				if !valid {
					self.error(false, "Invalid Unicode property name")
					return
				}
				self.read()
			}
			endOffset := self.offset
			if self.chr != '}' {
				self.error(true, "Unterminated Unicode property escape")
				return
			}
			// Capture the remaining content inside braces (opening brace already written).
			// endOffset = self.offset after reading '}', so subtract 1 to exclude it from passString range.
			self.passString(savedOffset+1, endOffset-1)
		} else {
			self.pass()
		}
		return
	}

	// Otherwise, we're a \u.... or \x...
	valueOffset := self.chrOffset // offset of first character consumed by case 'x'/'u' read
	hexDigitsConsumed := 0
	if length == 0 {
		// \u{...}: case '' read at line 348 consumed '{', then line 350 read consumed first hex digit.
		// chrOffset points to first hex digit, but the while loop will re-read it (it's in self.chr).
		// We need valueOffset to point one position before first hex so passString includes it.
		valueOffset--
	}

	if length > 0 {
		for length := length; length > 0; length-- {
			digit := uint32(digitValue(self.chr))
			if digit >= base {
				// Not a valid digit
				goto skip
			}
			self.read()
		}
	} else {
		for self.chr != '}' && self.chr != -1 {
			digit := uint32(digitValue(self.chr))
			if digit >= base {
				// Not a valid digit
				self.error(true, "Invalid Unicode escape")
				return
			}
			self.read()
			hexDigitsConsumed++
		}
	}

	if length == 4 || length == 0 {
		self.write([]byte{
			'\\',
			'x',
			'{',
		})
		if length == 0 {
			// \u{...}: we already consumed the '{' and wrote '\\x{' to goRegexp.
			// chrOffset points past last hex digit; subtract digits consumed + 1 for brace position.
			valueOffset = self.chrOffset - hexDigitsConsumed
		}
		self.passString(valueOffset, self.chrOffset)
		self.read() // advance past stale chr so scan() doesn't re-process it
		if length != 0 {
			self.writeByte('}')
		}
	} else if length == 2 {
		self.passString(offset-1, valueOffset+2)
	} else {
		// Should never, ever get here...
		self.error(true, "re2: Illegal branch in scanEscape")
		return
	}

	return

skip:
	self.passString(offset, self.chrOffset)
	self.read() // advance past stale chr so scan() doesn't re-process it
	// fmt.Fprintf(os.Stderr, "[scanEscape RETURN] chr=%c offset=%d goRegexp=%q\n", self.chr, self.offset, self.goRegexp.String())
}

func (self *_RegExp_parser) pass() {
	//	fmt.Fprintf(os.Stderr, "[pass] chr=%c(%04x) chrOffset=%d offset=%d passOffset=%d\n", self.chr, self.chr, self.chrOffset, self.offset, self.passOffset)
	if self.passOffset == self.chrOffset {
		self.passOffset = self.offset
	} else {
		if self.passOffset != -1 {
			self.stopPassing()
		}
		if self.chr != -1 {
			self.goRegexp.WriteRune(self.chr)
		}
	}
	//	fmt.Fprintf(os.Stderr, "[pass DONE] goRegexp=%q passOffset=%d\n", self.goRegexp.String(), self.passOffset)
	self.read()
}

func (self *_RegExp_parser) passString(start, end int) {
	if self.passOffset == start || (self.passOffset == -1 && start == 0) {
		self.passOffset = end
		// fmt.Fprintf(os.Stderr, "[passString cont] ce=%d mi_before=%d glen=%d po_after=%d\n", self.consumedEnd, self.mi, self.goRegexp.Len(), self.passOffset)
		if self.consumedEnd == 0 && self.passOffset != start {
			// Entering mixed mode: record manually inserted bytes (content in goRegexp that doesn't correspond to positions in self.str)
			self.mi = self.goRegexp.Len()
			// fmt.Fprintf(os.Stderr, "[passString mixed] mi=%d glen=%d\n", self.mi, self.goRegexp.Len())

			// consumedEndAfterStopPass tracks how many str bytes would be written if stopPassing ran now.
			// In continuous pass: goRegexp only contains manual inserts; all remaining is from str [0..passOffset).
			self.consumedEndAfterStopPass = self.goRegexp.Len() - self.mi
		}
		self.offset = end // Only advance offset in continuous-pass mode
		return
	}
	if self.passOffset != -1 {
		self.stopPassing()
		// fmt.Fprintf(os.Stderr, "[passString POST-STOP-MIXED] glen=%d mi=%d ce_boundary=%d consumedEnd=%d\n", self.goRegexp.Len(), self.mi, self.consumedEnd, self.consumedEnd)
	}
	// fmt.Fprintf(os.Stderr, "[passString POST-STOP] glen=%d mi=%d ce_boundary=%d\n", self.goRegexp.Len(), self.mi, self.consumedEnd)

	// Fill gap from consumedEnd to start (stopPassing may have written up to passOffset).
	if alreadyWritten := self.consumedEnd; start < alreadyWritten {
		start = alreadyWritten // Skip overlap with what's already written
	}

	// fmt.Fprintf(os.Stderr, "[passString] writing str[%d:%d]=%q ce=%d\n", start, end, self.str[start:end], self.consumedEnd)
	if start < end {
		self.goRegexp.WriteString(self.str[start:end])
	}
	// Update consumedEnd to track furthest position in self.str written so far.
	// mi tracks manual bytes (hex escapes like "\x{") that are in goRegexp but not from self.str.
	// After this call, consumedEnd = how much of self.str is covered by goRegexp content.
	self.consumedEnd = max(self.consumedEnd, end)
	self.mi = max(0, self.goRegexp.Len()-self.consumedEnd)
	// fmt.Fprintf(os.Stderr, "[passString END] ce=%d glen=%d mi=%d consumedEnd-mi=%d\n", self.consumedEnd, self.goRegexp.Len(), self.mi, self.consumedEnd-self.mi)
	self.offset = end // After writing, ensure read() continues from 'end' not stopPassing's old position
}

func (self *_RegExp_parser) error(fatal bool, msg string, msgValues ...interface{}) {
	if self.err != nil {
		return
	}
	e := regexpParseError{
		offset: self.offset,
		err:    fmt.Sprintf(msg, msgValues...),
	}
	if fatal {
		self.err = RegexpSyntaxError{e}
	} else {
		self.err = RegexpErrorIncompatible{e}
	}
	self.offset = self.length
	self.chr = -1
}
