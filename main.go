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

type patch struct {
	File     string
	DoBackup bool `default:"true"`
	SrcHex   string
	NewHex   string
}

var Config = struct {
	patch
	PatchFiles []patch
}{}

func main() {

	App := cli.App("unihexp", "Universal HEX Patch")
	App.Version("v version", buildVersion())
	App.ErrorHandling = flag.ExitOnError
	var config string

	App.StringOptPtr(&Config.SrcHex, "src", "", "src hex string to replace")
	App.StringOptPtr(&Config.NewHex, "hex", "", "new hex string to replace")
	App.StringOptPtr(&Config.File, "f file", "", "path to file")
	App.BoolOptPtr(&Config.DoBackup, "b backup", true, "do backup patching file")
	App.StringOptPtr(&config, "c config", "", "config file to patch")

	App.Before = func() {

		if len(config) > 0 {
			_ = configor.Load(&Config, config)
		} else {
			_ = configor.Load(&Config)
		}

		if len(Config.File) > 0 && len(Config.SrcHex) > 0 {

			Config.PatchFiles = append(Config.PatchFiles, patch{
				File:     Config.File,
				DoBackup: Config.DoBackup,
				SrcHex:   Config.SrcHex,
				NewHex:   Config.NewHex,
			})
		}

	}

	App.Action = func() {

		var err error

		if len(Config.PatchFiles) == 0 {
			err = errors.New("nothing to path")
			failOnErr(err)
		}

		err = doPatch(Config.PatchFiles...)

	}

	_ = App.Run(os.Args)

}

func doPatch(files ...patch) error {

	for _, file := range files {

		err := pathFile(file.File, file.SrcHex, file.NewHex, file.DoBackup)
		if err != nil {
			log.Errorf("error patch file <%s>, hex <%s>", file.File, file.SrcHex)
		}
	}
	return nil
}

func pathFile(file string, src, patch string, backup bool) error {

	log.Infof("Patch file <%s>", file)

	var err error

	if backup {
		err = doBackup(file)
		if err != nil {
			return err
		}
	}

	if ok, _ := IsNoExist(file); ok {
		return errors.New("file not exist")
	}

	patchBytes, err := hex.DecodeString(patch)

	if err != nil {
		return err
	}

	srcBytes, err := hex.DecodeString(src)

	if err != nil {
		return err
	}

	fileBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	if bytes.Contains(fileBytes, srcBytes) {
		return errors.New("not found src hex string")
	}

	newBytes := bytes.ReplaceAll(fileBytes, srcBytes, patchBytes)

	err = ioutil.WriteFile(file, newBytes, 0777)

	if err != nil {
		log.Infof("Patch file <%s> successful", file)

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
