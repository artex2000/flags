package main

import (
        "os"
        "io"
        "fmt"
        "log"
        "bufio"
        "bytes"
        "strings"
        "flag"
        "sort"
        "regexp"
        "path/filepath"
)

var (
        target      string
        toolchain   string
        arch        string
        command     string
        attribute   string
    )


func init () {
    flag.StringVar(&target,    "t", "*", "set target")
    flag.StringVar(&toolchain, "n", "*", "set toolchain")
    flag.StringVar(&arch,      "a", "*", "set arch")
    flag.StringVar(&command,   "c", "*", "set command")
    flag.StringVar(&attribute, "r", "*", "set attribute")
}

func Usage() {
        fmt.Printf("Usage:\n\tflags.exe [flags] filename\n")
        flag.PrintDefaults()
}

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
        target    := (t.Target == "*")    || (p.Target == t.Target)
        toolchain := (t.Toolchain == "*") || (p.Toolchain == t.Toolchain)
        arch      := (t.Arch == "*")      || (p.Arch == t.Arch)
        comm      := (t.Command == "*")   || (p.Command == t.Command)
        attr      := (t.Attribute == "*") || (p.Attribute == t.Attribute)

        return target && toolchain && arch && comm && attr
}

func (p Property) FullName() string {
        return (p.Target + "_" +
                p.Toolchain + "_" +
                p.Arch + "_" +
                p.Command + "_" +
                p.Attribute)
}

func (p Property) IsStar() bool {
        return (p.Target    == "*" &&
                p.Toolchain == "*" &&
                p.Arch      == "*" &&
                p.Command   == "*")
}

func main() {
        flag.Parse()

        if flag.NArg() == 0 {
                Usage()
                os.Exit(1)
        }

        file := flag.Arg(0)
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

        err = export_flags(prop)
        fatal(err)

        /*
        tmp := Property{ Target : target,
                         Toolchain : toolchain, 
                         Arch : arch, 
                         Command : command,
                         Attribute: attribute }

        var buf bytes.Buffer
        fil := filter(prop, tmp) 
        if len(fil) != 0 {
                print_props(fil, true, &buf)
                out := buf.String()
                fmt.Println(out)
        } else {
                fmt.Printf("No matches for %v\n", tmp)
        }
        */
}

func export_flags(p []Property) error {
        var out bytes.Buffer

        tmp := Property { Target    : target,
                          Toolchain : toolchain,
                          Arch      : arch,
                          Command   : command,
                          Attribute : "FLAGS" }

        command := [] string {
                "CC",
                "ASM",
                "DLINK",
                "DLINK2",
                "PP",
                "VFRPP",
                "ASLPP",
                "APP",
                "ASLCC",
                "ASLDLINK",
                "RC",
                "NASM",
                "*",
        }

        arch := []string { "IA32", "X64", "AARCH64" }

        cnt := 0
        fl, _ := filter(p, tmp)
        fmt.Printf("%d properties filtered\n", len(fl))

        for _, a := range arch {
                var r []Property
                tmp.Arch = a
                for _, c := range command {
                        tmp.Command = c
                        r, fl = filter(fl, tmp)
                        cnt += len(r)
                        print_props(r, true, &out)
                }
        }

        fmt.Printf("%d properties outed\n", cnt)

        var name string
        if tmp.Toolchain != "*" {
                name = tmp.Toolchain + "_flags.txt"
        } else {
                name = "AMI" + "_flags.txt"
        }
        fname, err := filepath.Abs(name)
        if err != nil {
                return err
        }

        f, err := os.Create(fname)
        if err != nil {
                return err
        }
        defer f.Close()

        _, err = f.WriteString(out.String())
        if err != nil {
                return err
        }
        return nil
}



func print_props(p []Property, pretty bool, w io.Writer) {
        var stream io.Writer
        if w == nil {
                stream = os.Stdout
        } else {
                stream = w
        }

        for _, v := range p {
                fmt.Fprintf(stream, "%s = ", v.FullName())
                if len(v.Value) < 50 || !pretty{
                        fmt.Fprintf(stream, "%s\n", v.Value)
                } else {
                        fmt.Fprintf(stream, "\n")
                        lst := split_flags(v.Value)
                        for _, it := range lst {
                                fmt.Fprintf(stream, "\t%s\n", it)
                        }
                }
        }
}

func split_flags(f string) []string {
        var r []string
        //remove extra spaces between words
        sp := regexp.MustCompile(`\s+`)
        s := sp.ReplaceAllString(f, " ")

        tmp := strings.Split(s, " ")
        i := 1
        for i <= len(tmp) {
                a := tmp[i-1]
                if i < len(tmp) {
                        b := tmp[i]
                        if b[0] != '-' && b[0] != '/' {
                                a = a + " " + b
                                i += 1
                        }
                }
                r = append(r, a)
                i += 1
        }

        sort.Strings(r)
        return r
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

func filter(pr []Property, t Property) ([]Property, []Property) {
        var need []Property
        var rest []Property
        for _, v := range pr {
                if v.IsMatch(t) {
                        need = append(need, v)
                } else {
                        rest = append(rest, v)
                }
        }
        return need, rest
}

func fatal(err error, msg ...string) {
        if err != nil {
                log.Fatalf("Error: %v - %v\n", msg, err)
        }
}
