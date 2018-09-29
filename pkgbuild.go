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

type pkgbuild struct {
  pkgname interface{}
}

func errChk(e error) {
  if e != nil {
    panic(e)
  }
}

func parse(s string) pkgbuild {
  _, main := splitSubPkgStr(s)
  return pkgbuild{matchPkgname(main)} 
}

func splitSubPkgStr(s string) (sub interface{}, main string) {
  r := regexp.MustCompile(`(?m)^package_.*?\(\) {\n[^}]+}`)
  if r.MatchString(s) {
    m := r.FindAllStringSubmatch(s, -1)
    var sub []string
    for _, i := range m { sub = append(sub, i[0]) }
    var main string
    main = s
    for _, j := range sub {
      main = strings.Replace(main, j, "", -1)
    }
    return sub, main
  }
  return "",s
}

func matchPkgname(s string) interface{} {
  if len(s) == 0 { return "" }
  r := regexp.MustCompile(`(?m)^pkgname=(\()?(\'|\")?(.*?)(\'|\")?(\))?\n`)
  m := r.FindStringSubmatch(s)[3]
  qr := regexp.MustCompile(`('|")`)
  if qr.MatchString(m) {
    q := qr.FindString(m)
    // further split quot marks
    ss := regexp.MustCompile(`(`+regexp.QuoteMeta(q)+`(\s+)?)+`).Split(m, -1)
    return ss
  }
  return m
}

func main() {
  f, e := ioutil.ReadFile("PKGBUILD-split.proto.txt")
  errChk(e)
  raw := string(f)
  pkg := parse(raw)
  fmt.Println(pkg)
}
