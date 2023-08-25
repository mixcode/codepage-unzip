package main

/*
	codepage-unzip: A tool to unzip non-unicode zip.

	copyright github.com/mixcode
*/

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	iconv "github.com/djimenez/iconv-go"
	tty "github.com/mattn/go-tty"
)

type CmdType int

const (
	CmdNone CmdType = iota
	CmdUnzip
	CmdList
)

const (
	UTF8 = "utf-8"

	//FLAG_EFS = 0x800 // EFS: Language Encoding Flag: if set, the filename is in UTF-8
)

var (
	cmd CmdType // operation to perform

	convertFrom = UTF8 // file name conversion
	convertTo   = UTF8

	destDir = "." // output directory

	overwrite   = false
	quiet       = false
	keepFileDir = false // make a subdirectory of the zip file and put files into there
)

// show Yes/No prompt
func promptYN(msg string, defaultYes bool) bool {
	tt, err := tty.Open()
	if err != nil {
		return defaultYes
	}
	defer tt.Close()

	fmt.Print(msg)
	r, err := tt.ReadRune()
	fmt.Print("\n")
	if err == nil {
		s := strings.ToLower(string(r))
		if s == "y" {
			return true
		} else if s == "n" {
			return false
		}
	}
	return defaultYes
}

func run() (err error) {
	arg := flag.Args()
	if len(arg) == 0 {
		return fmt.Errorf("a zip filename must be given (use --help for help)")
	}

	// check the output directory
	if !overwrite {
		st, err := os.Stat(destDir)
		if os.IsNotExist(err) {
			return err
		}
		if !st.IsDir() {
			return fmt.Errorf("the destination path is not a directory")
		}
	}

	// make a zip reader
	zipname := arg[0]
	zr, err := zip.OpenReader(zipname)
	if err != nil {
		return
	}
	defer zr.Close()

	if keepFileDir { // keep-organized; append the zip file name to the output path
		// append the basename of ZIP to the output path
		_, file := filepath.Split(zipname)
		ext := filepath.Ext(file)
		basename := file[:len(file)-len(ext)]
		destDir = filepath.Join(destDir, basename)
	}

	// write files
	for _, fileEntry := range zr.File {
		// convert the filename
		cf := convertFrom
		//if fileEntry.Flags&FLAG_EFS != 0 {
		if !fileEntry.NonUTF8 { // Note that EFS flag checking is done in archive/zip package
			cf = UTF8
		}
		name := fileEntry.Name
		name, err = iconv.ConvertString(name, cf, convertTo) // Note that it's safe to store non-UTF8 bytes in Go string, because it's internally just a []byte
		if err != nil {
			err = fmt.Errorf("converting from %s to %s: %w", convertFrom, convertTo, err)
			return
		}

		switch cmd {
		case CmdList:
			fmt.Printf("%s\n", name)

		case CmdUnzip:
			err = writeFile(fileEntry, name)
			if err != nil {
				return
			}
		}
	}

	return
}

var (
	hasPath = make(map[string]bool)
)

func dbgj(e any) string {
	s, _ := json.Marshal(e)
	return string(s)
}

func writeFile(entry *zip.File, name string) (err error) {

	if name == "" {
		return fmt.Errorf("empty filename")
	}
	outpath := filepath.Join(destDir, name)

	if (name[len(name)-1] == '/' || name[len(name)-1] == '\\') && entry.UncompressedSize64 == 0 {
		// the entry is a directory
		err = os.MkdirAll(outpath, fs.ModePerm)
		return
	}

	st, err := os.Stat(outpath)
	if !os.IsNotExist(err) {
		if _, ok := err.(*fs.PathError); ok { // intermediate path error
			// try to create intermediate paths
			path := filepath.Dir(outpath)
			err = os.MkdirAll(path, fs.ModePerm)
			if err != nil {
				return
			}
			hasPath[path] = true
			st, err = os.Stat(outpath)
		}
	}
	if !os.IsNotExist(err) {
		if st.IsDir() {
			// a directory with the same name exists
			return fmt.Errorf("cannot create file %s", name)
		}
		if !overwrite {
			fmt.Printf("The output file '%s' already exists.", name)
			yes := promptYN(" Overwrite? (y/N)", false)
			if !yes {
				// ignore this file
				return nil
			}
		}
	}

	if !quiet {
		fmt.Printf("%s\n", name)
	}
	fi, err := entry.Open()
	if err != nil {
		return
	}
	defer fi.Close()

	// ensure the file path exists
	path := filepath.Dir(outpath)
	if !hasPath[path] {
		st, err = os.Stat(path)
		if os.IsNotExist(err) {
			// make the path
			err = os.MkdirAll(path, fs.ModePerm)
			if err != nil {
				return
			}
			hasPath[path] = true
		} else if st.IsDir() {
			hasPath[path] = true
		} else {
			return err
		}
	}

	fo, err := os.Create(outpath)
	if err != nil {
		return
	}
	defer fo.Close()
	sz, err := io.Copy(fo, fi)
	if err != nil {
		return
	}
	if sz != int64(entry.UncompressedSize64) {
		err = fmt.Errorf("decompressed size does not match")
	}

	return
}

func main() {

	flag.Usage = func() {
		fo := flag.CommandLine.Output()
		fmt.Fprintf(fo, "Decompress a ZIP file with non-unicode filenames.\n")
		fmt.Fprintf(fo, "\n")
		fmt.Fprintf(fo, "Usage: %s [flags] [-f codepage] ZIPfile\n", os.Args[0])
		fmt.Fprintf(fo, "\n")
		fmt.Fprintf(fo, "Filenames are converted from the specified codepage to unicode.\n")
		fmt.Fprintf(fo, "See iconv man page for avaliable codepages.\n")
		fmt.Fprintf(fo, "\n")

		fmt.Fprintf(fo, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(fo, "\n")
	}

	flagList := false
	flag.BoolVar(&flagList, "l", false, "print filenames without extracting")
	flag.StringVar(&destDir, "d", destDir, "Directory to which to extract files")
	flag.BoolVar(&overwrite, "o", overwrite, "overwrite existing files")
	flag.BoolVar(&keepFileDir, "k", keepFileDir, "keep-organized; make a subdirectory of the same name with ZIP file and put files there")
	flag.BoolVar(&quiet, "q", quiet, "suppress messages")
	flag.StringVar(&convertFrom, "f", convertFrom, "codepage of filenames in ZIP")
	flag.StringVar(&convertTo, "t", convertTo, "codepage of output filenames. WARNING: change this only if you know exactly what you are doing!")
	flag.Parse()

	if flagList {
		cmd = CmdList
	} else {
		cmd = CmdUnzip
	}

	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err.Error())
		os.Exit(1)
	}
}
