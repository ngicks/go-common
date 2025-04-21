package exver

import (
	"cmp"
	"encoding"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	componentMax = 9999
)

// Core represents only numeric part of Version.
type Core struct {
	component [4]uint16
	length    int
}

func NewCore(nums []uint16) (Core, error) {
	if len(nums) == 0 || len(nums) > 4 {
		return Core{}, fmt.Errorf("input must be larger than 1 and be less than or equals to 4")
	}

	var component [4]uint16
	for i, num := range nums {
		if num > componentMax {
			return Core{}, fmt.Errorf("%q too large: larger than %d", componentName(i), componentMax)
		}
		component[i] = num
	}

	return Core{
		component: component,
		length:    len(nums),
	}, nil
}

func ParseCore(s string) (Core, error) {
	a, b, c, d, s, err := version(s)
	if err != nil {
		return Core{}, err
	}

	core, err := convertCore(a, b, c, d)
	if err != nil {
		return core, err
	}

	if len(s) > 0 {
		return Core{}, fmt.Errorf("extra string %q after version core", s)
	}

	return core, nil
}

func convertCore(a, b, c, d int64) (Core, error) {
	rawComponent := [4]int64{a, b, c, d}
	var comopnent [4]uint16
	var i int
	// for-range-int works a bit differently.
	// It will skip last increament (stops at 3 even if the iteration was not break-ed)
	for i = 0; i < 4; i++ {
		if rawComponent[i] < 0 {
			break
		}
		if rawComponent[i] > componentMax {
			return Core{}, fmt.Errorf("%q too large: larger than %d", componentName(i), componentMax)
		}
		comopnent[i] = uint16(rawComponent[i])
	}

	return Core{component: comopnent, length: i}, nil
}

func (c Core) Component() [4]uint16 {
	return c.component
}

func (c Core) Nums() []uint {
	out := make([]uint, c.length)
	for i := range c.length {
		out[i] = uint(c.component[i])
	}
	return out
}

func (c Core) Int64() int64 {
	var out int64
	for i := range c.length {
		mult := int64(1)
		for range 3 - i {
			mult *= 10000
		}
		out += int64(c.component[i]) * mult
	}
	return out
}

func (c Core) String() string {
	if c == (Core{}) {
		return "0.0.0"
	}
	var builder strings.Builder
	for i := range c.length {
		if i > 0 {
			builder.WriteByte('.')
		}
		builder.WriteString(strconv.FormatUint(uint64(c.component[i]), 10))
	}
	return builder.String()
}

var (
	_ encoding.TextMarshaler   = Core{}
	_ encoding.TextUnmarshaler = (*Core)(nil)
)

func (c Core) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c *Core) UnmarshalText(text []byte) error {
	core, err := ParseCore(string(text))
	if err != nil {
		return err
	}
	*c = core
	return nil
}

var (
	_ json.Marshaler   = Core{}
	_ json.Unmarshaler = (*Core)(nil)
)

func (c Core) MarshalJSON() ([]byte, error) {
	// No need to escape: only numeric tokens are permitted.
	return []byte("\"" + c.String() + "\""), nil
}

func (c *Core) UnmarshalJSON(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("too short")
	}
	return c.UnmarshalText(data[1 : len(data)-1])
}

// Compare returns
//
//	-1 if c is less than u,
//	 0 if c equals u,
//	+1 if c is greater than u.
//
// Missing parts are always treated as 0.
// e.g. comparing 1 and 1.0 returns 0, 1.0.0.2 and 1.0 returns +1.
func (c Core) Compare(u Core) int {
	for i := range 4 {
		if c := cmp.Compare(c.component[i], u.component[i]); c != 0 {
			return c
		}
	}
	return 0
}

// [v]A[.B[.C[.D][-PRERELEASE][+BUILD]]
type Version struct {
	v                 bool // v prefix
	core              Core
	prerelease, build string
}

func Parse(s string) (Version, error) {
	v, a, b, c, d, pre_, build_, err := vPrefixedValidExtendedVer(s)
	if err != nil {
		return Version{}, err
	}

	core, err := convertCore(a, b, c, d)
	if err != nil {
		return Version{}, err
	}

	return Version{
		v:          v,
		core:       core,
		prerelease: pre_,
		build:      build_,
	}, nil
}

func (v Version) Core() Core {
	return v.core
}

func (v Version) Compare(u Version) int {
	if c := v.core.Compare(u.core); c != 0 {
		return c
	}
	if v.core.length != u.core.length && v.prerelease == "" && u.prerelease == "" {
		return cmp.Compare(v.core.length, u.core.length)
	}
	// When major, minor, and patch are equal, a pre-release version has lower precedence than a normal version:
	//
	// Example: 1.0.0-alpha < 1.0.0.
	//
	// Precedence for two pre-release versions with the same major, minor, and patch version
	// MUST be determined by comparing each dot separated identifier
	// from left to right until a difference is found as follows:
	//
	// Identifiers consisting of only digits are compared numerically.
	//
	// Identifiers with letters or hyphens are compared lexically in ASCII sort order.
	//
	// Numeric identifiers always have lower precedence than non-numeric identifiers.
	//
	// A larger set of pre-release fields has a higher precedence than a smaller set, if all of the preceding identifiers are equal.
	//
	// Example: 1.0.0-alpha < 1.0.0-alpha.1 < 1.0.0-alpha.beta < 1.0.0-beta < 1.0.0-beta.2 < 1.0.0-beta.11 < 1.0.0-rc.1 < 1.0.0.
	if v.prerelease == u.prerelease {
		return 0
	}
	if v.prerelease == "" {
		return +1
	}
	if u.prerelease == "" {
		return -1
	}

	var left, leftRest, right, rightRest string
	left = v.prerelease
	right = u.prerelease
	for left != "" && right != "" {
		left, leftRest, _ = strings.Cut(left, ".")
		right, rightRest, _ = strings.Cut(right, ".")
		if left != right {
			il := isNum(left)
			ir := isNum(right)
			if il != ir {
				if il {
					return -1
				} else {
					return +1
				}
			}
			if il {
				if len(left) < len(right) {
					return -1
				}
				if len(left) > len(right) {
					return +1
				}
			}
			if left < right {
				return -1
			} else {
				return +1
			}
		}
		left = leftRest
		right = rightRest
	}
	if left == "" {
		return -1
	} else {
		return +1
	}
}

func isNum(s string) bool {
	i := 0
	for i < len(s) && '0' <= s[i] && s[i] <= '9' {
		i++
	}
	return i == len(s)
}

func takeString(s string, cond func(r rune) bool) (took, rest string, ok bool) {
	var leng int
	for {
		r, n := utf8.DecodeRuneInString(s[leng:])
		if n == 0 || !cond(r) {
			break
		}
		leng += n
	}
	return s[:leng], s[leng:], leng > 0
}

// <v prefixed valid extended ver> ::= "v" <valid extended ver>
//
//	| <valid extended ver>
func vPrefixedValidExtendedVer(s string) (v bool, a, b, c, d int64, pre_, build_ string, err error) {
	if len(s) > 0 && s[0] == 'v' {
		v = true
		s = s[1:]
	}
	a, b, c, d, pre_, build_, err = validExtendedVer(s)
	return
}

// <valid extended ver> ::= <major>
//
//	| <major> "." <minor>
//	| <full version core>
//	| <full version core> "-" <pre-release>
//	| <full version core> "+" <build>
//	| <full version core> "-" <pre-release> "+" <build>
func validExtendedVer(s string) (a, b, c, d int64, pre_, build_ string, err error) {
	a, b, c, d, s, err = version(s)

	var (
		ok bool
	)

	if len(s) == 0 {
		if a < 0 {
			err = fmt.Errorf("empty string")
		}
		return
	}

	if c < 0 {
		err = fmt.Errorf("missing patch")
		return
	}

	if s[0] != '-' && s[0] != '+' {
		err = fmt.Errorf("extra string after version core")
	}

	if s[0] == '-' {
		s = s[1:]
		pre_, s, ok = preRelease(s)
		if !ok {
			err = fmt.Errorf("invalid \"pre-prelease\"")
			return
		}
		if len(s) == 0 {
			return
		}
	}

	if s[0] == '+' {
		s = s[1:]
		build_, s, ok = build(s)
		if !ok {
			err = fmt.Errorf("invalid \"build\"")
			return
		}
	}

	if len(s) != 0 {
		err = fmt.Errorf("extra string %q after extended ver", s)
		return
	}

	return
}

func componentName(idx int) string {
	switch idx {
	case 0:
		return "major"
	case 1:
		return "minor"
	case 2:
		return "patch"
	default:
		return "extra"
	}
}

// <version> ::= <major>
//
//	| <major> "." <minor>
//	| <major> "." <minor> "." <patch>
//	| <major> "." <minor> "." <patch> "." <extra>
func version(s string) (a, b, c, d int64, rest string, err error) {
	a, b, c, d = -1, -1, -1, -1
	var (
		num    string
		parsed uint64
		ok     bool
	)

LOOP:
	for i := range 4 {
		if i > 0 {
			if len(s) == 0 {
				break
			}
			switch s[0] {
			case '.':
				s = s[1:]
			case '-', '+':
				break LOOP
			default:
				err = fmt.Errorf("missing '.', '-' or '+' after %q", componentName(i-1))
				return
			}
		}

		num, s, ok = numericIdentifier(s)
		if !ok {
			err = fmt.Errorf("missing %q", componentName(i))
			return
		}
		parsed, err = strconv.ParseUint(num, 10, 63)
		if err != nil {
			err = fmt.Errorf("parsing %q: %w", componentName(i), err)
			return
		}
		switch i {
		case 0:
			a = int64(parsed)
		case 1:
			b = int64(parsed)
		case 2:
			c = int64(parsed)
		case 3:
			d = int64(parsed)
		}
	}

	return a, b, c, d, s, nil
}

// modified
//
// <full version core> ::= <major> "." <minor> "." <patch>
//                        | <major> "." <minor> "." <patch> "." <extra>

// <major> ::= <numeric identifier>

// <minor> ::= <numeric identifier>

// <patch> ::= <numeric identifier>

// <pre-release> ::= <dot-separated pre-release identifiers>
func preRelease(s string) (ident, rest string, ok bool) {
	return dotSeparatedPreReleaseIdentifiers(s)
}

// <dot-separated pre-release identifiers> ::= <pre-release identifier>
//
//	| <pre-release identifier> "." <dot-separated pre-release identifiers>
func dotSeparatedPreReleaseIdentifiers(s string) (ident, rest string, ok bool) {
	rest = s
	var leng int
	for len(rest) > 0 {
		ident, rest, ok = preReleaseIdentifier(rest)
		leng += len(ident)
		if !ok {
			return "", s, false
		}
		if len(rest) > 0 {
			if rest[0] != '.' {
				break
			}
			leng += 1
			rest = rest[1:]
		}
	}
	return s[:leng], rest, true
}

// <build> ::= <dot-separated build identifiers>
func build(s string) (ident, rest string, ok bool) {
	return dotSeparatedBuildIdentifiers(s)
}

// <dot-separated build identifiers> ::= <build identifier>
//
//	| <build identifier> "." <dot-separated build identifiers>
func dotSeparatedBuildIdentifiers(s string) (ident, rest string, ok bool) {
	rest = s
	var leng int
	for len(rest) > 0 {
		ident, rest, ok = buildIdentidiers(rest)
		leng += len(ident)
		if !ok {
			return "", s, false
		}
		if len(rest) > 0 {
			if rest[0] != '.' {
				break
			}
			leng += 1
			rest = rest[1:]
		}
	}
	return s[:leng], rest, true
}

// <pre-release identifier> ::= <alphanumeric identifier>
//
//	| <numeric identifier>
func preReleaseIdentifier(s string) (ident, rest string, ok bool) {
	ident, rest, ok = alphanumericIdentifier(s)
	if ok {
		return
	}
	return numericIdentifier(s)
}

// <build identifier> ::= <alphanumeric identifier>
//
//	| <digits>
func buildIdentidiers(s string) (ident, rest string, ok bool) {
	ident, rest, ok = alphanumericIdentifier(s)
	if ok {
		return
	}
	return digits(s)
}

// <alphanumeric identifier> ::= <non-digit>
//
//	| <non-digit> <identifier characters>
//	| <identifier characters> <non-digit>
//	| <identifier characters> <non-digit> <identifier characters>
func alphanumericIdentifier(s string) (ident, rest string, ok bool) {
	if len(s) >= 2 {
		if nonDigit(rune(s[0])) {
			_idents, rest, ok := identifierCharacters(s[1:])
			if ok {
				return s[:len(_idents)+1], rest, true
			}
			return s[:1], s[1:], true
		}
		if _chars, rest, ok := identifierCharacters(s); ok {
			// As per spec, identifier characters contains non-digit.
			// So...basically last 2 lines can be coalesced to <identifier characters> I guess?
			return _chars, rest, true
		}
	}
	if len(s) >= 1 && nonDigit(rune(s[0])) {
		return s[:1], s[1:], true
	}
	return "", s, false
}

// <numeric identifier> ::= "0"
//
//	| <positive digit>
//	| <positive digit> <digits>
func numericIdentifier(s string) (ident, rest string, ok bool) {
	if len(s) >= 2 && positiveDigit(rune(s[0])) {
		_digits, _, ok := digits(s[1:])
		if ok {
			return s[:len(_digits)+1], s[len(_digits)+1:], true
		}
	}
	if len(s) >= 1 && (s[0] == '0' || positiveDigit(rune(s[0]))) {
		return s[:1], s[1:], true
	}
	return "", s, false
}

// <identifier characters> ::= <identifier character>
//
//	| <identifier character> <identifier characters>
func identifierCharacters(s string) (characters, rest string, ok bool) {
	return takeString(s, identifierCharacter)
}

// <identifier character> ::= <digit>
//
//	| <non-digit>
func identifierCharacter(r rune) bool {
	return digit(r) || nonDigit(r)
}

// <non-digit> ::= <letter>
//
//	| "-"
func nonDigit(r rune) bool {
	return r == '-' || letter(r)
}

// <digits> ::= <digit>
//
//	| <digit> <digits>
func digits(s string) (digits, rest string, ok bool) {
	return takeString(s, digit)
}

// <digit> ::= "0"
//
//	| <positive digit>
func digit(r rune) bool {
	return r == '0' || positiveDigit(r)
}

// <positive digit> ::= "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9"
func positiveDigit(r rune) bool {
	return '1' <= r && r <= '9'
}

// <letter> ::= "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J"
//
// | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T"
// | "U" | "V" | "W" | "X" | "Y" | "Z" | "a" | "b" | "c" | "d"
// | "e" | "f" | "g" | "h" | "i" | "j" | "k" | "l" | "m" | "n"
// | "o" | "p" | "q" | "r" | "s" | "t" | "u" | "v" | "w" | "x"
// | "y" | "z"
func letter(r rune) bool {
	return ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z')
}
