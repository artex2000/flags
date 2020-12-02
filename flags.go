package main

import (
        "os"
        "fmt"
        "log"
        "bufio"
        "strings"
        "regexp"
        "path/filepath"
)

type Property struct {
        Target    string
        Toolchain string
        Arch      string
        Command   string
        Attribute string
        Value     string
}

type Define struct {
        Name  string
        Value string
}

func (p Property) IsMatch(t Property) bool {
        target := (t.Target == "") || (t.Target == "*") ||
                  (p.Target == "*") || (p.Target == t.Target)
        toolchain := (t.Toolchain == "") || (t.Toolchain == "*") ||
                  (p.Toolchain == "*") || (p.Toolchain == t.Toolchain)
        arch := (t.Arch == "") || (t.Arch == "*") ||
                  (p.Arch == "*") || (p.Arch == t.Arch)
        comm := (t.Command == "") || (t.Command == "*") ||
                  (p.Command == "*") || (p.Command == t.Command)
        return target && toolchain && arch && comm
}

func (p Property) FullName() string {
        return (p.Target + "_" +
                p.Toolchain + "_" +
                p.Arch + "_" +
                p.Command + "_" +
                p.Attribute)
}

func (p Property) IsStar() bool {
        return (p.Target == "*" &&
                p.Toolchain == "*" &&
                p.Arch == "*" &&
                p.Command == "*")
}

func main() {
        if len(os.Args) < 2 {
                fmt.Printf("Usage: %v filename\n", filepath.Base(os.Args[0]))
                os.Exit(1)
        }

        file := os.Args[1]
        if ok := filepath.IsAbs(file); !ok {
                var err error
                file, err = filepath.Abs(file)
                fatal(err, "Can't get absolute path")
        }
        f, err := os.Open(file)
        fatal(err, "Can't read file")
        defer f.Close()

        sc := bufio.NewScanner(f)
        var df []Define
        var pr []Property
        for sc.Scan() {
                l := sc.Text()
                if l == "" || l[0] == '#' {
                        continue
                }

                if strings.HasPrefix(l, "DEFINE ") {
                        l = strings.TrimPrefix(l, "DEFINE ")
                        p := strings.SplitN(l, "=", 2)
                        k := strings.TrimSpace(p[0])
                        v := strings.TrimSpace(p[1])
                        df = append(df, Define{k, v})
                } else {
                        p := strings.SplitN(l, "=", 2)
                        k := strings.TrimSpace(p[0])
                        fl := strings.Split(k, "_")
                        if len(fl) != 5 {
                                fmt.Printf("Invalid property %s\n", k)
                                continue
                        }
                        v := strings.TrimSpace(p[1])
                        pr = append(pr, Property{ fl[0], fl[1], fl[2],
                                                  fl[3], fl[4], v }) 
                }
        }
        defs, err := unwrap_defs(df)
        fatal(err)
        prop, err := unwrap_prop(pr, defs)
        fatal(err)
        fmt.Printf("Success, %d records\n", len(prop))

        tmp := Property{ Toolchain : "GCC5", Arch : "AARCH64", Command : "DLINK" }
        fil := filter(prop, tmp, false) 
        if len(fil) != 0 {
                for _, v := range fil {
                        fmt.Printf("%s = %s\n", v.FullName(), v.Value)
                }
        } else {
                fmt.Printf("No matches for %v\n", tmp)
        }
}

func unwrap_defs(d []Define) (map[string]string, error) {
        var w []Define
        def := make(map[string]string)

        re := regexp.MustCompile(`DEF\((.*?)\)`)
        move := false //flag that new def was added
        for {
                for _, v := range d {
                        subs := re.FindAllString(v.Value, -1)
                        if len(subs) == 0 {
                                def[v.Name] = v.Value
                                move = true
                                continue
                        }

                        unresolv := false
                        for _, s := range subs {
                                k := strings.TrimPrefix(s, "DEF(")
                                k = strings.TrimSuffix(k, ")")
                                if d, ok := def[k]; !ok { //unresolved define
                                        unresolv = true
                                } else {
                                        v.Value = strings.Replace(v.Value, s, d, 1) //replace define with value
                                }
                        }
                        if unresolv {
                                w = append(w, v)
                        } else {
                                def[v.Name] = v.Value
                                move = true
                        }
                }
                if len(w) == 0 { //nothing left to process
                        return def, nil
                }
                if !move { //nothing was resolved
                        break
                }
                d = w
                w = w[:0] //reset working slice
                move = false
        }
        //if we're here there are still unresolved defines but we can't resolve them
        for _, v := range w {
                fmt.Printf("Unresolved define %v:\t%v\n", v.Name, v.Value)
        }
        return nil, fmt.Errorf("Can't resolve defines")
}

func unwrap_prop(pr []Property, df map[string]string) ([]Property, error) {
        var rs []Property
        re := regexp.MustCompile(`DEF\((.*?)\)`)
        for _, v := range pr {
                subs := re.FindAllString(v.Value, -1)
                if len(subs) == 0 {
                        rs = append(rs, v)
                        continue
                }
                for _, s := range subs {
                        k := strings.TrimPrefix(s, "DEF(")
                        k = strings.TrimSuffix(k, ")")
                        if d, ok := df[k]; !ok { //unresolved define
                                return nil, fmt.Errorf("Unresolved define %s", k)
                        } else {
                                v.Value = strings.Replace(v.Value, s, d, 1) //replace define with value
                        }
                }
                rs = append(rs, v)
        }
        return rs, nil
}

func filter(pr []Property, t Property, all bool) []Property {
        var rs []Property
        for _, v := range pr {
                if !all && v.IsStar() {
                        continue
                }
                if v.IsMatch(t) {
                        rs = append(rs, v)
                }
        }
        return rs
}

func fatal(err error, msg ...string) {
        if err != nil {
                log.Fatalf("Error: %v - %v\n", msg, err)
        }
}
