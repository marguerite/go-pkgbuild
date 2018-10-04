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

type pkgbuild struct {
	pkgname     interface{}
	pkgbase     interface{}
	pkgver      interface{}
	pkgrel      interface{}
	pkgdesc     interface{}
	url         interface{}
	arch        interface{}
	license     interface{}
	makedepends interface{}
	optdepends  interface{}
	source      interface{}
	sha256sums  interface{}
	sha512sums  interface{}
}

func errChk(e error) {
	if e != nil {
		panic(e)
	}
}

func parse(s string) pkgbuild {
	_, main := splitSubPkgStr(s)
	fmt.Println(main)
	p := pkgbuild{}
	p.pkgname = match("pkgname", main)
	p.pkgbase = match("pkgbase", main)
	p.pkgver = match("pkgver", main)
	p.pkgrel = match("pkgrel", main)
	p.pkgdesc = match("pkgdesc", main)
	p.url = match("url", main)
	p.arch = match("arch", main)
	p.license = match("license", main)
	p.makedepends = match("makedepends", main)
	p.optdepends = match("optdepends", main)
	p.source = match("source", main)
	p.sha256sums = match("sha256sums", main)
	p.sha512sums = match("sha512sums", main)
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

func findTagText(t, s string) string {
	// golang's regexp has no negative lookahead, so we have to parse string line by line
	text := ""
	begin := 0
	end := 0
	strs := strings.Split(s, "\n")
	begin_r := regexp.MustCompile(`^(\s+)?` + regexp.QuoteMeta(t) + `=`)
	end_r := regexp.MustCompile(`^(\s+)?(\s+$|\w+(=|\())`)
	// loop once to find the begin pos
	for i, j := range strs {
		if begin_r.MatchString(j) {
			begin = i
			break
		}
	}
	// early return if no begin pos (no such tag)
	if begin == 0 {
		return text
	}
	// loop again to find the end pos
	for m, n := range strs {
		if m <= begin {
			continue
		}
		if end_r.MatchString(n) {
			end = m - 1
			break
		}
	}
	// loop the third time to find the real text
	for k := begin; k <= end; k++ {
		if k == end {
			text += strs[k]
			break
		}
		text += strs[k] + "\n"
	}
	return text
}

func match(t, s string) interface{} {
	// return reasonable value if the provided text to match is empty
	if len(s) == 0 {
		return ""
	}
	text := findTagText(t, s)
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
    return strings.Split(m," ")
  }

	return m
}

func matchPkgName(s string) interface{} {
	return match("pkgname", s)
}

func main() {
	f, e := ioutil.ReadFile("nm.pkgbuild")
	errChk(e)
	raw := string(f)
	pkg := parse(raw)
	fmt.Println(pkg)
}
