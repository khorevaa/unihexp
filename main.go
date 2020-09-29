package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"github.com/apex/log"
	cli "github.com/jawher/mow.cli"
	"github.com/jinzhu/configor"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
	builtBy = ""
)

type file struct {
	File            string
	DoBackup        bool `default:"true"`
	ContinueOnError bool `default:"false"`
	Patches         []patch

	Old string
	New string
}

type patch struct {
	Old string
	New string
}

func (p patch) Patch(in []byte) (out []byte, err error) {

	patchBytes, err := hex.DecodeString(p.New)

	if err != nil {
		return
	}

	srcBytes, err := hex.DecodeString(p.Old)

	if err != nil {
		return
	}

	if !bytes.Contains(in, srcBytes) {
		return out, errors.New("not found src hex string")
	}

	out = bytes.ReplaceAll(in, srcBytes, patchBytes)
	return
}

func (f file) Patch() error {

	patches := f.Patches

	if len(f.New) > 0 && len(f.Old) > 0 {

		patches = append(patches, patch{
			Old: f.Old,
			New: f.New,
		})

	}

	if len(patches) == 0 {
		return errors.New("nothing patch in file")
	}

	files, err := filepath.Glob(f.File)

	if err != nil {
		return err
	}

	for _, fl := range files {

		log.Infof("Patch file <%s>", fl)

		var errPatch error

		if f.DoBackup && !checkFile {
			errPatch = doBackup(fl)
		}

		if errPatch != nil {
			log.Error(errPatch.Error())
			continue
		}

		fileBytes, errPatch := ioutil.ReadFile(fl)
		if errPatch != nil {
			log.Error(errPatch.Error())
			continue
		}

		for _, p := range patches {

			fileBytes, errPatch = p.Patch(fileBytes)

			if errPatch != nil && !f.ContinueOnError {
				break
			}

		}

		if errPatch != nil && !f.ContinueOnError {
			log.Error(errPatch.Error())
			continue
		}

		if checkFile {
			log.Infof("Patch file <%s> can be successful", fl)
			continue
		}

		err = ioutil.WriteFile(fl, fileBytes, 0777)
		if err != nil {
			log.Error(err.Error())
		} else {
			log.Infof("Patch file <%s> successful", fl)
		}

	}

	return nil

}

var Config = struct {
	File            string
	DoBackup        bool `default:"true"`
	ContinueOnError bool `default:"false"`
	Old             string
	New             string

	Files   []file
	Patches []patch
}{}

var checkFile bool

func main() {

	App := cli.App("unihexp", "Universal HEX Patch")
	App.Version("v version", buildVersion())
	App.ErrorHandling = flag.ExitOnError
	var config string

	App.StringOptPtr(&Config.Old, "src", "", "src hex string to replace")
	App.StringOptPtr(&Config.New, "hex", "", "new hex string to replace")
	App.StringOptPtr(&Config.File, "f file", "", "path to file")
	App.BoolOptPtr(&checkFile, "check", false, "check file to patch")
	App.BoolOptPtr(&Config.DoBackup, "b backup", true, "do backup patching file")
	App.BoolOptPtr(&Config.ContinueOnError, "continue-on-error", false, "continue patch on error")
	App.StringOptPtr(&config, "c config", "", "config file to patch")

	App.Before = func() {

		if len(config) > 0 {
			err := configor.Load(&Config, config)

			if err != nil {
				failOnErr(err)
			}

		} else {
			_ = configor.Load(&Config)
		}

		if len(Config.File) > 0 && len(Config.Old) > 0 {

			Config.Files = append(Config.Files, file{
				File:            Config.File,
				DoBackup:        Config.DoBackup,
				ContinueOnError: Config.ContinueOnError,
				Old:             Config.Old,
				New:             Config.New,
			})
		}

		if len(Config.File) > 0 && len(Config.Patches) > 0 {

			Config.Files = append(Config.Files, file{
				File:            Config.File,
				DoBackup:        Config.DoBackup,
				ContinueOnError: Config.ContinueOnError,
				Patches:         Config.Patches,
			})
		}

	}

	App.Action = func() {

		var err error

		if len(Config.Files) == 0 {
			err = errors.New("nothing to patch")
			failOnErr(err)
		}

		err = doPatch(Config.Files...)

	}

	_ = App.Run(os.Args)

}

func doPatch(files ...file) error {

	for _, file := range files {

		err := file.Patch()
		if err != nil {
			log.Errorf("error patch file <%s>, hex <%s>", file.File, file.Old)
		}
	}
	return nil
}

func doBackup(file string) error {

	src := filepath.Clean(file)
	dst := filepath.Join(filepath.Dir(src), fmt.Sprintf("%s.bak", filepath.Base(src)))

	return CopyFile(src, dst)

}

func failOnErr(err error) {
	if err != nil {
		log.Errorf("runtime error: %v \n", err.Error())
		cli.Exit(1)
	}
}

func buildVersion() string {
	var result = version
	if commit != "" {
		result = fmt.Sprintf("%s\ncommit: %s", result, commit)
	}
	if date != "" {
		result = fmt.Sprintf("%s\nbuilt at: %s", result, date)
	}
	if builtBy != "" {
		result = fmt.Sprintf("%s\nbuilt by: %s", result, builtBy)
	}
	return result
}

func CopyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return
	}

	err = out.Sync()
	if err != nil {
		return
	}

	si, err := os.Stat(src)
	if err != nil {
		return
	}
	err = os.Chmod(dst, si.Mode())
	if err != nil {
		return
	}

	return
}

func Exists(name string) (bool, error) {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false, err
	}
	return true, nil
}
func IsNoExist(name string) (bool, error) {

	ok, err := Exists(name)
	return !ok, err
}
