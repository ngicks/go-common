package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

func main() {
	buf, bufOut := new(bytes.Buffer), new(bytes.Buffer)

	pkgMax := 5
	for pkgIdx := range pkgMax {
		path := filepath.FromSlash("internal/callstack/pkg" + strconv.Itoa(pkgIdx))
		err := os.Mkdir(path, fs.ModePerm)
		if err != nil && !errors.Is(err, fs.ErrExist) {
			panic(err)
		}
		var fileMax = 5

		for fileIdx := range fileMax {
			func() {
				buf.Reset()

				fmt.Fprintf(
					buf,
					`package pkg%d

import (
`,
					pkgIdx,
				)
				for ii := range 5 {
					if ii == pkgIdx {
						continue
					}
					fmt.Fprintf(buf, `"github.com/ngicks/go-common/serr/internal/callstack/pkg%d"
`,
						ii,
					)
				}
				_, _ = buf.WriteString(`"github.com/ngicks/go-common/serr/internal/callstack/baseerr"
`)
				_, _ = buf.WriteString(")\n\n")

				var funcPerFile = 10
				for funcIdx := range funcPerFile {
					fmt.Fprintf(buf, "func F_%d_%d() error {\n", fileIdx, funcIdx)
					if funcIdx != funcPerFile-1 {
						fmt.Fprintf(buf, "return F_%d_%d()\n", fileIdx, funcIdx+1)
					} else if fileIdx != fileMax-1 {
						fmt.Fprintf(buf, "return F_%d_%d()\n", fileIdx+1, 0)
					} else if pkgIdx != pkgMax-1 {
						fmt.Fprintf(buf, "return pkg%d.F_0_0()\n", pkgIdx+1)
					} else {
						_, _ = buf.WriteString("return baseerr.WrapBase()\n")
					}
					_, _ = buf.WriteString("}\n\n")
				}
			}()

			cmd := exec.Command("goimports")
			cmd.Stdin = buf
			bufOut.Reset()
			cmd.Stdout = bufOut
			err := cmd.Run()
			if err != nil {
				panic(err)
			}

			fileName := filepath.Join(path, "a"+strconv.Itoa(fileIdx)+".go")
			err = os.WriteFile(fileName, bufOut.Bytes(), fs.ModePerm)
			if err != nil {
				panic(err)
			}
		}
	}
}
