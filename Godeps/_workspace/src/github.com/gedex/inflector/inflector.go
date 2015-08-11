// Copyright 2013 Akeda Bagus <admin@gedex.web.id>. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package inflector pluralizes and singularizes English nouns.

There are only two exported functions: `Pluralize` and `Singularize`.

	s := "People"
	fmt.Println(inflector.Singularize(s)) // will print "Person"

	s2 := "octopus"
	fmt.Println(inflector.Pluralize(s2)) // will print "octopuses"

*/
package inflector

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// Rule represents name of the inflector rule, can be
// Plural or Singular
type Rule int

const (
	Plural = iota
	Singular
)

// InflectorRule represents inflector rule
type InflectorRule struct {
	Rules               []*ruleItem
	Irregular           []*irregularItem
	Uninflected         []string
	compiledIrregular   *regexp.Regexp
	compiledUninflected *regexp.Regexp
	compiledRules       []*compiledRule
}

type ruleItem struct {
	pattern     string
	replacement string
}

type irregularItem struct {
	word        string
	replacement string
}

// compiledRule represents compiled version of Inflector.Rules.
type compiledRule struct {
	replacement string
	*regexp.Regexp
}

var rules = make(map[Rule]*InflectorRule)

// Words that should not be inflected
var uninflected = []string{
	`Amoyese`, `bison`, `Borghese`, `bream`, `breeches`, `britches`, `buffalo`,
	`cantus`, `carp`, `chassis`, `clippers`, `cod`, `coitus`, `Congoese`,
	`contretemps`, `corps`, `debris`, `diabetes`, `djinn`, `eland`, `elk`,
	`equipment`, `Faroese`, `flounder`, `Foochowese`, `gallows`, `Genevese`,
	`Genoese`, `Gilbertese`, `graffiti`, `headquarters`, `herpes`, `hijinks`,
	`Hottentotese`, `information`, `innings`, `jackanapes`, `Kiplingese`,
	`Kongoese`, `Lucchese`, `mackerel`, `Maltese`, `.*?media`, `mews`, `moose`,
	`mumps`, `Nankingese`, `news`, `nexus`, `Niasese`, `Pekingese`,
	`Piedmontese`, `pincers`, `Pistoiese`, `pliers`, `Portuguese`, `proceedings`,
	`rabies`, `rice`, `rhinoceros`, `salmon`, `Sarawakese`, `scissors`,
	`sea[- ]bass`, `series`, `Shavese`, `shears`, `siemens`, `species`, `swine`,
	`testes`, `trousers`, `trout`, `tuna`, `Vermontese`, `Wenchowese`, `whiting`,
	`wildebeest`, `Yengeese`,
}

// Plural words that should not be inflected
var uninflectedPlurals = []string{
	`.*[nrlm]ese`, `.*deer`, `.*fish`, `.*measles`, `.*ois`, `.*pox`, `.*sheep`,
	`people`,
}

// Singular words that should not be inflected
var uninflectedSingulars = []string{
	`.*[nrlm]ese`, `.*deer`, `.*fish`, `.*measles`, `.*ois`, `.*pox`, `.*sheep`,
	`.*ss`,
}

type cache map[string]string

// Inflected words that already cached for immediate retrieval from a given Rule
var caches = make(map[Rule]cache)

// map of irregular words where its key is a word and its value is the replacement
var irregularMaps = make(map[Rule]cache)

func init() {

	rules[Plural] = &InflectorRule{
		Rules: []*ruleItem{
			&ruleItem{`(?i)(s)tatus$`, `${1}${2}tatuses`},
			&ruleItem{`(?i)(quiz)$`, `${1}zes`},
			&ruleItem{`(?i)^(ox)$`, `${1}${2}en`},
			&ruleItem{`(?i)([m|l])ouse$`, `${1}ice`},
			&ruleItem{`(?i)(matr|vert|ind)(ix|ex)$`, `${1}ices`},
			&ruleItem{`(?i)(x|ch|ss|sh)$`, `${1}es`},
			&ruleItem{`(?i)([^aeiouy]|qu)y$`, `${1}ies`},
			&ruleItem{`(?i)(hive)$`, `$1s`},
			&ruleItem{`(?i)(?:([^f])fe|([lre])f)$`, `${1}${2}ves`},
			&ruleItem{`(?i)sis$`, `ses`},
			&ruleItem{`(?i)([ti])um$`, `${1}a`},
			&ruleItem{`(?i)(p)erson$`, `${1}eople`},
			&ruleItem{`(?i)(m)an$`, `${1}en`},
			&ruleItem{`(?i)(c)hild$`, `${1}hildren`},
			&ruleItem{`(?i)(buffal|tomat)o$`, `${1}${2}oes`},
			&ruleItem{`(?i)(alumn|bacill|cact|foc|fung|nucle|radi|stimul|syllab|termin|vir)us$`, `${1}i`},
			&ruleItem{`(?i)us$`, `uses`},
			&ruleItem{`(?i)(alias)$`, `${1}es`},
			&ruleItem{`(?i)(ax|cris|test)is$`, `${1}es`},
			&ruleItem{`s$`, `s`},
			&ruleItem{`^$`, ``},
			&ruleItem{`$`, `s`},
		},
		Irregular: []*irregularItem{
			&irregularItem{`atlas`, `atlases`},
			&irregularItem{`beef`, `beefs`},
			&irregularItem{`brother`, `brothers`},
			&irregularItem{`cafe`, `cafes`},
			&irregularItem{`child`, `children`},
			&irregularItem{`cookie`, `cookies`},
			&irregularItem{`corpus`, `corpuses`},
			&irregularItem{`cow`, `cows`},
			&irregularItem{`ganglion`, `ganglions`},
			&irregularItem{`genie`, `genies`},
			&irregularItem{`genus`, `genera`},
			&irregularItem{`graffito`, `graffiti`},
			&irregularItem{`hoof`, `hoofs`},
			&irregularItem{`loaf`, `loaves`},
			&irregularItem{`man`, `men`},
			&irregularItem{`money`, `monies`},
			&irregularItem{`mongoose`, `mongooses`},
			&irregularItem{`move`, `moves`},
			&irregularItem{`mythos`, `mythoi`},
			&irregularItem{`niche`, `niches`},
			&irregularItem{`numen`, `numina`},
			&irregularItem{`occiput`, `occiputs`},
			&irregularItem{`octopus`, `octopuses`},
			&irregularItem{`opus`, `opuses`},
			&irregularItem{`ox`, `oxen`},
			&irregularItem{`penis`, `penises`},
			&irregularItem{`person`, `people`},
			&irregularItem{`sex`, `sexes`},
			&irregularItem{`soliloquy`, `soliloquies`},
			&irregularItem{`testis`, `testes`},
			&irregularItem{`trilby`, `trilbys`},
			&irregularItem{`turf`, `turfs`},
			&irregularItem{`potato`, `potatoes`},
			&irregularItem{`hero`, `heroes`},
			&irregularItem{`tooth`, `teeth`},
			&irregularItem{`goose`, `geese`},
			&irregularItem{`foot`, `feet`},
		},
	}
	prepare(Plural)

	rules[Singular] = &InflectorRule{
		Rules: []*ruleItem{
			&ruleItem{`(?i)(s)tatuses$`, `${1}${2}tatus`},
			&ruleItem{`(?i)^(.*)(menu)s$`, `${1}${2}`},
			&ruleItem{`(?i)(quiz)zes$`, `$1`},
			&ruleItem{`(?i)(matr)ices$`, `${1}ix`},
			&ruleItem{`(?i)(vert|ind)ices$`, `${1}ex`},
			&ruleItem{`(?i)^(ox)en`, `$1`},
			&ruleItem{`(?i)(alias)(es)*$`, `$1`},
			&ruleItem{`(?i)(alumn|bacill|cact|foc|fung|nucle|radi|stimul|syllab|termin|viri?)i$`, `${1}us`},
			&ruleItem{`(?i)([ftw]ax)es`, `$1`},
			&ruleItem{`(?i)(cris|ax|test)es$`, `${1}is`},
			&ruleItem{`(?i)(shoe|slave)s$`, `$1`},
			&ruleItem{`(?i)(o)es$`, `$1`},
			&ruleItem{`ouses$`, `ouse`},
			&ruleItem{`([^a])uses$`, `${1}us`},
			&ruleItem{`(?i)([m|l])ice$`, `${1}ouse`},
			&ruleItem{`(?i)(x|ch|ss|sh)es$`, `$1`},
			&ruleItem{`(?i)(m)ovies$`, `${1}${2}ovie`},
			&ruleItem{`(?i)(s)eries$`, `${1}${2}eries`},
			&ruleItem{`(?i)([^aeiouy]|qu)ies$`, `${1}y`},
			&ruleItem{`(?i)(tive)s$`, `$1`},
			&ruleItem{`(?i)([lre])ves$`, `${1}f`},
			&ruleItem{`(?i)([^fo])ves$`, `${1}fe`},
			&ruleItem{`(?i)(hive)s$`, `$1`},
			&ruleItem{`(?i)(drive)s$`, `$1`},
			&ruleItem{`(?i)(^analy)ses$`, `${1}sis`},
			&ruleItem{`(?i)(analy|diagno|^ba|(p)arenthe|(p)rogno|(s)ynop|(t)he)ses$`, `${1}${2}sis`},
			&ruleItem{`(?i)([ti])a$`, `${1}um`},
			&ruleItem{`(?i)(p)eople$`, `${1}${2}erson`},
			&ruleItem{`(?i)(m)en$`, `${1}an`},
			&ruleItem{`(?i)(c)hildren$`, `${1}${2}hild`},
			&ruleItem{`(?i)(n)ews$`, `${1}${2}ews`},
			&ruleItem{`eaus$`, `eau`},
			&ruleItem{`^(.*us)$`, `$1`},
			&ruleItem{`(?i)s$`, ``},
		},
		Irregular: []*irregularItem{
			&irregularItem{`foes`, `foe`},
			&irregularItem{`waves`, `wave`},
			&irregularItem{`curves`, `curve`},
			&irregularItem{`atlases`, `atlas`},
			&irregularItem{`beefs`, `beef`},
			&irregularItem{`brothers`, `brother`},
			&irregularItem{`cafes`, `cafe`},
			&irregularItem{`children`, `child`},
			&irregularItem{`cookies`, `cookie`},
			&irregularItem{`corpuses`, `corpus`},
			&irregularItem{`cows`, `cow`},
			&irregularItem{`ganglions`, `ganglion`},
			&irregularItem{`genies`, `genie`},
			&irregularItem{`genera`, `genus`},
			&irregularItem{`graffiti`, `graffito`},
			&irregularItem{`hoofs`, `hoof`},
			&irregularItem{`loaves`, `loaf`},
			&irregularItem{`men`, `man`},
			&irregularItem{`monies`, `money`},
			&irregularItem{`mongooses`, `mongoose`},
			&irregularItem{`moves`, `move`},
			&irregularItem{`mythoi`, `mythos`},
			&irregularItem{`niches`, `niche`},
			&irregularItem{`numina`, `numen`},
			&irregularItem{`occiputs`, `occiput`},
			&irregularItem{`octopuses`, `octopus`},
			&irregularItem{`opuses`, `opus`},
			&irregularItem{`oxen`, `ox`},
			&irregularItem{`penises`, `penis`},
			&irregularItem{`people`, `person`},
			&irregularItem{`sexes`, `sex`},
			&irregularItem{`soliloquies`, `soliloquy`},
			&irregularItem{`testes`, `testis`},
			&irregularItem{`trilbys`, `trilby`},
			&irregularItem{`turfs`, `turf`},
			&irregularItem{`potatoes`, `potato`},
			&irregularItem{`heroes`, `hero`},
			&irregularItem{`teeth`, `tooth`},
			&irregularItem{`geese`, `goose`},
			&irregularItem{`feet`, `foot`},
		},
	}
	prepare(Singular)
}

// prepare rule, e.g., compile the pattern.
func prepare(r Rule) error {
	var reString string

	switch r {
	case Plural:
		// Merge global uninflected with singularsUninflected
		rules[r].Uninflected = merge(uninflected, uninflectedPlurals)
	case Singular:
		// Merge global uninflected with singularsUninflected
		rules[r].Uninflected = merge(uninflected, uninflectedSingulars)
	}

	// Set InflectorRule.compiledUninflected by joining InflectorRule.Uninflected into
	// a single string then compile it.
	reString = fmt.Sprintf(`(?i)(^(?:%s))$`, strings.Join(rules[r].Uninflected, `|`))
	rules[r].compiledUninflected = regexp.MustCompile(reString)

	// Prepare irregularMaps
	irregularMaps[r] = make(cache, len(rules[r].Irregular))

	// Set InflectorRule.compiledIrregular by joining the irregularItem.word of Inflector.Irregular
	// into a single string then compile it.
	vIrregulars := make([]string, len(rules[r].Irregular))
	for i, item := range rules[r].Irregular {
		vIrregulars[i] = item.word
		irregularMaps[r][item.word] = item.replacement
	}
	reString = fmt.Sprintf(`(?i)(.*)\b((?:%s))$`, strings.Join(vIrregulars, `|`))
	rules[r].compiledIrregular = regexp.MustCompile(reString)

	// Compile all patterns in InflectorRule.Rules
	rules[r].compiledRules = make([]*compiledRule, len(rules[r].Rules))
	for i, item := range rules[r].Rules {
		rules[r].compiledRules[i] = &compiledRule{item.replacement, regexp.MustCompile(item.pattern)}
	}

	// Prepare caches
	caches[r] = make(cache)

	return nil
}

// merge slice a and slice b
func merge(a []string, b []string) []string {
	result := make([]string, len(a)+len(b))
	copy(result, a)
	copy(result[len(a):], b)

	return result
}

// Pluralize returns string s in plural form.
func Pluralize(s string) string {
	return getInflected(Plural, s)
}

// Singularize returns string s in singular form.
func Singularize(s string) string {
	return getInflected(Singular, s)
}

func getInflected(r Rule, s string) string {
	if v, ok := caches[r][s]; ok {
		return v
	}

	// Check for irregular words
	if res := rules[r].compiledIrregular.FindStringSubmatch(s); len(res) >= 3 {
		var buf bytes.Buffer

		buf.WriteString(res[1])
		buf.WriteString(s[0:1])
		buf.WriteString(irregularMaps[r][strings.ToLower(res[2])][1:])

		// Cache it then returns
		caches[r][s] = buf.String()
		return caches[r][s]
	}

	// Check for uninflected words
	if rules[r].compiledUninflected.MatchString(s) {
		caches[r][s] = s
		return caches[r][s]
	}

	// Check each rule
	for _, re := range rules[r].compiledRules {
		if re.MatchString(s) {
			caches[r][s] = re.ReplaceAllString(s, re.replacement)
			return caches[r][s]
		}
	}

	// Returns unaltered
	caches[r][s] = s
	return caches[r][s]
}
