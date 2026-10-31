package goja

import (
	"sort"
	"strings"
)

// intlVariantAliases holds the CLDR deprecated variant aliases.
var intlLanguageAliasExtra = map[string]string{
	"hy-arevela": "hy",
}

var intlVariantAliases = map[string]string{
	"heploc":   "alalc97",
	"polytoni": "polyton",
}

// intlTag is a parsed, canonicalized Unicode language tag (UTS 35).
type intlTag struct {
	language string
	script   string
	region   string
	variants []string
	ext      map[byte][]string // extensions other than "u"
	uAttr    []string
	uKw      map[string]string
	uKwOrder []string
	private  []string
}

func intlIsAlpha(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z') {
			return false
		}
	}
	return true
}

func intlIsDigit(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func intlIsAlnum(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func intlIsLanguageSubtag(s string) bool {
	if !intlIsAlpha(s) {
		return false
	}
	n := len(s)
	return n >= 2 && n <= 3 || n >= 5 && n <= 8
}

func intlIsScriptSubtag(s string) bool { return len(s) == 4 && intlIsAlpha(s) }

func intlIsRegionSubtag(s string) bool {
	if len(s) == 2 && intlIsAlpha(s) {
		return true
	}
	return len(s) == 3 && intlIsDigit(s)
}

func intlIsVariantSubtag(s string) bool {
	n := len(s)
	if n >= 5 && n <= 8 && intlIsAlnum(s) {
		return true
	}
	return n == 4 && s[0] >= '0' && s[0] <= '9' && intlIsAlnum(s[1:])
}

// intlIsUKey reports whether s is a valid Unicode extension keyword key
// (alphanum followed by alpha).
func intlIsUKey(s string) bool {
	if len(s) != 2 {
		return false
	}
	c := s[0]
	if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
		return false
	}
	return s[1] >= 'a' && s[1] <= 'z' || s[1] >= 'A' && s[1] <= 'Z'
}

// intlParseTag parses a structurally valid Unicode language tag.
func intlParseTag(s string) (*intlTag, bool) {
	if s == "" {
		return nil, false
	}
	subs := strings.Split(s, "-")
	for _, sub := range subs {
		if sub == "" || len(sub) > 8 || !intlIsAlnum(sub) {
			return nil, false
		}
	}
	t := &intlTag{uKw: map[string]string{}}
	i := 0
	if !intlIsLanguageSubtag(subs[0]) {
		return nil, false
	}
	t.language = strings.ToLower(subs[0])
	i++
	if i < len(subs) && intlIsScriptSubtag(subs[i]) {
		t.script = intlTitleCase(subs[i])
		i++
	}
	if i < len(subs) && intlIsRegionSubtag(subs[i]) {
		t.region = strings.ToUpper(subs[i])
		i++
	}
	seenVariant := map[string]bool{}
	for i < len(subs) && intlIsVariantSubtag(subs[i]) {
		v := strings.ToLower(subs[i])
		if seenVariant[v] {
			return nil, false
		}
		seenVariant[v] = true
		t.variants = append(t.variants, v)
		i++
	}
	seenSingleton := map[byte]bool{}
	for i < len(subs) {
		if len(subs[i]) != 1 {
			return nil, false
		}
		singleton := subs[i][0]
		if singleton >= 'A' && singleton <= 'Z' {
			singleton += 'a' - 'A'
		}
		if !(singleton >= 'a' && singleton <= 'z' || singleton >= '0' && singleton <= '9') {
			return nil, false
		}
		if seenSingleton[singleton] {
			return nil, false
		}
		seenSingleton[singleton] = true
		i++
		if singleton == 'x' {
			if i == len(subs) {
				return nil, false
			}
			for _, b := range subs[i:] {
				t.private = append(t.private, strings.ToLower(b))
			}
			i = len(subs)
			continue
		}
		start := i
		for i < len(subs) && len(subs[i]) > 1 {
			if len(subs[i]) > 8 {
				return nil, false
			}
			i++
		}
		body := subs[start:i]
		if len(body) == 0 {
			return nil, false
		}
		switch singleton {
		case 'u':
			if !intlParseUnicodeExtension(t, body) {
				return nil, false
			}
		case 't':
			if !intlValidateTExtension(body) {
				return nil, false
			}
			if t.ext == nil {
				t.ext = map[byte][]string{}
			}
			raw := make([]string, len(body))
			for j, b := range body {
				raw[j] = strings.ToLower(b)
			}
			t.ext['t'] = raw
		default:
			if t.ext == nil {
				t.ext = map[byte][]string{}
			}
			e := make([]string, len(body))
			for j, b := range body {
				e[j] = strings.ToLower(b)
			}
			t.ext[singleton] = e
		}
	}
	return t, true
}

func intlTitleCase(s string) string {
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func intlParseUnicodeExtension(t *intlTag, body []string) bool {
	i := 0
	seenAttr := map[string]bool{}
	for i < len(body) && len(body[i]) >= 3 {
		a := strings.ToLower(body[i])
		if seenAttr[a] {
			return false
		}
		seenAttr[a] = true
		t.uAttr = append(t.uAttr, a)
		i++
	}
	for i < len(body) {
		if !intlIsUKey(body[i]) {
			return false
		}
		key := strings.ToLower(body[i])
		i++
		var vals []string
		for i < len(body) && len(body[i]) >= 3 {
			vals = append(vals, strings.ToLower(body[i]))
			i++
		}
		if _, dup := t.uKw[key]; dup {
			continue
		}
		t.uKw[key] = strings.Join(vals, "-")
		t.uKwOrder = append(t.uKwOrder, key)
	}
	return true
}

// intlIsTKey reports whether s is a valid transformed extension field key
// (alpha followed by digit).
func intlIsTKey(s string) bool {
	if len(s) != 2 {
		return false
	}
	c := s[0]
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// intlValidateTExtension validates a transformed extension body.
func intlValidateTExtension(body []string) bool {
	i := 0
	if i < len(body) && len(body[i]) >= 2 && intlIsAlpha(body[i]) {
		if !intlIsLanguageSubtag(body[i]) || strings.EqualFold(body[i], "root") {
			return false
		}
		i++
		if i < len(body) && intlIsScriptSubtag(body[i]) {
			i++
		}
		if i < len(body) && intlIsRegionSubtag(body[i]) {
			i++
		}
		seen := map[string]bool{}
		for i < len(body) && intlIsVariantSubtag(body[i]) {
			v := strings.ToLower(body[i])
			if seen[v] {
				return false
			}
			seen[v] = true
			i++
		}
	}
	seenKey := map[string]bool{}
	for i < len(body) {
		if !intlIsTKey(body[i]) {
			return false
		}
		key := strings.ToLower(body[i])
		if seenKey[key] {
			return false
		}
		seenKey[key] = true
		i++
		n := 0
		for i < len(body) && len(body[i]) >= 3 {
			n++
			i++
		}
		if n == 0 {
			return false
		}
	}
	return true
}

func (t *intlTag) clone() *intlTag {
	r := *t
	r.variants = append([]string(nil), t.variants...)
	r.uAttr = append([]string(nil), t.uAttr...)
	if t.ext != nil {
		r.ext = make(map[byte][]string, len(t.ext))
		for k, v := range t.ext {
			r.ext[k] = append([]string(nil), v...)
		}
	}
	r.uKw = make(map[string]string, len(t.uKw))
	for k, v := range t.uKw {
		r.uKw[k] = v
	}
	r.uKwOrder = append([]string(nil), t.uKwOrder...)
	r.private = append([]string(nil), t.private...)
	return &r
}

func (t *intlTag) hasExtensions() bool {
	return len(t.ext) > 0 || len(t.uKw) > 0 || len(t.uAttr) > 0 || len(t.private) > 0
}

// baseName returns the language tag without extensions.
func (t *intlTag) baseName() string {
	parts := make([]string, 0, 3+len(t.variants))
	parts = append(parts, t.language)
	if t.script != "" {
		parts = append(parts, t.script)
	}
	if t.region != "" {
		parts = append(parts, t.region)
	}
	parts = append(parts, t.variants...)
	return strings.Join(parts, "-")
}

func (t *intlTag) String() string {
	var sb strings.Builder
	sb.WriteString(t.baseName())
	singletons := make([]byte, 0, len(t.ext)+1)
	for s := range t.ext {
		singletons = append(singletons, s)
	}
	if len(t.uAttr) > 0 || len(t.uKw) > 0 {
		singletons = append(singletons, 'u')
	}
	sort.Slice(singletons, func(i, j int) bool { return singletons[i] < singletons[j] })
	for _, s := range singletons {
		sb.WriteString("-")
		sb.WriteByte(s)
		if s == 'u' {
			attrs := append([]string(nil), t.uAttr...)
			sort.Strings(attrs)
			for _, a := range attrs {
				sb.WriteString("-")
				sb.WriteString(a)
			}
			keys := make([]string, 0, len(t.uKw))
			for k := range t.uKw {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				sb.WriteString("-")
				sb.WriteString(k)
				if v := t.uKw[k]; v != "" {
					sb.WriteString("-")
					sb.WriteString(v)
				}
			}
			continue
		}
		for _, sub := range t.ext[s] {
			sb.WriteString("-")
			sb.WriteString(sub)
		}
	}
	if len(t.private) > 0 {
		sb.WriteString("-x")
		for _, p := range t.private {
			sb.WriteString("-")
			sb.WriteString(p)
		}
	}
	return sb.String()
}

// intlCanonicalizeBase applies language, script, region and variant aliases.
func intlCanonicalizeBase(t *intlTag) *intlTag {
	r := t.clone()
	r.applyLanguageAlias()
	if repl, ok := intlScriptAlias[r.script]; ok && repl != "" {
		r.script = repl
	}
	r.applyTerritoryAlias()
	r.applyVariantAlias()
	sort.Strings(r.variants)
	return r
}

func (r *intlTag) applyLanguageAlias() {
	lang := r.language
	if lang == "" {
		lang = "und"
	}
	type cand struct {
		key             string
		script, region  bool
		variantsMatched []string
	}
	var cands []cand
	allVariants := append([]string(nil), r.variants...)
	vars := strings.Join(allVariants, "-")
	build := func(language string) {
		prefix := language
		if r.script != "" && r.region != "" {
			k := prefix + "-" + r.script + "-" + r.region
			if vars != "" {
				cands = append(cands, cand{k + "-" + vars, true, true, allVariants})
			}
			cands = append(cands, cand{k, true, true, nil})
		}
		if r.region != "" {
			k := prefix + "-" + r.region
			if vars != "" {
				cands = append(cands, cand{k + "-" + vars, false, true, allVariants})
			}
			cands = append(cands, cand{k, false, true, nil})
		}
		if r.script != "" {
			k := prefix + "-" + r.script
			if vars != "" {
				cands = append(cands, cand{k + "-" + vars, true, false, allVariants})
			}
			cands = append(cands, cand{k, true, false, nil})
		}
		for _, v := range allVariants {
			cands = append(cands, cand{prefix + "-" + v, false, false, []string{v}})
		}
		if vars != "" {
			cands = append(cands, cand{prefix + "-" + vars, false, false, allVariants})
		}
		cands = append(cands, cand{prefix, false, false, nil})
	}
	build(lang)
	if lang != "und" {
		build("und")
	}
	for _, c := range cands {
		repl, ok := intlLanguageAlias[c.key]
		if !ok {
			repl, ok = intlLanguageAliasExtra[c.key]
		}
		if !ok || repl == "" {
			continue
		}
		nt, ok := intlParseTag(repl)
		if !ok {
			continue
		}
		if c.script || nt.script != "" && r.script == "" {
			r.script = nt.script
		}
		if c.region || nt.region != "" && r.region == "" {
			r.region = nt.region
		}
		if len(c.variantsMatched) > 0 {
			matched := map[string]bool{}
			for _, v := range c.variantsMatched {
				matched[v] = true
			}
			var remaining []string
			for _, v := range r.variants {
				if !matched[v] {
					remaining = append(remaining, v)
				}
			}
			r.variants = append(remaining, nt.variants...)
		}
		if nt.language != "" && nt.language != "und" {
			r.language = nt.language
		}
		return
	}
}

func (r *intlTag) applyTerritoryAlias() {
	repl, ok := intlTerritoryAlias[r.region]
	if !ok || repl == "" {
		return
	}
	choices := strings.Split(repl, " ")
	if len(choices) == 1 {
		r.region = choices[0]
		return
	}
	if r.region != "" {
		lt := intlMaximize(&intlTag{language: r.language, script: r.script, uKw: map[string]string{}})
		for _, c := range choices {
			if c == lt.region {
				r.region = c
				return
			}
		}
	}
	r.region = choices[0]
}

func (r *intlTag) applyVariantAlias() {
	var out []string
	seen := map[string]bool{}
	for _, v := range r.variants {
		if a, ok := intlVariantAliases[v]; ok {
			v = a
		}
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	r.variants = out
}

// intlCanonicalizeTag applies CLDR aliases and case canonicalization.
func intlCanonicalizeTag(t *intlTag) *intlTag {
	r := intlCanonicalizeBase(t)
	r.canonicalizeUnicodeExtension()
	if raw, ok := r.ext['t']; ok {
		r.ext['t'] = intlCanonicalizeTExtension(raw)
	}
	return r
}

func (r *intlTag) canonicalizeUnicodeExtension() {
	if len(r.uKw) == 0 {
		return
	}
	out := make(map[string]string, len(r.uKw))
	for k, v := range r.uKw {
		nv := v
		if m, ok := intlUKeywordAlias[k]; ok {
			if a, ok := m[nv]; ok {
				nv = a
			}
		}
		switch k {
		case "rg", "sd":
			if a, ok := intlSubdivisionAlias[nv]; ok && a != "" {
				nv = strings.Split(a, " ")[0]
			}
		case "tz":
			if a, ok := intlTimezoneUCanonical[nv]; ok && a != "" {
				nv = a
			} else if a, ok := intlTimezoneUCanonicalExtra[nv]; ok && a != "" {
				nv = a
			}
		}
		if nv == "true" {
			nv = ""
		}
		out[k] = nv
	}
	r.uKw = out
}

func intlCanonicalizeTExtension(body []string) []string {
	out := make([]string, 0, len(body))
	i := 0
	if i < len(body) && len(body[i]) >= 2 && intlIsAlpha(body[i]) {
		t := &intlTag{language: strings.ToLower(body[i]), uKw: map[string]string{}}
		i++
		if i < len(body) && intlIsScriptSubtag(body[i]) {
			t.script = intlTitleCase(body[i])
			i++
		}
		if i < len(body) && intlIsRegionSubtag(body[i]) {
			t.region = strings.ToUpper(body[i])
			i++
		}
		for i < len(body) && intlIsVariantSubtag(body[i]) {
			t.variants = append(t.variants, strings.ToLower(body[i]))
			i++
		}
		ct := intlCanonicalizeBase(t)
		out = append(out, strings.ToLower(ct.language))
		if ct.script != "" {
			out = append(out, strings.ToLower(ct.script))
		}
		if ct.region != "" {
			out = append(out, strings.ToLower(ct.region))
		}
		for _, v := range ct.variants {
			out = append(out, strings.ToLower(v))
		}
	}
	type field struct {
		key  string
		vals []string
	}
	var fields []field
	for i < len(body) {
		key := strings.ToLower(body[i])
		i++
		var vals []string
		for i < len(body) && len(body[i]) >= 3 {
			v := strings.ToLower(body[i])
			if m, ok := intlTValueAlias[key]; ok {
				if a, ok := m[v]; ok {
					v = a
				}
			}
			vals = append(vals, v)
			i++
		}
		fields = append(fields, field{key, vals})
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].key < fields[j].key })
	for _, f := range fields {
		out = append(out, f.key)
		out = append(out, f.vals...)
	}
	return out
}

// intlCanonicalizeTagString parses and canonicalizes s, returning false if
// the tag is not structurally valid.
func intlCanonicalizeTagString(s string) (string, bool) {
	t, ok := intlParseTag(s)
	if !ok {
		return "", false
	}
	return intlCanonicalizeTag(t).String(), true
}

func intlMaximize(t *intlTag) *intlTag {
	r := t.clone()
	if r.language == "" {
		return r
	}
	keys := []string{}
	if r.script != "" {
		if r.region != "" {
			keys = append(keys, r.language+"-"+r.script+"-"+r.region)
		}
		keys = append(keys, r.language+"-"+r.script)
	}
	if r.region != "" {
		keys = append(keys, r.language+"-"+r.region)
	}
	keys = append(keys, r.language)
	for _, key := range keys {
		if v, ok := intlLikelySubtags[key]; ok {
			lt, ok := intlParseTag(v)
			if !ok {
				continue
			}
			if r.script == "" {
				r.script = lt.script
			}
			if r.region == "" {
				r.region = lt.region
			}
			if r.language == "" || r.language == "und" {
				r.language = lt.language
			}
			return r
		}
	}
	return r
}

func intlMinimize(t *intlTag) *intlTag {
	if t.language == "" {
		return t.clone()
	}
	max := intlMaximize(t)
	res := t.clone()
	if res.language == "und" {
		res.language = max.language
		if res.language == "" {
			res.language = "en"
		}
	}
	vars := strings.Join(res.variants, "-")
	try := func(lang, script, region string) *intlTag {
		var parts []string
		if lang != "" {
			parts = append(parts, lang)
		}
		if script != "" {
			parts = append(parts, script)
		}
		if region != "" {
			parts = append(parts, region)
		}
		if vars != "" {
			parts = append(parts, vars)
		}
		if len(parts) == 0 {
			return nil
		}
		cand, ok := intlParseTag(strings.Join(parts, "-"))
		if !ok {
			return nil
		}
		if intlMaximize(cand).baseName() == max.baseName() {
			return cand
		}
		return nil
	}
	var found *intlTag
	if c := try(res.language, "", ""); c != nil {
		found = c
	}
	if found == nil && max.region != "" {
		found = try(res.language, "", max.region)
	}
	if found == nil && res.script != "" {
		found = try(res.language, res.script, "")
	}
	if found == nil {
		return res
	}
	res.script = found.script
	res.region = found.region
	return res
}
