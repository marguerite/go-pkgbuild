package main

// 1. find subpackages
// 2. parse the main package
// 3. parse subpackages and fill the un-overrided tags with the ones from main package
// 4. link the subpackages with the main package

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
)

type Optdepends struct {
	Pkgname string
	Desc    string
}

type Pkgbuild struct {
	Pkgname      interface{}
	Pkgbase      interface{}
	Pkgver       interface{}
	Pkgrel       interface{}
	Pkgdesc      interface{}
	Url          interface{}
	Arch         interface{}
	License      interface{}
	Makedepends  interface{}
	Optdepends   interface{}
	Source       interface{}
	Sha256sums   interface{}
	Sha512sums   interface{}
	Func_check   string
	Func_build   string
	Func_prepare string
	Func_package string
	Func_pkgver  string
	Preamble     string
	Macros       []string
}

func errChk(e error) {
	if e != nil {
		panic(e)
	}
}

func parse(s string) Pkgbuild {
	_, main := splitSubPkgStr(s)
	fmt.Println(main)
	p := Pkgbuild{}
	p.Pkgname = match("pkgname", main)
	p.Pkgbase = match("pkgbase", main)
	p.Pkgver = match("pkgver", main)
	p.Pkgrel = match("pkgrel", main)
	p.Pkgdesc = match("pkgdesc", main)
	p.Url = match("url", main)
	p.Arch = match("arch", main)
	p.License = match("license", main)
	p.Makedepends = match("makedepends", main)
	p.Optdepends = match("optdepends", main)
	p.Source = match("source", main)
	p.Sha256sums = match("sha256sums", main)
	p.Sha512sums = match("sha512sums", main)
	p.Func_prepare = matchFunc("prepare", main)
	p.Func_build = matchFunc("build", main)
	p.Func_check = matchFunc("check", main)
	p.Func_package = matchFunc("package", main)
	p.Func_pkgver = matchFunc("pkgver", main)
	p.Preamble = matchPreamble(main)
	p.Macros = matchMacros(main)
	return p
}

func splitSubPkgStr(s string) (sub []string, main string) {
	r := regexp.MustCompile(`(?m)^package_.*?\(\) {\n[^}]+}`)
	// return reasonable value if no subpackage
	if !r.MatchString(s) {
		return sub, s
	}
	m := r.FindAllStringSubmatch(s, -1)
	main = s
	for _, i := range m {
		sub = append(sub, i[0])
	}
	for _, j := range sub {
		main = strings.Replace(main, j, "", -1)
	}
	return sub, main
}

func filter(t string, s string, fn bool) string {
	// golang's regexp has no negative lookahead, so we have to parse string line by line
	lines := strings.Split(s, "\n")
	// subtext holder, begin pos, end pos
	st, bp, ep := "", 0, 0
	// begin pos regex, end pos regex
	var br *regexp.Regexp
	var er *regexp.Regexp

	if fn {
		br = regexp.MustCompile(`^` + regexp.QuoteMeta(t) + `\(\)\s*{`)
		er = regexp.MustCompile(`^}$`)
	} else {
		br = regexp.MustCompile(`^\s*` + regexp.QuoteMeta(t) + `=`)
		er = regexp.MustCompile(`^\s*($|\w+(=|\())`)
	}

	// loop once to find the begin pos
	for i, j := range lines {
		if br.MatchString(j) {
			if fn {
				// pkgbuild function don't need the first line eg "build() {" as subtext
				bp = i + 1
			} else {
				bp = i
			}
			break
		}
	}
	// early return if no begin pos (no such tag)
	if bp == 0 {
		return st
	}
	// loop again to find the end pos
	for m, n := range lines {
		if m <= bp {
			continue
		}
		if er.MatchString(n) {
			ep = m - 1
			break
		}
	}
	// loop the third time to find the real text
	for k := bp; k <= ep; k++ {
		if k == ep {
			st += lines[k]
			break
		}
		st += lines[k] + "\n"
	}
	return st
}

func match(t, s string) interface{} {
	// return reasonable value if the provided text to match is empty
	if len(s) == 0 {
		return ""
	}
	text := filter(t, s, false)
	// return reasonable value if no such tag
	if len(text) == 0 {
		return ""
	}
	// we have to use different regex for multiline text
	// because multiline text certainly has a ")" as its end.
	// but singleline text may be package description, which
	// may have ")" from upstream description.
	text_len := len(strings.Split(text, "\n"))
	var r *regexp.Regexp
	var m string
	if text_len == 1 {
		// FIXME: I can't regex out the ending ")" for the first time, so use two regexes.
		r = regexp.MustCompile(`^(\s+)?` + regexp.QuoteMeta(t) + `=(\()?(.*)`)
		end_bracket_r := regexp.MustCompile(`(.*)\)$`)
		res := r.FindStringSubmatch(text)[3]
		if end_bracket_r.MatchString(res) {
			m = end_bracket_r.ReplaceAllString(res, "$1")
		} else {
			m = res
		}
	} else {
		r = regexp.MustCompile(`(?m)^(\s+)?` + regexp.QuoteMeta(t) + `=\(([^\)]+)\)`)
		m = r.FindStringSubmatch(text)[2]
	}

	// we need to find quots' numbers to see if it is a single sentence
	quot_r := regexp.MustCompile(`(\s+)?('|")(.*?)('|")(\s+)?`)
	colon_r := regexp.MustCompile(`(.*):\s+(.*)`)
	quot_n := len(quot_r.FindAllStringSubmatch(m, -1))
	// return the plain sentence with beginning and ending quots stripped directly
	if quot_n == 1 {
		var res string
		a := []rune(m)
		for i, j := range a {
			if i > 0 && i < len(a)-1 {
				res += string(j)
			}
		}

		// if the sentence is the only value for optdepends, and contains ":",
		// we should return type Optdepends
		if t == "optdepends" && colon_r.MatchString(res) {
			matched := colon_r.FindStringSubmatch(res)
			return Optdepends{matched[1], matched[2]}
		}

		return res
	}

	// now deal with the multi quots condition
	if quot_n > 1 {
		matched := quot_r.FindAllStringSubmatch(m, -1)
		if colon_r.MatchString(matched[0][3]) {
			var res []Optdepends
			for _, i := range matched {
				m := colon_r.FindStringSubmatch(i[3])
				res = append(res, Optdepends{m[1], m[2]})
			}
			return res
		} else {
			var res []string
			for _, i := range matched {
				res = append(res, i[3])
			}
			return res
		}
	}

	// now deal with the multi whitespace condition
	whitespace_r := regexp.MustCompile(`\s+`)
	if whitespace_r.MatchString(m) {
		return strings.Split(m, " ")
	}

	return m
}

func matchPreamble(s string) string {
	if len(s) == 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	preamble := ""
	for _, v := range lines {
		if strings.HasPrefix(v, "#") {
			preamble += v + "\n"
		}
	}
	return preamble
}

func matchMacros(s string) []string {

}

func matchFunc(t, s string) string {
	// return reasonable value if zero-length text provided
	if len(s) == 0 {
		return ""
	}
	return filter(t, s, true)
}

func main() {
	f, e := ioutil.ReadFile("nm.pkgbuild")
	errChk(e)
	raw := string(f)
	pkg := parse(raw)
	fmt.Println(pkg)
}
